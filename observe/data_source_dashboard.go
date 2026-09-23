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
			"schema_version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: schemaDashboardSchemaVersionDescription,
			},
			"definition": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardDefinitionDescription,
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
			"object_tags": objectTagsSchemaFieldComputed(),
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

	workspaceOid := oid.WorkspaceOid(dashboard.WorkspaceId)
	values, err := decodeJsonFields(data, "definition", "stages", "parameters", "parameter_values", "layout")
	if err != nil {
		return err
	}
	gen.CollectOid(workspaceOid)
	for _, v := range values {
		gen.Collect(v)
	}
	if err := gen.Resolve(ctx); err != nil {
		return err
	}

	// replace the original ids in the json data with local variable references
	workspaceRef, _ := gen.TryBindOid(workspaceOid)
	for _, v := range values {
		gen.Generate(v)
	}

	// Insert the bindings into whichever field carries the dashboard's content, to be
	// used to generate data sources and local variable definitions at a later point. A
	// schema_version >= 2 dashboard carries its content entirely in definition; layout
	// is left empty in that case, so definition is the bindings carrier instead.
	bindingsField := "layout"
	if dashboardUsesRestAPI(dashboard.SchemaVersion) {
		bindingsField = "definition"
	}
	if _, ok := values[bindingsField]; !ok {
		values[bindingsField] = map[string]interface{}{}
	}
	carrier, ok := values[bindingsField].(map[string]interface{})
	if !ok {
		return fmt.Errorf("field '%s' is not a JSON object", bindingsField)
	}
	if err := gen.InsertBindingsObject(carrier); err != nil {
		return err
	}
	if err := encodeJsonFields(values); err != nil {
		return err
	}
	values["workspace"] = workspaceRef
	return setAll(data, values)
}
