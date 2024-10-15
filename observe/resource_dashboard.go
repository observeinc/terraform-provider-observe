package observe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/binding"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

const (
	schemaDashboardWorkspaceDescription       = "OID of workspace dashboard is contained in."
	schemaDashboardNameDescription            = "Dashboard name. Must be unique within workspace."
	schemaDashboardDescriptionDescription     = "Dashboard description."
	schemaDashboardIconDescription            = "Icon image."
	schemaDashboardJSONDescription            = "Dashboard stages in JSON format."
	schemaDashboardLayoutDescription          = "Dashboard layout in JSON format."
	schemaDashboardOIDDescription             = "The Observe ID for dashboard."
	schemaDashboardParametersDescription      = "Dashboard parameters in JSON format."
	schemaDashboardParameterValuesDescription = "Dashboard parameter values in JSON format."
)

func resourceDashboard() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages an Observe dashboard, which predefines visualizations of Observe data in a grid of cards.",
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
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      schemaDashboardWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDashboardNameDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDashboardDescriptionDescription,
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
				DiffSuppressFunc: diffSuppressStageQueryInput,
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
				DiffSuppressFunc: diffSuppressParameters,
				Description:      schemaDashboardParametersDescription,
			},
			"parameter_values": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressParameterValues,
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

func newDashboardConfig(data *schema.ResourceData) (input *gql.DashboardInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	input = &gql.DashboardInput{
		Name: &name,
	}

	{
		// always reset to empty string if description not set
		input.Description = stringPtr(data.Get("description").(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("stages"); ok {
		data := v.(string)
		if err := json.Unmarshal([]byte(data), &input.Stages); err != nil {
			diagErr := fmt.Errorf("failed to parse 'stages' request field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		}
	}

	if v, ok := data.GetOk("layout"); ok {
		input.Layout = types.JsonObject(v.(string)).Ptr()
	}

	if v, ok := data.GetOk("parameters"); ok {
		data := v.(string)
		if err := json.Unmarshal([]byte(data), &input.Parameters); err != nil {
			diagErr := fmt.Errorf("failed to parse 'parameters' request field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		}
	}

	if v, ok := data.GetOk("parameter_values"); ok {
		data := v.(string)
		if err := json.Unmarshal([]byte(data), &input.ParameterValues); err != nil {
			diagErr := fmt.Errorf("failed to parse 'parameter_values' request field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		}
	}

	return input, diags
}

func dashboardToResourceData(ctx context.Context, d *gql.Dashboard, data *schema.ResourceData,
	client *observe.Client, genBindings bool) (diags diag.Diagnostics) {
	bindFor := binding.NewKindSet(binding.KindDataset, binding.KindWorkspace)
	gen, err := binding.NewGenerator(ctx, genBindings, "dashboard", d.Name, client, bindFor)
	if err != nil {
		return diag.Errorf("Failed to initialize binding generator: %s", err.Error())
	}

	if err := data.Set("workspace", gen.TryBind(binding.KindWorkspace, oid.WorkspaceOid(d.WorkspaceId).String())); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", d.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.IconUrl != nil {
		if err := data.Set("icon_url", *d.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Stages != nil {
		// Hack hack hack hack hack
		for i, stage := range d.Stages {
			if stage.Id != nil && *stage.Id == "" {
				d.Stages[i].Id = nil
			}
			for j, input := range stage.Input {
				if input.StageId != nil && *input.StageId == "" {
					d.Stages[i].Input[j].StageId = nil
				}
			}
			if stage.Params != nil && *stage.Params == types.JsonObject("null") {
				d.Stages[i].Params = nil
			} else if stage.Params != nil && string(*stage.Params) == "" {
				d.Stages[i].Params = nil
			}
		}
		if stagesRaw, err := json.Marshal(d.Stages); err != nil {
			diagErr := fmt.Errorf("failed to parse 'stages' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if stagesRaw, err := gen.GenerateJson(stagesRaw); err != nil {
			diagErr := fmt.Errorf("failed to generate bindings for 'stages' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if err := data.Set("stages", string(stagesRaw)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Parameters != nil {
		if parametersRaw, err := json.Marshal(d.Parameters); err != nil {
			diagErr := fmt.Errorf("failed to parse 'parameters' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if parametersRaw, err := gen.GenerateJson(parametersRaw); err != nil {
			diagErr := fmt.Errorf("failed to generate bindings for 'parameters' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if err := data.Set("parameters", string(parametersRaw)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.ParameterValues != nil {
		if parameterValuesRaw, err := json.Marshal(d.ParameterValues); err != nil {
			diagErr := fmt.Errorf("failed to parse 'parameter_values' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if parameterValuesRaw, err := gen.GenerateJson(parameterValuesRaw); err != nil {
			diagErr := fmt.Errorf("failed to generate bindings for 'parameterValuesRaw' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if err := data.Set("parameter_values", string(parameterValuesRaw)); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.Layout != nil || gen.Enabled {
		if d.Layout == nil {
			empty := types.JsonObject("{}")
			d.Layout = &empty
		}
		if layout, err := d.Layout.MarshalJSON(); err != nil {
			diagErr := fmt.Errorf("failed to parse 'layout' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else if layout, err := gen.GenerateJson(layout); err != nil {
			diagErr := fmt.Errorf("failed to generate bindings for 'layout' response field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		} else {
			layoutJson := types.JsonObject(string(layout))
			if layout, err := gen.InsertBindingsObjectJson(&layoutJson); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			} else if err := data.Set("layout", layout); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}

	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
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

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateDashboard(ctx, id.Id, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dashboard",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceDashboardRead(ctx, data, meta)...)
}

func resourceDashboardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetDashboard(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve dashboard [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return dashboardToResourceData(ctx, result, data, client, false)
}

func resourceDashboardUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newDashboardConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.UpdateDashboard(ctx, data.Id(), oid.Id, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update dashboard [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return dashboardToResourceData(ctx, result, data, client, false)
}

func resourceDashboardDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDashboard(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete dashboard: %s", err)
	}
	return diags
}
