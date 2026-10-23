package observe

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceInboundShare() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("inbound_share", "description"),
		CreateContext: resourceInboundShareCreate,
		ReadContext:   resourceInboundShareRead,
		UpdateContext: resourceInboundShareUpdate,
		DeleteContext: resourceInboundShareDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceInboundShareImport,
		},
		Schema: map[string]*schema.Schema{
			"share_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: descriptions.Get("inbound_share", "schema", "share_name"),
			},
			"provider_account": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: descriptions.Get("inbound_share", "schema", "provider_account"),
			},
			"label": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("inbound_share", "schema", "label"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"provider_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "provider_type"),
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "status"),
			},
			"health": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "health"),
			},
			"health_message": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "health_message"),
			},
			"last_health_check": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "last_health_check"),
			},
			"table_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "table_count"),
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "created_at"),
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("inbound_share", "schema", "updated_at"),
			},
		},
	}
}

func inboundShareToResourceData(s *rest.Share, data *schema.ResourceData) (diags diag.Diagnostics) {
	healthMessage := ""
	if s.Status.HealthMessage != nil {
		healthMessage = *s.Status.HealthMessage
	}
	lastHealthCheck := ""
	if s.Status.LastHealthCheckAt != nil {
		lastHealthCheck = *s.Status.LastHealthCheckAt
	}

	// An unset label is kept unset while the server reports the default
	// label, so that removing label from the config is a change to apply.
	label := s.Label
	if data.Get("label").(string) == "" && isDefaultInboundShareLabel(s, data.Get("share_name").(string)) {
		label = ""
	}

	values := map[string]interface{}{
		"label":             label,
		"oid":               s.Oid().String(),
		"provider_type":     s.ProviderType,
		"status":            s.Status.State,
		"health":            s.Status.Health,
		"health_message":    healthMessage,
		"last_health_check": lastHealthCheck,
		"table_count":       s.TableCount,
		"created_at":        s.CreatedAt,
		"updated_at":        s.UpdatedAt,
	}
	// The server may normalise the case of unquoted names, so the configured
	// values are kept unless there are none (on import).
	if _, ok := data.GetOk("share_name"); !ok {
		values["share_name"] = s.SnowflakeShareName()
	}
	if _, ok := data.GetOk("provider_account"); !ok {
		values["provider_account"] = s.SnowflakeProviderAccount()
	}
	for key, value := range values {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	return diags
}

// isDefaultInboundShareLabel reports whether the share's label is the one the
// server assigns when none is given, which is the share name.
func isDefaultInboundShareLabel(s *rest.Share, configuredShareName string) bool {
	return s.Label == "" || s.Label == configuredShareName || s.Label == s.ShareName || s.Label == s.SnowflakeShareName()
}

// inboundShareLabel is the label to send: the configured one, or the share
// name when label is unset.
func inboundShareLabel(data *schema.ResourceData) string {
	if label := data.Get("label").(string); label != "" {
		return label
	}
	return data.Get("share_name").(string)
}

func resourceInboundShareCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	shareName := data.Get("share_name").(string)
	providerAccount := data.Get("provider_account").(string)
	share, created, err := client.RegisterShare(ctx, &rest.RegisterShareRequest{
		ProviderAccount: providerAccount,
		ShareName:       shareName,
	})
	if err != nil {
		return inboundShareErrorDiags("failed to register inbound share", err)
	}
	if !created {
		return inboundShareAlreadyRegisteredDiags(share, shareName, providerAccount)
	}
	data.SetId(share.Id)

	if label, ok := data.GetOk("label"); ok && label.(string) != share.Label {
		share, err = client.UpdateShare(ctx, share.Id, &rest.UpdateShareRequest{Label: label.(string)})
		if err != nil {
			return inboundShareErrorDiags("failed to set inbound share label", err)
		}
	}

	return inboundShareToResourceData(share, data)
}

func resourceInboundShareRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	share, err := client.GetShare(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return inboundShareErrorDiags("failed to read inbound share", err)
	}

	return inboundShareToResourceData(share, data)
}

func resourceInboundShareUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	share, err := client.UpdateShare(ctx, data.Id(), &rest.UpdateShareRequest{
		Label: inboundShareLabel(data),
	})
	if err != nil {
		return inboundShareErrorDiags("failed to update inbound share", err)
	}

	return inboundShareToResourceData(share, data)
}

func resourceInboundShareDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observe.Client)

	if err := client.DeleteShare(ctx, data.Id(), false); err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			return nil
		}
		return inboundShareErrorDiags("failed to delete inbound share", err)
	}
	return nil
}

// resourceInboundShareImport accepts either the share ID or its OID.
//
// Example: terraform import observe_inbound_share.example 41012345
func resourceInboundShareImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if shareOid, err := oid.NewOID(data.Id()); err == nil {
		data.SetId(shareOid.Id)
	}
	return []*schema.ResourceData{data}, nil
}
