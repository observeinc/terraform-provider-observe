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
)

const (
	schemaRbacStatementResourceNameDescription    = "RbacStatement name. Must be unique per account."
	schemaRbacStatementObjectIdDescription        = "The Observe ID for an object."
	schemaRbacStatementObjectFolderDescription    = "The Observe ID for a folder."
	schemaRbacStatementObjectWorkspaceDescription = "The Observe ID for a workspace."
	schemaRbacStatementObjectTypeDescription      = "The type of object such as dataset."
	schemaRbacStatementObjectNameDescription      = "The name of object. Can be provided along with `type`."
	schemaRbacStatementObjectOwnerDescription     = "True to bind to objects owned by the user. Can be provided along with `type`."

	schemaRbacStatementSubjectUserDescription  = "OID of a user."
	schemaRbacStatementSubjectGroupDescription = "OID of a RBAC Group."
)

var rbacStatementObjectTypes = []string{
	"object.0.id",
	"object.0.folder",
	"object.0.workspace",
	"object.0.type",
	"object.0.all",
}

func resourceRbacStatement() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a RBAC Statement.",
		CreateContext: resourceRbacStatementCreate,
		UpdateContext: resourceRbacStatementUpdate,
		ReadContext:   resourceRbacStatementRead,
		DeleteContext: resourceRbacStatementDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"subject": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user": {
							Type:             schema.TypeString,
							ExactlyOneOf:     []string{"subject.0.user", "subject.0.group", "subject.0.all"},
							Optional:         true,
							ValidateDiagFunc: validateOID(oid.TypeUser),
							Description:      schemaRbacStatementSubjectUserDescription,
						},
						"group": {
							Type:             schema.TypeString,
							ExactlyOneOf:     []string{"subject.0.user", "subject.0.group", "subject.0.all"},
							Optional:         true,
							ValidateDiagFunc: validateOID(oid.TypeRbacGroup),
							Description:      schemaRbacStatementSubjectGroupDescription,
						},
						"all": {
							Type:         schema.TypeBool,
							ExactlyOneOf: []string{"subject.0.user", "subject.0.group", "subject.0.all"},
							Optional:     true,
							Default:      false,
						},
					},
				},
			},
			"object": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							ExactlyOneOf: rbacStatementObjectTypes,
							Optional:     true,
							Description:  schemaRbacStatementObjectIdDescription,
						},
						"folder": {
							Type:         schema.TypeString,
							ExactlyOneOf: rbacStatementObjectTypes,
							Optional:     true,
							Description:  schemaRbacStatementObjectFolderDescription,
						},
						"workspace": {
							Type:         schema.TypeString,
							ExactlyOneOf: rbacStatementObjectTypes,
							Optional:     true,
							Description:  schemaRbacStatementObjectWorkspaceDescription,
						},
						"type": {
							Type:         schema.TypeString,
							ExactlyOneOf: rbacStatementObjectTypes,
							Optional:     true,
							Description:  schemaRbacStatementObjectTypeDescription,
						},
						"name": {
							Type:         schema.TypeString,
							RequiredWith: []string{"object.0.type"},
							Optional:     true,
							Description:  schemaRbacStatementObjectNameDescription,
						},
						"owner": {
							Type:         schema.TypeBool,
							RequiredWith: []string{"object.0.type"},
							Optional:     true,
							Default:      false,
							Description:  schemaRbacStatementObjectOwnerDescription,
						},
						"all": {
							Type:         schema.TypeBool,
							ExactlyOneOf: rbacStatementObjectTypes,
							Optional:     true,
							Default:      false,
						},
					},
				},
			},
			"role": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(gql.AllRbacRoles),
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func newRbacStatementConfig(data *schema.ResourceData) (input *gql.RbacStatementInput, diags diag.Diagnostics) {
	input = &gql.RbacStatementInput{}

	if v, ok := data.GetOk("description"); ok {
		input.Description = v.(string)
	}

	subject, err := newRbacSubjectInput(data)
	if err != nil {
		return nil, diag.Errorf(err.Error())
	}
	input.Subject = subject

	object, err := newRbacObjectInput(data)
	if err != nil {
		return nil, diag.Errorf(err.Error())
	}
	input.Object = object

	if v, ok := data.GetOk("role"); ok {
		input.Role = gql.RbacRole(v.(string))
	}
	return
}

func newRbacSubjectInput(data *schema.ResourceData) (gql.RbacSubjectInput, error) {
	subject := gql.RbacSubjectInput{}
	if v, ok := data.GetOk("subject.0.user"); ok {
		subUser, _ := oid.NewOID(v.(string))
		uid, err := types.StringToUserIdScalar(subUser.Id)
		if err != nil {
			return subject, fmt.Errorf("error parsing subject user: %s", err.Error())
		}
		subject.UserId = &uid
	}
	if v, ok := data.GetOk("subject.0.group"); ok {
		subGroup, _ := oid.NewOID(v.(string))
		subject.GroupId = &subGroup.Id
	}
	subject.All = boolPtr(data.Get("subject.0.all").(bool))
	return subject, nil
}

func newRbacObjectInput(data *schema.ResourceData) (gql.RbacObjectInput, error) {
	object := gql.RbacObjectInput{}
	if v, ok := data.GetOk("object.0.id"); ok {
		object.ObjectId = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("object.0.folder"); ok {
		object.FolderId = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("object.0.workspace"); ok {
		object.WorkspaceId = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("object.0.type"); ok {
		object.Type = stringPtr(v.(string))
		if oname, ok := data.GetOk("object.0.name"); ok {
			object.Name = stringPtr(oname.(string))
		}
	}
	object.Owner = boolPtr(data.Get("object.0.owner").(bool))
	object.All = boolPtr(data.Get("object.0.all").(bool))
	return object, nil
}

func resourceRbacStatementCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newRbacStatementConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateRbacStatement(ctx, config)
	if err != nil {
		return diag.Errorf("failed to create rbacstatement: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceRbacStatementRead(ctx, data, meta)...)
}

func resourceRbacStatementUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newRbacStatementConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateRbacStatement(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update rbacstatement: %s", err.Error())
	}
	return append(diags, resourceRbacStatementRead(ctx, data, meta)...)
}

func resourceRbacStatementRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	stmt, err := client.GetRbacStatement(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read rbacstatement: %s", err.Error())
	}
	return rbacStatementToResourceData(stmt, data)
}

func resourceRbacStatementDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteRbacStatement(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete rbacstatement: %s", err.Error())
	}
	return diags
}

func rbacStatementToResourceData(r *gql.RbacStatement, data *schema.ResourceData) (diags diag.Diagnostics) {
	data.SetId(r.Id)

	if err := data.Set("description", r.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// subject
	subject := make(map[string]interface{}, 0)
	if r.Subject.UserId != nil {
		subject["user"] = oid.UserOid(*r.Subject.UserId).String()
	} else if r.Subject.GroupId != nil {
		subject["group"] = oid.RbacGroupOid(*r.Subject.GroupId).String()
	} else if r.Subject.All != nil {
		subject["all"] = *r.Subject.All
	}
	if err := data.Set("subject", []interface{}{subject}); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// object
	object := make(map[string]interface{}, 0)
	if r.Object.ObjectId != nil {
		object["id"] = *r.Object.ObjectId
	} else if r.Object.FolderId != nil {
		object["folder"] = *r.Object.FolderId
	} else if r.Object.WorkspaceId != nil {
		object["workspace"] = *r.Object.WorkspaceId
	} else if r.Object.Type != nil {
		object["type"] = *r.Object.Type
		if r.Object.Name != nil {
			object["name"] = *r.Object.Name
		}
		if r.Object.Owner != nil {
			object["owner"] = *r.Object.Owner
		}
	} else if r.Object.All != nil {
		object["all"] = *r.Object.All
	}
	if err := data.Set("object", []interface{}{object}); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// role
	if err := data.Set("role", string(r.Role)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", r.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}
