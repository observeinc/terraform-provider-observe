package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

const (
	schemaRbacGroupmemberResourceNameDescription = "RbacGroupmember name. Must be unique per account."
)

func resourceRbacGroupmember() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a RBAC Groupmember.",
		CreateContext: resourceRbacGroupmemberCreate,
		UpdateContext: resourceRbacGroupmemberUpdate,
		ReadContext:   resourceRbacGroupmemberRead,
		DeleteContext: resourceRbacGroupmemberDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"group": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeRbacGroup),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"member": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user": {
							Type:             schema.TypeString,
							ExactlyOneOf:     []string{"member.0.user", "member.0.group"},
							Optional:         true,
							ValidateDiagFunc: validateOID(oid.TypeUser),
						},
						"group": {
							Type:             schema.TypeString,
							ExactlyOneOf:     []string{"member.0.user", "member.0.group"},
							Optional:         true,
							ValidateDiagFunc: validateOID(oid.TypeRbacGroup),
						},
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

func newRbacGroupmemberConfig(data *schema.ResourceData) (input *gql.RbacGroupmemberInput, diags diag.Diagnostics) {
	input = &gql.RbacGroupmemberInput{}

	group, _ := oid.NewOID(data.Get("group").(string))
	input.GroupId = group.Id

	if v, ok := data.GetOk("description"); ok {
		input.Description = v.(string)
	}

	if v, ok := data.GetOk("member.0.user"); ok {
		memUser, _ := oid.NewOID(v.(string))
		uid, err := types.StringToUserIdScalar(memUser.Id)
		if err != nil {
			return nil, diag.Errorf("error parsing member user: %s", err.Error())
		}
		input.MemberUserId = &uid
	}
	if v, ok := data.GetOk("member.0.group"); ok {
		memGroup, _ := oid.NewOID(v.(string))
		input.MemberGroupId = &memGroup.Id
	}
	return
}

func resourceRbacGroupmemberCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newRbacGroupmemberConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateRbacGroupmember(ctx, config)
	if err != nil {
		return diag.Errorf("failed to create rbacgroupmember: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceRbacGroupmemberRead(ctx, data, meta)...)
}

func resourceRbacGroupmemberUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newRbacGroupmemberConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateRbacGroupmember(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update rbacgroupmember: %s", err.Error())
	}
	return append(diags, resourceRbacGroupmemberRead(ctx, data, meta)...)
}

func resourceRbacGroupmemberRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	group, err := client.GetRbacGroupmember(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read rbacgroupmember: %s", err.Error())
	}
	return rbacGroupmemberToResourceData(group, data)
}

func resourceRbacGroupmemberDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteRbacGroupmember(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete rbacgroupmember: %s", err.Error())
	}
	return diags
}

func rbacGroupmemberToResourceData(r *gql.RbacGroupmember, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("group", oid.RbacGroupOid(r.GroupId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("description", r.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("oid", r.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	data.SetId(r.Id)

	member := make(map[string]interface{}, 0)
	if r.MemberUserId != nil {
		member["user"] = oid.UserOid(*r.MemberUserId).String()
	} else if r.MemberGroupId != nil {
		member["group"] = oid.RbacGroupOid(*r.MemberGroupId).String()
	}
	if err := data.Set("member", []interface{}{member}); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}
