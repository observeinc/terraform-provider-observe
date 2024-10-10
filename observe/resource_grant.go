package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceGrant() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("grant", "description"),
		CreateContext: resourceGrantCreate,
		UpdateContext: resourceGrantUpdate,
		ReadContext:   resourceGrantRead,
		DeleteContext: resourceGrantDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"subject": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeUser, oid.TypeRbacGroup),
				Description:      descriptions.Get("grant", "schema", "subject"),
			},
			"role": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(validGrantRoles),
				Description:      descriptions.Get("grant", "schema", "role"),
			},
			"qualifier": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"oid": {
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validateOID(validRbacV2Types...),
							DiffSuppressFunc: diffSuppressOIDVersion,
							Description:      descriptions.Get("grant", "schema", "qualifier", "oid"),
						},
						// in the future, will contain other qualifiers such as "tags"
					},
				},
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// for now, translates grants into rbac v1 statements until api support is added
func newGrantInput(data *schema.ResourceData) (input *gql.RbacStatementInput, diags diag.Diagnostics) {
	input = &gql.RbacStatementInput{
		Version: intPtr(2),
	}

	// subject
	subject, err := oid.NewOID(data.Get("subject").(string))
	if err != nil {
		return nil, diag.Errorf("error parsing subject: %s", err.Error())
	}
	input.Subject.All = boolPtr(false)
	if subject.Type == oid.TypeUser {
		uid, err := types.StringToUserIdScalar(subject.Id)
		if err != nil {
			return nil, diag.Errorf("error parsing subject user: %s", err.Error())
		}
		input.Subject.UserId = &uid
	} else if subject.Type == oid.TypeRbacGroup {
		if subject.Id == "1" {
			input.Subject.All = boolPtr(true)
		} else {
			input.Subject.GroupId = &subject.Id
		}
	}

	// role
	role := GrantRole(toCamel(data.Get("role").(string)))
	input.Role, err = role.ToRbacRole()
	if err != nil {
		return nil, diag.Errorf(err.Error())
	}

	// object
	var resourceId *string
	resourceOidStr, ok := data.GetOk("qualifier.0.oid")
	if ok {
		resourceOid, err := oid.NewOID(resourceOidStr.(string))
		if err != nil {
			return nil, diag.Errorf("error parsing qualifier.oid: %s", err.Error())
		}
		resourceId = &resourceOid.Id
	}
	input.Object, err = role.ToRbacObject(resourceId)
	if err != nil {
		return nil, diag.Errorf(err.Error())
	}

	return input, diags
}

// for now, receives an rbac v1 statement and translates it into a grant until api support is added
func grantToResourceData(stmt *gql.RbacStatement, data *schema.ResourceData) (diags diag.Diagnostics) {
	data.SetId(stmt.Id)

	if stmt.Version == nil || *stmt.Version != 2 {
		diags = append(diags, diag.Errorf("observe_grant only works with rbac v2 statements")...)
	}

	// subject
	subject := ""
	if stmt.Subject.All != nil && *stmt.Subject.All {
		subject = oid.RbacGroupOid("1").String()
	} else if stmt.Subject.UserId != nil {
		subject = oid.UserOid(*stmt.Subject.UserId).String()
	} else if stmt.Subject.GroupId != nil {
		subject = oid.RbacGroupOid(*stmt.Subject.GroupId).String()
	} else {
		diags = append(diags, diag.Errorf("invalid subject")...)
	}
	if err := data.Set("subject", subject); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// role and qualifier
	var role string
	qualifier := make(map[string]interface{}, 0)
	if stmt.Role == gql.RbacRoleManager && stmt.Object.All != nil && *stmt.Object.All {
		role = toSnake(string(Administrator))
	} else if stmt.Object.Type != nil {
		objType := oid.Type(*stmt.Object.Type)
		if !sliceContains(validRbacV2Types, objType) {
			diags = append(diags, diag.Errorf("invalid object type for v2 statment: %s", objType)...)
		}

		if stmt.Object.ObjectId != nil {
			resourceOid := oid.OID{Type: objType, Id: *stmt.Object.ObjectId}
			qualifier["oid"] = resourceOid.String()

			if stmt.Role == gql.RbacRoleViewer {
				if grantRole, ok := viewGrantRoleForType[objType]; ok {
					role = toSnake(string(grantRole))
				}
			} else if stmt.Role == gql.RbacRoleEditor {
				if grantRole, ok := editGrantRoleForType[objType]; ok {
					role = toSnake(string(grantRole))
				}
			}
		} else {
			// editor without object id is create
			if stmt.Role == gql.RbacRoleEditor {
				if grantRole, ok := createGrantRoleForType[objType]; ok {
					role = toSnake(string(grantRole))
				}
			}
		}
	}
	if role == "" {
		diags = append(diags, diag.Errorf("invalid statement")...)
	}
	if err := data.Set("role", string(role)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	// since qualifier is optional, we want it to be nil unless it actually has values
	var qualifierSlice []interface{}
	if len(qualifier) > 0 {
		qualifierSlice = []interface{}{qualifier}
	}
	if err := data.Set("qualifier", qualifierSlice); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", stmt.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func resourceGrantCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newGrantInput(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateRbacStatement(ctx, input)
	if err != nil {
		return diag.Errorf("failed to create grant: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceGrantRead(ctx, data, meta)...)
}

func resourceGrantUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newGrantInput(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateRbacStatement(ctx, data.Id(), input)
	if err != nil {
		return diag.Errorf("failed to update grant: %s", err.Error())
	}

	return append(diags, resourceGrantRead(ctx, data, meta)...)
}

func resourceGrantRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	stmt, err := client.GetRbacStatement(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read grant: %s", err.Error())
	}
	return grantToResourceData(stmt, data)
}

func resourceGrantDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteRbacStatement(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete grant: %s", err.Error())
	}
	return diags
}

type GrantRole string

const (
	Administrator      GrantRole = "Administrator"
	DashboardCreator   GrantRole = "DashboardCreator"
	DashboardEditor    GrantRole = "DashboardEditor"
	DashboardViewer    GrantRole = "DashboardViewer"
	DatasetCreator     GrantRole = "DatasetCreator"
	DatasetEditor      GrantRole = "DatasetEditor"
	DatasetViewer      GrantRole = "DatasetViewer"
	DatastreamCreator  GrantRole = "DatastreamCreator"
	DatastreamEditor   GrantRole = "DatastreamEditor"
	DatastreamViewer   GrantRole = "DatastreamViewer"
	MonitorCreator     GrantRole = "MonitorCreator"
	MonitorEditor      GrantRole = "MonitorEditor"
	MonitorViewer      GrantRole = "MonitorViewer"
	MonitorGlobalMuter GrantRole = "MonitorGlobalMuter"
	WorksheetCreator   GrantRole = "WorksheetCreator"
	WorksheetEditor    GrantRole = "WorksheetEditor"
	WorksheetViewer    GrantRole = "WorksheetViewer"
)

var validGrantRoles = []GrantRole{
	Administrator,
	DashboardCreator,
	DashboardEditor,
	DashboardViewer,
	DatasetCreator,
	DatasetEditor,
	DatasetViewer,
	DatastreamCreator,
	DatastreamEditor,
	DatastreamViewer,
	MonitorCreator,
	MonitorEditor,
	MonitorViewer,
	MonitorGlobalMuter,
	WorksheetCreator,
	WorksheetEditor,
	WorksheetViewer,
}

var createGrantRoles = []GrantRole{DashboardCreator, DatasetCreator, DatastreamCreator, MonitorCreator, WorksheetCreator}
var editGrantRoles = []GrantRole{DashboardEditor, DatasetEditor, DatastreamEditor, MonitorEditor, WorksheetEditor}
var viewGrantRoles = []GrantRole{DashboardViewer, DatasetViewer, DatastreamViewer, MonitorViewer, WorksheetViewer}

var validRbacV2Types = []oid.Type{oid.TypeDashboard, oid.TypeDataset, oid.TypeDatastream, oid.TypeMonitor, oid.TypeWorksheet}

var createGrantRoleForType = map[oid.Type]GrantRole{
	oid.TypeDashboard:  DashboardCreator,
	oid.TypeDataset:    DatasetCreator,
	oid.TypeDatastream: DatastreamCreator,
	oid.TypeMonitor:    MonitorCreator,
	oid.TypeWorksheet:  WorksheetCreator,
}

var editGrantRoleForType = map[oid.Type]GrantRole{
	oid.TypeDashboard:  DashboardEditor,
	oid.TypeDataset:    DatasetEditor,
	oid.TypeDatastream: DatastreamEditor,
	oid.TypeMonitor:    MonitorEditor,
	oid.TypeWorksheet:  WorksheetEditor,
}

var viewGrantRoleForType = map[oid.Type]GrantRole{
	oid.TypeDashboard:  DashboardViewer,
	oid.TypeDataset:    DatasetViewer,
	oid.TypeDatastream: DatastreamViewer,
	oid.TypeMonitor:    MonitorViewer,
	oid.TypeWorksheet:  WorksheetViewer,
}

func (r GrantRole) ToRbacRole() (gql.RbacRole, error) {
	if r == Administrator {
		return gql.RbacRoleManager, nil
	} else if r == MonitorGlobalMuter {
		return gql.RbacRoleMonitorglobalmute, nil
	} else if sliceContains(createGrantRoles, r) || sliceContains(editGrantRoles, r) {
		return gql.RbacRoleEditor, nil
	} else if sliceContains(viewGrantRoles, r) {
		return gql.RbacRoleViewer, nil
	} else {
		return "", fmt.Errorf("invalid role: %s", r)
	}
}

func (r GrantRole) ToType() *oid.Type {
	switch r {
	case DashboardCreator, DashboardEditor, DashboardViewer:
		return asPointer(oid.TypeDashboard)
	case DatasetCreator, DatasetEditor, DatasetViewer:
		return asPointer(oid.TypeDataset)
	case DatastreamCreator, DatastreamEditor, DatastreamViewer:
		return asPointer(oid.TypeDatastream)
	case MonitorCreator, MonitorEditor, MonitorViewer:
		return asPointer(oid.TypeMonitor)
	case WorksheetCreator, WorksheetEditor, WorksheetViewer:
		return asPointer(oid.TypeWorksheet)
	default:
		return nil
	}
}

func (r GrantRole) ToRbacObject(resourceId *string) (gql.RbacObjectInput, error) {
	objectInput := gql.RbacObjectInput{
		Owner: boolPtr(false),
		All:   boolPtr(false),
	}
	// an oid qualifier is only valid for edit roles and view roles
	isResourceRole := sliceContains(editGrantRoles, r) || sliceContains(viewGrantRoles, r)
	if isResourceRole && resourceId == nil {
		return objectInput, fmt.Errorf("role %s must be qualified with an object id", r)
	}
	if !isResourceRole && resourceId != nil {
		return objectInput, fmt.Errorf("role %s cannot be qualified with an object id", r)
	}
	switch r {
	case Administrator:
		objectInput.All = boolPtr(true)
	case MonitorGlobalMuter:
		// this grant role doesn't require anything on the statement object,
		// just setting the statement role is sufficient
	default:
		objectInput.Type = (*string)(r.ToType())
		objectInput.ObjectId = resourceId
	}
	return objectInput, nil
}
