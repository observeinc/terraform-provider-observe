package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

const (
	schemaRbacGroupResourceNameDescription = "RbacGroup name. Must be unique per account."
)

func resourceRbacGroup() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a RBAC group.",
		CreateContext: resourceRbacGroupCreate,
		UpdateContext: resourceRbacGroupUpdate,
		ReadContext:   resourceRbacGroupRead,
		DeleteContext: resourceRbacGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaRbacGroupResourceNameDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaRbacGroupDescriptionDescription,
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaRbacGroupOIDDescription,
			},
		},
	}
}

func newRbacGroupConfig(data *schema.ResourceData) (input *gql.RbacGroupInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	input = &gql.RbacGroupInput{
		Name: name,
	}
	if v, ok := data.GetOk("description"); ok {
		input.Description = v.(string)
	}
	return
}

func resourceRbacGroupCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newRbacGroupConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateRbacGroup(ctx, config)
	if err != nil {
		return diag.Errorf("failed to create rbacgroup: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceRbacGroupRead(ctx, data, meta)...)
}

func resourceRbacGroupUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newRbacGroupConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateRbacGroup(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update rbacgroup: %s", err.Error())
	}
	return append(diags, resourceRbacGroupRead(ctx, data, meta)...)
}

func resourceRbacGroupRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	group, err := client.GetRbacGroup(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read rbacgroup: %s", err.Error())
	}
	return rbacGroupToResourceData(group, data)
}

func resourceRbacGroupDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteRbacGroup(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete rbacgroup: %s", err.Error())
	}
	return diags
}
