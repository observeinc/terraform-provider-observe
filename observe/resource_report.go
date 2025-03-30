package observe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceReport() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("report", "description"),
		CreateContext: resourceReportCreate,
		ReadContext:   resourceReportRead,
		UpdateContext: resourceReportUpdate,
		DeleteContext: resourceReportDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"created_by": {
				Type:        schema.TypeList,
				Computed:    true,
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
				Required:    true,
				Description: descriptions.Get("report", "schema", "label"),
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: descriptions.Get("report", "schema", "enabled"),
			},
			"dashboard": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: descriptions.Get("report", "schema", "dashboard", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "id"),
						},
						"label": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "label"),
						},
						"parameters": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "parameters", "description"),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Required:    true,
										Description: descriptions.Get("report", "schema", "dashboard", "parameters", "key"),
									},
									"value": {
										Type:        schema.TypeString,
										Required:    true,
										Description: descriptions.Get("report", "schema", "dashboard", "parameters", "value"),
									},
								},
							},
						},
						"query_window_duration_minutes": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: descriptions.Get("report", "schema", "dashboard", "query_window_duration_minutes"),
						},
					},
				},
			},
			"schedule": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: descriptions.Get("report", "schema", "schedule", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"frequency": {
							Type:             schema.TypeString,
							Required:         true,
							Description:      descriptions.Get("report", "schema", "schedule", "frequency"),
							ValidateDiagFunc: validateStringInSlice([]string{"Daily", "Weekly", "Monthly"}, false),
						},
						"every": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: descriptions.Get("report", "schema", "schedule", "every"),
						},
						"time_of_day": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("report", "schema", "schedule", "time_of_day"),
						},
						"timezone": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions.Get("report", "schema", "schedule", "timezone"),
						},
						"day_of_the_week": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions.Get("report", "schema", "schedule", "day_of_the_week"),
						},
						"day_of_the_month": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: descriptions.Get("report", "schema", "schedule", "day_of_the_month"),
						},
					},
				},
			},
			"email_subject": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("report", "schema", "email_subject"),
			},
			"email_body": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("report", "schema", "email_body"),
			},
			"email_recipients": {
				Type:        schema.TypeList,
				Required:    true,
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

func reportToResourceData(report *rest.ReportsResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	setResourceData := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	data.SetId(report.Id)
	setResourceData("oid", report.Oid().String())

	createdBy := map[string]interface{}{
		"id":    report.CreatedBy.Id,
		"label": report.CreatedBy.Label,
	}
	setResourceData("created_by", []map[string]interface{}{createdBy})
	setResourceData("created_at", report.CreatedAt)

	updatedBy := map[string]interface{}{
		"id":    report.UpdatedBy.Id,
		"label": report.UpdatedBy.Label,
	}
	setResourceData("updated_by", []map[string]interface{}{updatedBy})
	setResourceData("updated_at", report.UpdatedAt)

	setResourceData("enabled", report.Enabled)
	setResourceData("label", report.Label)

	dashboard := make(map[string]interface{})
	dashboard["id"] = report.Dashboard.Id
	dashboard["label"] = report.Dashboard.Label
	parameters := make([]map[string]interface{}, 0, len(report.Dashboard.Parameters))
	for k, v := range report.Dashboard.Parameters {
		parameters = append(parameters, map[string]interface{}{
			"key":   k,
			"value": v,
		})
	}
	dashboard["parameters"] = parameters
	dashboard["query_window_duration_minutes"] = report.Dashboard.QueryWindowDurationMinutes
	setResourceData("dashboard", []map[string]interface{}{dashboard})

	schedule := make(map[string]interface{})
	schedule["frequency"] = report.Schedule.Frequency
	schedule["every"] = report.Schedule.Every
	schedule["time_of_day"] = report.Schedule.TimeOfDay
	schedule["timezone"] = report.Schedule.Timezone
	schedule["day_of_the_week"] = report.Schedule.DayOfTheWeek
	schedule["day_of_the_month"] = report.Schedule.DayOfTheMonth
	setResourceData("schedule", []map[string]interface{}{schedule})

	setResourceData("email_subject", report.EmailSubject)
	setResourceData("email_body", report.EmailBody)
	setResourceData("email_recipients", report.EmailRecipients)

	if report.NextScheduleTime != nil {
		setResourceData("next_scheduled_time", *report.NextScheduleTime)
	}
	if report.LastRunTime != nil {
		setResourceData("last_run_time", *report.LastRunTime)
	}
	if report.LastRunStatus != nil {
		setResourceData("last_run_status", *report.LastRunStatus)
	}
	if report.LastRunError != nil {
		setResourceData("last_run_error", *report.LastRunError)
	}

	return diags
}

func reportDefinitionFromResourceData(data *schema.ResourceData) (req *rest.ReportsDefinition, diags diag.Diagnostics) {
	req = &rest.ReportsDefinition{}

	req.Label = data.Get("label").(string)

	var ok bool

	// Dashboard
	{
		req.Dashboard = rest.ReportsDashboard{}
		req.Dashboard.Id, ok = data.Get("dashboard.0.id").(string)
		if !ok {
			return
		}
		req.Dashboard.Parameters = make(map[string]interface{})
		for i := 0; i < data.Get("dashboard.0.parameters.#").(int); i++ {
			key := data.Get(fmt.Sprintf("dashboard.0.parameters.%d.key", i)).(string)
			value := data.Get(fmt.Sprintf("dashboard.0.parameters.%d.value", i))
			req.Dashboard.Parameters[key] = value
		}
		req.Dashboard.QueryWindowDurationMinutes = int32(data.Get("dashboard.0.query_window_duration_minutes").(int))
	}

	req.Enabled = data.Get("enabled").(bool)

	// Schedule
	{
		req.Schedule = rest.ReportsSchedule{}
		req.Schedule.Frequency = data.Get("schedule.0.frequency").(string)
		req.Schedule.Every = int32(data.Get("schedule.0.every").(int))
		req.Schedule.TimeOfDay = data.Get("schedule.0.time_of_day").(string)
		if v, ok := data.GetOk("schedule.0.timezone"); ok {
			req.Schedule.Timezone = v.(string)
		}
		if v, ok := data.GetOk("schedule.0.day_of_the_week"); ok {
			req.Schedule.DayOfTheWeek = v.(string)
		}
		if v, ok := data.GetOk("schedule.0.day_of_the_month"); ok {
			req.Schedule.DayOfTheMonth = v.(int)
		}
	}

	req.EmailSubject = data.Get("email_subject").(string)
	req.EmailBody = data.Get("email_body").(string)
	numEmailRecipients := data.Get("email_recipients.#").(int)
	req.EmailRecipients = make([]string, numEmailRecipients)
	for i := 0; i < numEmailRecipients; i++ {
		req.EmailRecipients[i] = data.Get(fmt.Sprintf("email_recipients.%d", i)).(string)
	}

	return
}

func resourceReportCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	req, diags := reportDefinitionFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	client := meta.(*observe.Client)
	report, err := client.CreateReport(ctx, req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create report",
			Detail:   err.Error(),
		})
		return diags
	}
	data.SetId(report.Id)

	return append(diags, reportToResourceData(report, data)...)
}

func resourceReportUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	req, diags := reportDefinitionFromResourceData(data)
	if diags.HasError() {
		return diags
	}

	client := meta.(*observe.Client)
	report, err := client.UpdateReport(ctx, data.Id(), req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to update report",
			Detail:   err.Error(),
		})
		return diags
	}

	return append(diags, reportToResourceData(report, data)...)

}

func resourceReportRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetReport(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve report: %s", err.Error())
	}

	return reportToResourceData(result, data)
}

func resourceReportDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	err := client.DeleteReport(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to delete report: %s", err.Error())
	}
	return diags
}
