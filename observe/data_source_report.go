package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceReport() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("report", "schema", "description"),
		ReadContext: dataSourceReportRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateID(),
				Description:      descriptions.Get("common", "schema", "id"),
			},
			// computed values
			"created_by": {
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Description: descriptions.Get("report", "schema", "created_by", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "created_by", "id"),
						},
						"label": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "created_by", "label"),
						},
					},
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "created_at"),
			},
			"updated_by": {
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Description: descriptions.Get("report", "schema", "updated_by", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "updated_by", "id"),
						},
						"label": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "updated_by", "label"),
						},
					},
				},
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "updated_at"),
			},
			"label": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "label"),
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "enabled"),
			},
			"dashboard": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "id"),
						},
						"label": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "label"),
						},
						"parameters": {
							Type:        schema.TypeList,
							Computed:    true,
							Optional:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "parameters", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("report", "schema", "dashboard", "parameters", "key"),
									},
									"value": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("report", "schema", "dashboard", "parameters", "value"),
									},
								},
							},
						},
						"query_window_duration_minutes": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "query_window_duration_minutes"),
						},
					},
				},
			},
			"schedule": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"frequency": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "schedule", "frequency"),
						},
						"every": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "schedule", "every"),
						},
						"time_of_day": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "schedule", "time_of_day"),
						},
						"timezone": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "schedule", "timezone"),
						},
						"day_of_the_week": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "schedule", "day_of_the_week"),
						},
						"day_of_the_month": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "schedule", "day_of_the_month"),
						},
					},
				},
			},
			"email_subject": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "email_subject"),
			},
			"email_body": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "email_body"),
			},
			"email_recipients": {
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Description: descriptions.Get("report", "schema", "email_recipients"),
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"next_scheduled_time": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "next_scheduled_time"),
			},
			"last_run_time": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "last_run_time"),
			},
			"last_run_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "last_run_status"),
			},
			"last_run_error": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("report", "schema", "last_run_error"),
			},
		},
	}
}

func dataSourceReportRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	id := data.Get("id").(string)

	report, err := client.GetReport(ctx, id)
	if err != nil {
		diags = diag.FromErr(err)
		return
	} else if report == nil {
		return diag.Errorf("failed to lookup report")
	}

	data.SetId(report.Id)
	return reportToResourceData(report, data)
}
