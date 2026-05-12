package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceSkill() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("skill", "description"),
		ReadContext: dataSourceSkillRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateID(),
				Description:      descriptions.Get("common", "schema", "id"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"label": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("skill", "schema", "label"),
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("skill", "schema", "description"),
			},
			"content": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("skill", "schema", "content"),
			},
			"visibility": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("skill", "schema", "visibility"),
			},
		},
	}
}

func dataSourceSkillRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	id := data.Get("id").(string)

	skill, err := client.GetSkill(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if skill == nil {
		return diag.Errorf("failed to lookup skill")
	}

	data.SetId(skill.Id)
	return skillToResourceData(skill, data)
}
