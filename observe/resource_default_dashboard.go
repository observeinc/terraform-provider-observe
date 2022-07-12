package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
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
				ValidateDiagFunc: validateOID(observe.TypeDataset),
				DiffSuppressFunc: diffSuppressOIDVersion,
			},
			"dashboard": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeDashboard),
			},
		},
	}
}

func resourceDefaultDashboardSet(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	dsid, _ := observe.NewOID(data.Get("dataset").(string))
	dashid, _ := observe.NewOID(data.Get("dashboard").(string))

	err := client.SetDefaultDashboard(ctx, dsid.ID, dashid.ID)
	if err != nil {
		return diag.Errorf("failed to set default dashboard: %s", err.Error())
	}

	data.SetId(dsid.ID)

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

func defaultDashboardToResourceData(dsidRaw string, dashid *observe.OID, data *schema.ResourceData) (diags diag.Diagnostics) {
	dsid := observe.OID{
		ID:   dsidRaw,
		Type: observe.TypeDataset,
	}
	if err := data.Set("dataset", dsid.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if dashid != nil {
		if err := data.Set("dashboard", dashid.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}

func resourceDefaultDashboardDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDefaultDashboard(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete default dashboard: %s", err.Error())
	}
	return diags
}
