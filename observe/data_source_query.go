package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
							Optional: true,
						},
					},
				},
			},
			"poll": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "15s",
							ValidateDiagFunc: validateTimeDuration,
						},
						"timeout": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "2m",
							ValidateDiagFunc: validateTimeDuration,
						},
					},
				},
			},
			"assert": &schema.Schema{
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Validate expected query output",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"update": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"golden_file": {
							Type:        schema.TypeString,
							Description: "Filename containing expected query output.",
							Required:    true,
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
		client      = meta.(*observe.Client)
		queryResult *observe.QueryResult
	)

	query, diags := newQueryConfig(data)
	if diags.HasError() {
		return diags
	}

	var poller Poller

	// if no interval is set, poller will run exactly once
	if v, ok := data.GetOk("poll.0.interval"); ok && v != nil {
		d, _ := time.ParseDuration(v.(string))
		poller.Interval = &d
	}

	if v, ok := data.GetOk("poll.0.timeout"); ok && v != nil {
		d, _ := time.ParseDuration(v.(string))
		poller.Timeout = &d
	}

	err := poller.Run(ctx, func(ctx context.Context) error {
		var err error

		if _, ok := data.GetOk("end"); !ok {
			// reset end time on every subsequent request
			query.End = time.Now().Truncate(time.Second).UTC()
		}

		queryResult, err = client.Query(ctx, query)
		return err
	}, func() bool {
		return queryResult != nil && len(queryResult.Rows) > 0
	})

	if err != nil {
		diags = diag.FromErr(err)
		return
	}

	data.SetId(queryResult.ID)
	if diags = queryToResourceData(queryResult, data); diags.HasError() {
		return
	}

	if v, ok := data.GetOk("assert.0.golden_file"); ok {
		var (
			filename = v.(string)
			update   = data.Get("assert.0.update").(bool)
		)

		if update {
			// we indent only when writing to golden file, since we want pretty diffs
			data, err := json.MarshalIndent(queryResult.Rows, "", "  ")
			if err != nil {
				return diag.Errorf("failed to marshal rows: %s", err)
			}

			if err := ioutil.WriteFile(filename, data, os.FileMode(0644)); err != nil {
				return diag.Errorf("failed to write to golden file: %s", err)
			}
		} else {
			golden_data, err := ioutil.ReadFile(v.(string))
			if err != nil {
				return diag.Errorf("failed to read golden file: %s", err)
			}

			// Unfortunately we need to marshal to JSON in order to compare
			// correctly with golden file, otherwise types won't match.
			// Fortunately perf is not an issue for the result sizes we'll be
			// handling.
			returned_rows, err := json.Marshal(queryResult.Rows)
			if err != nil {
				return diag.Errorf("failed to marshal returned rows: %s", err)
			}

			// compare JSON strings
			transformJSON := cmp.FilterValues(func(x, y []byte) bool {
				return json.Valid(x) && json.Valid(y)
			}, cmp.Transformer("ParseJSON", func(in []byte) (out interface{}) {
				_ = json.Unmarshal(in, &out)
				return out
			}))

			// ... while ignoring timestamps
			ignoreTimestamps := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
				typerep, ok := queryResult.ColTypeRep(k)
				return ok && typerep == "timestamp"
			})

			if diff := cmp.Diff(returned_rows, golden_data, transformJSON, ignoreTimestamps); diff != "" {
				return diag.Errorf("query result does not match golden file: %s", diff)
			}
		}
	}

	return
}

func queryToResourceData(q *observe.QueryResult, data *schema.ResourceData) (diags diag.Diagnostics) {
	rows, err := json.Marshal(q.Rows)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := data.Set("result", string(rows)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
