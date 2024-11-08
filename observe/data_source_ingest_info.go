package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceIngestInfo() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("ingest_info", "description"),
		ReadContext: dataSourceIngestInfoRead,
		Schema: map[string]*schema.Schema{
			// computed values
			"collect_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("ingest_info", "schema", "collect_url"),
			},
			"domain": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("ingest_info", "schema", "domain"),
			},
			"scheme": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("ingest_info", "schema", "scheme"),
			},
			"port": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("ingest_info", "schema", "port"),
			},
		},
	}
}

func dataSourceIngestInfoRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
	)

	info, err := client.GetIngestInfo(ctx)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	return ingestInfoToResourceData(info, data)
}

func ingestInfoToResourceData(info *gql.IngestInfo, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("collect_url", info.CollectUrl); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("scheme", info.Scheme); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("domain", info.Domain); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("port", info.Port); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	data.SetId(info.CollectUrl)
	return diags
}
