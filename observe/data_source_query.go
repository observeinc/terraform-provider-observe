package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceQuery() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceQueryRead,

		Schema: map[string]*schema.Schema{
			"start": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateTimestamp,
				DefaultFunc: func() (interface{}, error) {
					return time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339), nil
				},
			},
			"end": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "End timestamp. If omitted, query will be periodically re-run until results are returned.",
				ValidateDiagFunc: validateTimestamp,
			},
			"limit": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
			},
			"inputs": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateOID()),
			},
			"stage": &schema.Schema{
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:     schema.TypeString,
							Optional: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								// ignore alias for last stage, because it won't be set anyway
								stage := d.Get("stage").([]interface{})
								return k == fmt.Sprintf("stage.%d.alias", len(stage)-1)
							},
						},
						"input": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"pipeline": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"result": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func newQuery(data *schema.ResourceData) (query *observe.Query, diags diag.Diagnostics) {
	query = &observe.Query{
		Inputs: make(map[string]*observe.Input),
	}

	for k, v := range data.Get("inputs").(map[string]interface{}) {
		oid, _ := observe.NewOID(v.(string))
		query.Inputs[k] = &observe.Input{
			Dataset: &oid.ID,
		}
	}

	for i := range data.Get("stage").([]interface{}) {
		var stage observe.Stage

		if v, ok := data.GetOk(fmt.Sprintf("stage.%d.alias", i)); ok {
			s := v.(string)
			stage.Alias = &s
		}

		if v, ok := data.GetOk(fmt.Sprintf("stage.%d.input", i)); ok {
			s := v.(string)
			stage.Input = &s
		}

		if v, ok := data.GetOk(fmt.Sprintf("stage.%d.pipeline", i)); ok {
			stage.Pipeline = v.(string)
		}
		query.Stages = append(query.Stages, &stage)
	}

	return query, diags
}

func newQueryConfig(data *schema.ResourceData) (config *observe.QueryConfig, diags diag.Diagnostics) {
	var (
		start, _ = time.Parse(time.RFC3339, data.Get("start").(string))
		limit, _ = data.Get("limit").(int)
	)

	end := time.Now().Truncate(time.Second).UTC()
	if v, ok := data.GetOk("end"); ok {
		end, _ = time.Parse(time.RFC3339, v.(string))
	}

	query, diags := newQuery(data)
	if diags.HasError() {
		return nil, diags
	}

	return &observe.QueryConfig{
		Query: query,
		Limit: int64(limit),
		Start: start,
		End:   end,
	}, nil
}

func dataSourceQueryRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
	)

	query, diags := newQueryConfig(data)
	if diags.HasError() {
		return diags
	}

	if _, ok := data.GetOk("end"); !ok {
		// reset end on every subsequent request
		query.End = time.Now().Truncate(time.Second).UTC()
	}

	queryResult, err := client.Query(ctx, query)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}

	data.SetId(queryResult.ID)
	if diags = queryToResourceData(queryResult, data); diags.HasError() {
		return
	}
	return
}

func queryToResourceData(q *observe.QueryResult, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("result", string(q.JSON)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
