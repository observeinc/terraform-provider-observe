package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func dataSourceInboundShare() *schema.Resource {
	s := inboundShareStatusSchema()
	s["label"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "User-facing label for the share in Observe.",
	}
	for k, v := range dataSourceInboundShareLookupSchema() {
		s[k] = v
	}
	return &schema.Resource{
		Description: "Retrieves information about an inbound Snowflake data share.",
		ReadContext: dataSourceInboundShareRead,
		Schema:      s,
	}
}

func dataSourceInboundShareLookupSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// Input - either id OR snowflake_config
		"id": {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{"snowflake_config"},
			Description:   "The ID of the share to look up. Cannot be used with snowflake_config.",
		},
		"snowflake_config": {
			Type:          schema.TypeList,
			Optional:      true,
			Computed:      true,
			MaxItems:      1,
			ConflictsWith: []string{"id"},
			Description:   "Snowflake share configuration. Can be used as input for looking up the share, or as computed output containing the share's Snowflake configuration.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"share_name": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Description: "The Snowflake share name.",
					},
					"provider_account": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Description: "The Snowflake provider account (e.g., 'ACME_CORP.US-EAST-1').",
					},
				},
			},
		},
	}
}

func dataSourceInboundShareRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	var result *rest.Share
	var err error

	if id, ok := d.GetOk("id"); ok {
		// Direct GET by ID
		shareId := id.(string)
		result, err = client.GetShare(ctx, shareId)
		if err != nil {
			return inboundShareErrorDiags(fmt.Sprintf("failed to get inbound share %s", shareId), err)
		}
	} else if v, ok := d.GetOk("snowflake_config"); ok {
		// Lookup by snowflake_config (share_name + provider_account)
		configs := v.([]interface{})
		if len(configs) == 0 || configs[0] == nil {
			return diag.Errorf("snowflake_config must be specified")
		}
		config := configs[0].(map[string]interface{})
		shareName := config["share_name"].(string)
		providerAccount := config["provider_account"].(string)

		result, err = client.LookupShare(ctx, shareName, providerAccount)
		if err != nil {
			return inboundShareErrorDiags(fmt.Sprintf("failed to look up inbound share %s from provider account %s", shareName, providerAccount), err)
		}
	} else {
		return diag.Errorf("either 'id' or 'snowflake_config' must be provided")
	}

	return setShareData(d, result)
}

func setShareData(d *schema.ResourceData, s *rest.Share) diag.Diagnostics {
	d.SetId(s.Id)
	values := inboundShareStatusValues(s)
	values["label"] = s.Label
	values["snowflake_config"] = []interface{}{
		map[string]interface{}{
			"share_name":       s.SnowflakeShareName(),
			"provider_account": s.SnowflakeProviderAccount(),
		},
	}
	return setResourceDataValues(d, values)
}
