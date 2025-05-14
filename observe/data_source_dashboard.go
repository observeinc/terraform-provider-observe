package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/binding"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceDashboard() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches data for an existing Observe dashboard.",
		ReadContext: dataSourceDashboardRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateID(),
				Description:      "Dashboard ID.",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardOIDDescription,
			},
			"workspace": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardNameDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardDescriptionDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardIconDescription,
			},
			"stages": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardJSONDescription,
			},
			"layout": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardLayoutDescription,
			},
			"parameters": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardParametersDescription,
			},
			"parameter_values": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardParameterValuesDescription,
			},
		},
	}
}

func dataSourceDashboardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		id     = data.Get("id").(string)
	)

	dashboard, err := client.GetDashboard(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	data.SetId(dashboard.Id)

	diags = dashboardToResourceData(dashboard, data)
	if diags.HasError() {
		return diags
	}

	if client.ExportObjectBindings {
		err := generateDashboardBindings(ctx, dashboard, data, client)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

// Generates bindings for use in cross-tenant exports of dashboards. See binding.go for details.
func generateDashboardBindings(ctx context.Context, dashboard *gql.Dashboard, data *schema.ResourceData, client *observe.Client) error {
	bindFor := binding.NewKindSet(binding.KindDataset, binding.KindWorkspace)
	gen, err := binding.NewGenerator(ctx, binding.KindDashboard, dashboard.Name, client, bindFor)
	if err != nil {
		return fmt.Errorf("failed to initialize binding generator: %w", err)
	}

	// generate binding for workspace
	workspaceRef, _ := gen.TryBindOid(oid.WorkspaceOid(dashboard.WorkspaceId))
	if err := data.Set("workspace", workspaceRef); err != nil {
		return err
	}

	// generate bindings for stages, parameters, parameter_values, and layout,
	// replacing the original ids in the json data with local variable references
	for _, field := range []string{"stages", "parameters", "parameter_values", "layout"} {
		jsonWithRawIds := data.Get(field).(string)
		if jsonWithRawIds == "" {
			continue
		}
		jsonWithReferences, err := gen.GenerateJson([]byte(jsonWithRawIds))
		if err != nil {
			return fmt.Errorf("failed to generate bindings for field '%s': %w", field, err)
		}
		if err := data.Set(field, string(jsonWithReferences)); err != nil {
			return err
		}
	}

	// insert the bindings into the layout field to be used to generate data sources
	// and local variable definitions at a later point
	layout := data.Get("layout").(string)
	if layout == "" {
		layout = "{}"
	}
	layoutWithBindings, err := gen.InsertBindingsObjectJson([]byte(layout))
	if err != nil {
		return err
	}
	if err := data.Set("layout", string(layoutWithBindings)); err != nil {
		return err
	}
	return nil
}
