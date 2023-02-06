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

func dataSourceMonitor() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("monitor", "description"),
		ReadContext: dataSourceMonitorRead,
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				RequiredWith: []string{"workspace"},
				Description: descriptions.Get("monitor", "schema", "name") +
					"One of `name` or `id` must be set. If `name` is provided, `workspace` must be set.",
			},
			"id": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
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
			"comment": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor", "schema", "comment"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: descriptions.Get("monitor", "schema", "disabled"),
			},
			"is_template": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: descriptions.Get("monitor", "schema", "is_template"),
			},
			"inputs": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: descriptions.Get("transform", "schema", "inputs"),
			},
			"stage": {
				Type:     schema.TypeList,
				Computed: true,
				// we need to declare optional, otherwise we won't get block
				// formatting in state
				Optional:    true,
				Description: descriptions.Get("transform", "schema", "stage", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
						"input": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "input"),
						},
						"pipeline": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "pipeline"),
						},
					},
				},
			},
			"rule": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source_column": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"group_by_group": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"columns": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"group_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"count": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"compare_values": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeFloat},
									},
									"lookback_time": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"change": {
							Type:     schema.TypeList,
							Computed: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"change_type": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"compare_function": {
										Type:     schema.TypeString,
										Required: true,
									},
									"aggregate_function": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"compare_values": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeFloat},
									},
									"lookback_time": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"baseline_time": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"facet": {
							Type:     schema.TypeList,
							Computed: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"facet_function": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"facet_values": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"time_function": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"time_value": {
										Type:     schema.TypeFloat,
										Computed: true,
									},
									"lookback_time": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"threshold": {
							Type:     schema.TypeList,
							Computed: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"compare_function": {
										Type:             schema.TypeString,
										Computed:         true,
										Optional:         true,
										ValidateDiagFunc: validateEnums(gql.AllCompareFunctions),
									},
									"compare_values": {
										Type:     schema.TypeList,
										Computed: true,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeFloat},
									},
									"lookback_time": {
										Type:             schema.TypeString,
										Computed:         true,
										Optional:         true,
										DiffSuppressFunc: diffSuppressTimeDuration,
										ValidateDiagFunc: validateTimeDuration,
									},
									"threshold_agg_function": {
										Type:             schema.TypeString,
										Computed:         true,
										Optional:         true,
										ValidateDiagFunc: validateEnums(gql.AllThresholdAggFunctions),
									},
								},
							},
						},
						"promote": {
							Type:     schema.TypeList,
							Computed: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"primary_key": {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"kind_field": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"description_field": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"notification_spec": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"importance": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"merge": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceMonitorRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		name       = data.Get("name").(string)
		explicitId = data.Get("id").(string)
	)

	var m *gql.Monitor
	var err error

	if explicitId != "" {
		m, err = client.GetMonitor(ctx, explicitId)
	} else if name != "" {

		var implicitId *oid.OID
		implicitId, _ = oid.NewOID(data.Get("workspace").(string))
		if err == nil {
			m, err = client.LookupMonitor(ctx, implicitId.Id, name)
		}
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(m.Id)
	return resourceMonitorRead(ctx, data, meta)
}
