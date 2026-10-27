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
		Schema: inboundShareResourceSchema(),
	}
}

func inboundShareResourceSchema() map[string]*schema.Schema {
	s := inboundShareStatusSchema()
	s["share_name"] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		DiffSuppressFunc: suppressCanonicalDiff(canonicalInboundShareName),
		Description:      descriptions.Get("inbound_share", "schema", "share_name"),
	}
	s["provider_account"] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		DiffSuppressFunc: suppressCanonicalDiff(canonicalInboundShareProviderAccount),
		Description:      descriptions.Get("inbound_share", "schema", "provider_account"),
	}
	s["label"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: descriptions.Get("inbound_share", "schema", "label"),
	}
	return s
}

func inboundShareToResourceData(s *rest.Share, data *schema.ResourceData) diag.Diagnostics {
	// An unset label is kept unset while the server reports the default
	// label, so that removing label from the config is a change to apply.
	label := s.Label
	if data.Get("label").(string) == "" && isDefaultInboundShareLabel(s, data.Get("share_name").(string)) {
		label = ""
	}

	values := inboundShareStatusValues(s)
	values["label"] = label
	values["share_name"] = inboundShareStateName(s.SnowflakeShareName())
	values["provider_account"] = s.SnowflakeProviderAccount()
	return setResourceDataValues(data, values)
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

	label, hasLabel := data.GetOk("label")
	if !hasLabel || label.(string) == share.Label {
		return inboundShareToResourceData(share, data)
	}
	updated, err := client.UpdateShare(ctx, share.Id, &rest.UpdateShareRequest{Label: label.(string)})
	if err != nil {
		// The share is registered, so it is kept rather than tainted: the
		// state holds the server's label and the next apply retries it.
		diags := inboundShareToResourceData(share, data)
		d := inboundShareErrorDiags("inbound share registered, but setting its label failed", err)[0]
		d.Severity = diag.Warning
		return append(diags, d)
	}
	return inboundShareToResourceData(updated, data)
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
