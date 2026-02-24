package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceMonitorAction() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("monitor", "description"),
		ReadContext: dataSourceMonitorActionRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Computed:     true,
				RequiredWith: []string{"workspace"},
				Description: descriptions.Get("monitor", "schema", "name") +
					"One of `name` or `id` must be set. If `name` is provided, `workspace` must be set.",
			},
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"name", "id"},
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateID(),
				Description: descriptions.Get("common", "schema", "id") +
					"One of `id` or `name` must be provided",
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor", "schema", "description"),
			},
			"rate_limit": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_action", "schema", "rate_limit"),
			},
			"notify_on_close": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: descriptions.Get("monitor_action", "schema", "notify_on_close"),
			},
			"email": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_addresses": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"subject_template": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"body_template": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"is_html": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"webhook": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url_template": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"method": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"body_template": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"headers": {
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceMonitorActionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		name       = data.Get("name").(string)
		explicitId = data.Get("id").(string)
	)

	var m *gql.MonitorAction
	var err error

	if explicitId != "" {
		m, err = client.GetMonitorAction(ctx, explicitId)
	} else if name != "" {
		var implicitId *oid.OID
		implicitId, _ = oid.NewOID(data.Get("workspace").(string))
		if err == nil {
			m, err = client.LookupMonitorAction(ctx, implicitId.Id, name)
		}
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId((*m).GetId())
	return resourceMonitorActionRead(ctx, data, meta)
}
