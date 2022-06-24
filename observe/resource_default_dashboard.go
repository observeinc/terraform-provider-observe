package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourceDefaultDashboard() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDefaultDashboardSet,
		UpdateContext: resourceDefaultDashboardSet,
		ReadContext:   resourceDefaultDashboardRead,
		DeleteContext: resourceDefaultDashboardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"dataset": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				DiffSuppressFunc: diffSuppressOIDVersion,
			},
			"dashboard": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDashboard),
			},
		},
	}
}

func resourceDefaultDashboardSet(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	dsid, _ := oid.NewOID(data.Get("dataset").(string))
	dashid, _ := oid.NewOID(data.Get("dashboard").(string))

	err := client.SetDefaultDashboard(ctx, dsid.Id, dashid.Id)
	if err != nil {
		return diag.Errorf("failed to set default dashboard: %s", err.Error())
	}

	data.SetId(dsid.Id)

	return append(diags, resourceDefaultDashboardRead(ctx, data, meta)...)
}

func resourceDefaultDashboardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	dashid, err := client.GetDefaultDashboard(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read default dashboard: %s", err.Error())
	}

	return defaultDashboardToResourceData(data.Id(), dashid, data)
}

func defaultDashboardToResourceData(dsidRaw string, dashid *string, data *schema.ResourceData) (diags diag.Diagnostics) {
	dsid := oid.DatasetOid(dsidRaw)
	if err := data.Set("dataset", dsid.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if dashid != nil {
		dashoid := oid.DashboardOid(*dashid)
		if err := data.Set("dashboard", dashoid.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

func resourceDefaultDashboardDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.ClearDefaultDashboard(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete default dashboard: %s", err.Error())
	}
	return diags
}
