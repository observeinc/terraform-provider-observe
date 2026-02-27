package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
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
		Description:        "Deprecated. Only for use with RBAC v1. For v2, use `observe_grant`.\n\nManages a RBAC Statement.",
		DeprecationMessage: "RBAC Statements are deprecated and only for use with RBAC v1. For v2, see `observe_grant`.",
		CreateContext:      resourceRbacStatementCreate,
		UpdateContext:      resourceRbacStatementUpdate,
		ReadContext:        resourceRbacStatementRead,
		DeleteContext:      resourceRbacStatementDelete,
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
				DiffSuppressFunc: diffSuppressEnums,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceRbacStatementCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return diag.Errorf("observe_rbac_statement is deprecated and can no longer be created, use observe_grant or observe_resource_grants instead")
}

func resourceRbacStatementUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return diag.Errorf("observe_rbac_statement is deprecated and can no longer be updated, delete this resource and use observe_grant or observe_resource_grants instead")
}

func resourceRbacStatementRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	// Just leave the state as is when reading so that terraform runs with old
	// observe_rbac_statement resources still work even if the statement was deleted
	// from the backend.
	return nil
}

func resourceRbacStatementDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	// Always consider deletion of the resource successful as the underlying statement
	// will be automatically deleted by the backend anyway.
	return nil
}
