package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func inboundShareSetupSqlStepSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share_setup", "schema", "sql_step", "description"),
			},
			"sql": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share_setup", "schema", "sql_step", "sql"),
			},
		},
	}
}

func dataSourceInboundShareSetup() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("inbound_share_setup", "description"),
		ReadContext: dataSourceInboundShareSetupRead,
		Schema: map[string]*schema.Schema{
			"consumer_account": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share_setup", "schema", "consumer_account"),
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share_setup", "schema", "region"),
			},
			"cloud": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share_setup", "schema", "cloud"),
			},
			"setup_sql": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        inboundShareSetupSqlStepSchema(),
				Description: descriptions.Get("inbound_share_setup", "schema", "setup_sql"),
			},
		},
	}
}

func dataSourceInboundShareSetupRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	info, err := client.GetShareSetupInfo(ctx)
	if err != nil {
		return inboundShareErrorDiags("failed to get inbound share setup info", err)
	}

	data.SetId(info.ConsumerAccount)
	return inboundShareSetupToResourceData(info, data)
}

func flattenSetupSqlSteps(steps []rest.SetupSqlStep) []interface{} {
	result := make([]interface{}, 0, len(steps))
	for _, step := range steps {
		result = append(result, map[string]interface{}{
			"description": step.Description,
			"sql":         step.Sql,
		})
	}
	return result
}

func inboundShareSetupToResourceData(info *rest.SetupInfo, data *schema.ResourceData) diag.Diagnostics {
	cloud := ""
	if info.Cloud != nil {
		cloud = *info.Cloud
	}

	guidance := info.Guidance
	values := map[string]interface{}{
		"consumer_account": info.ConsumerAccount,
		"region":           info.Region,
		"cloud":            cloud,
		"setup_sql":        flattenSetupSqlSteps(guidance.SetupSql),
	}
	return setResourceDataValues(data, values)
}
