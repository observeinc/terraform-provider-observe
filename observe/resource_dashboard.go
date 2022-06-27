package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

const (
	schemaDashboardWorkspaceDescription       = "OID of workspace dashboard is contained in."
	schemaDashboardNameDescription            = "Dashboard name. Must be unique within workspace."
	schemaDashboardIconDescription            = "Icon image."
	schemaDashboardJSONDescription            = "Dashboard stages in JSON format."
	schemaDashboardLayoutDescription          = "Dashboard layout in JSON format."
	schemaDashboardOIDDescription             = "The Observe ID for dashboard."
	schemaDashboardParametersDescription      = "Dashboard parameters in JSON format."
	schemaDashboardParameterValuesDescription = "Dashboard parameter values in JSON format."
)

func resourceDashboard() *schema.Resource {
	return &schema.Resource{
		Description:   "",
		CreateContext: resourceDashboardCreate,
		ReadContext:   resourceDashboardRead,
		UpdateContext: resourceDashboardUpdate,
		DeleteContext: resourceDashboardDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
				Description:      schemaDashboardWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDashboardNameDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDashboardIconDescription,
			},
			"stages": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaDashboardJSONDescription,
			},
			"layout": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaDashboardLayoutDescription,
			},
			"parameters": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaDashboardParametersDescription,
			},
			"parameter_values": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaDashboardParameterValuesDescription,
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardOIDDescription,
			},
		},
	}
}

func newDashboardConfig(data *schema.ResourceData) (config *observe.DashboardConfig, diags diag.Diagnostics) {
	config = &observe.DashboardConfig{
		Name: data.Get("name").(string),
	}

	if v, ok := data.GetOk("icon_url"); ok {
		icon := v.(string)
		config.Icon = &icon
	}

	if v, ok := data.GetOk("stages"); ok {
		data := v.(string)
		config.Stages = &data
	}

	if v, ok := data.GetOk("layout"); ok {
		data := v.(string)
		config.Layout = &data
	}

	if v, ok := data.GetOk("parameters"); ok {
		data := v.(string)
		config.Parameters = &data
	}

	if v, ok := data.GetOk("parameter_values"); ok {
		data := v.(string)
		config.ParameterValues = &data
	}

	return config, diags
}

func dashboardToResourceData(d *observe.Dashboard, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", d.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Config.Icon != nil {
		if err := data.Set("icon_url", d.Config.Icon); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.Stages != nil {
		if err := data.Set("stages", d.Config.Stages); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.Layout != nil {
		if err := data.Set("layout", d.Config.Layout); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.Parameters != nil {
		if err := data.Set("parameters", d.Config.Parameters); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Config.ParameterValues != nil {
		if err := data.Set("parameter_values", d.Config.ParameterValues); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("oid", d.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceDashboardCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDashboardConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateDashboard(ctx, oid.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dashboard",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)
	return append(diags, resourceDashboardRead(ctx, data, meta)...)
}

func resourceDashboardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetDashboard(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve dashboard [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return dashboardToResourceData(result, data)
}

func resourceDashboardUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDashboardConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.UpdateDashboard(ctx, data.Id(), oid.ID, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update dashboard [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return dashboardToResourceData(result, data)
}

func resourceDashboardDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDashboard(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete dashboard: %s", err)
	}
	return diags
}
