package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceOID() *schema.Resource {
	return &schema.Resource{
		Description: "Parses or formats an Observe OID. To parse an OID, only set the `oid` attribute. To format an OID, set the `type`, `id`, and optionally the `version` attributes. This is a logical data source and does not make any API calls.",

		ReadContext: dataSourceOIDRead,

		Schema: map[string]*schema.Schema{
			"oid": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      descriptions.Get("common", "schema", "oid"),
				ValidateDiagFunc: validateOID(),
			},
			"type": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      descriptions.Get("common", "schema", "type"),
				ValidateDiagFunc: validateOIDType,
				ConflictsWith:    []string{"oid"},
				RequiredWith:     []string{"id"},
			},
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				Description:   descriptions.Get("common", "schema", "id"),
				ConflictsWith: []string{"oid"},
				RequiredWith:  []string{"type"},
			},
			"version": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				Description:   descriptions.Get("common", "schema", "version"),
				ConflictsWith: []string{"oid"},
				RequiredWith:  []string{"id", "type"},
			},
		},
	}
}

func dataSourceOIDRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	if v, ok := d.GetOk("oid"); ok {
		o, err := oid.NewOID(v.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		d.SetId(o.Id)
		d.Set("type", o.Type)
		d.Set("version", o.Version)

		return diags
	}

	o := oid.OID{
		Type: oid.Type(d.Get("type").(string)),
		Id:   d.Get("id").(string),
	}

	if v, ok := d.GetOk("version"); ok {
		o.Version = stringPtr(v.(string))
	}

	d.SetId(o.Id)
	d.Set("oid", o.String())

	return diags
}
