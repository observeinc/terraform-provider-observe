package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	oid "github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceTerraform() *schema.Resource {
	return &schema.Resource{
		Description: "Generates Terraform configuration for a given resource in Observe. Datasets, monitors, and dashboards are supported.",
		ReadContext: dataSourceTerraformRead,
		Schema: map[string]*schema.Schema{
			"target": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset, oid.TypeMonitor, oid.TypeDashboard),
				Description:      descriptions.Get("terraform", "schema", "target"),
			},
			"resource": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("terraform", "schema", "resource"),
			},
			"data_source": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("terraform", "schema", "data_source"),
			},
			"import_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("terraform", "schema", "import_id"),
			},
			"import_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("terraform", "schema", "import_name"),
			},
		},
	}
}

func dataSourceTerraformRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client    = meta.(*observe.Client)
		target, _ = oid.NewOID(data.Get("target").(string))
	)

	var objType gql.TerraformObjectType
	switch target.Type {
	case oid.TypeDataset:
		objType = gql.TerraformObjectTypeDataset
	case oid.TypeMonitor:
		objType = gql.TerraformObjectTypeMonitor
	case oid.TypeDashboard:
		objType = gql.TerraformObjectTypeDashboard
	}

	r, err := client.GetTerraform(ctx, target.Id, objType)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(target.Id)

	if err := data.Set("data_source", r.DataSource); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("resource", r.Resource); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("import_id", r.ImportId); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("import_name", r.ImportName); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
