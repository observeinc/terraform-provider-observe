package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceDefaultDashboardForTag() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("default_dashboard_for_tag", "description"),
		CreateContext: resourceDefaultDashboardForTagSet,
		UpdateContext: resourceDefaultDashboardForTagSet,
		ReadContext:   resourceDefaultDashboardForTagRead,
		DeleteContext: resourceDefaultDashboardForTagDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"tag": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      descriptions.Get("default_dashboard_for_tag", "schema", "tag"),
			},
			"dashboard": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDashboard),
				Description:      descriptions.Get("default_dashboard_for_tag", "schema", "dashboard"),
			},
		},
	}
}

func resourceDefaultDashboardForTagSet(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	tag := data.Get("tag").(string)
	dashid, err := oid.NewOID(data.Get("dashboard").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.SetDefaultDashboardForTag(ctx, tag, dashid.Id)
	if err != nil {
		return diag.Errorf("failed to set default dashboard for tag: %s", err.Error())
	}

	data.SetId(tag)

	return append(diags, resourceDefaultDashboardForTagRead(ctx, data, meta)...)
}

func resourceDefaultDashboardForTagRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	tag := data.Id()
	dashid, err := client.GetDefaultDashboardForTag(ctx, tag)
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read default dashboard for tag: %s", err.Error())
	}
	if dashid == nil {
		data.SetId("")
		return nil
	}

	return defaultDashboardForTagToResourceData(tag, dashid, data)
}

func defaultDashboardForTagToResourceData(tag string, dashid *string, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("tag", tag); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	dashboard := ""
	if dashid != nil {
		dashboard = oid.DashboardOid(*dashid).String()
	}
	if err := data.Set("dashboard", dashboard); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceDefaultDashboardForTagDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.ClearDefaultDashboardForTag(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete default dashboard for tag: %s", err.Error())
	}
	return diags
}
