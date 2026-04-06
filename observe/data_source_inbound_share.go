package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func dataSourceInboundShare() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about an inbound Snowflake data share.",
		ReadContext: dataSourceInboundShareRead,

		Schema: map[string]*schema.Schema{
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

			// Outputs
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full OID of the share.",
			},
			"provider_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of share provider (e.g., 'Snowflake').",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operational state of the share (Pending, Creating, Active, Inactive, Error, Deleting).",
			},
			"health": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The health status of the share (Healthy, Unhealthy, Unknown).",
			},
			"health_message": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Human-readable health status message.",
			},
			"last_health_check": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of last health check.",
			},
			"table_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of tracked tables in this share.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last update timestamp.",
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
			return diag.Errorf("failed to get share %s: %v", shareId, err)
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
			return diag.Errorf("failed to lookup share %s from provider %s: %v", shareName, providerAccount, err)
		}
	} else {
		return diag.Errorf("either 'id' or 'snowflake_config' must be provided")
	}

	return setShareData(d, result)
}

func setShareData(d *schema.ResourceData, s *rest.Share) diag.Diagnostics {

	var diags diag.Diagnostics

	// Set resource ID
	d.SetId(s.Id)

	// Set all computed fields
	if err := d.Set("oid", s.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("provider_type", s.ProviderType); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("status", s.Status.State); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("health", s.Status.Health); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if s.Status.HealthMessage != nil {
		if err := d.Set("health_message", *s.Status.HealthMessage); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if s.Status.LastHealthCheck != nil {
		if err := d.Set("last_health_check", *s.Status.LastHealthCheck); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if err := d.Set("table_count", s.TableCount); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("created_at", s.CreatedAt); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("updated_at", s.UpdatedAt); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// Snowflake-specific configuration (output only - not used for input)
	// Set snowflake_config as a computed block to match API shape
	if s.SnowflakeConfig != nil {
		snowflakeConfig := []interface{}{
			map[string]interface{}{
				"share_name":       s.SnowflakeConfig.ShareName,
				"provider_account": s.SnowflakeConfig.ProviderAccount,
			},
		}
		if err := d.Set("snowflake_config", snowflakeConfig); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}
