package observe

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceQuery() *schema.Resource {
	return &schema.Resource{
		Description: "Queries data stored in Observe and returns the results.",

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
			"stage": {
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
			"poll": {
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
			"assert": {
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

type Query struct {
	Inputs map[string]*Input `json:"inputs"`
	Stages []*Stage          `json:"stages"`
}

// Stage applies a pipeline to an input
// If no input is provided, stage will follow on from previous stage
// An alias must be provided for callers to be able to reference this stage in OPAL
// Internally, the alias does not map to the stageID - it is the input name we
// use when refering to this stage
type Stage struct {
	Alias    *string `json:"alias,omitempty"`
	Input    *string `json:"input,omitempty"`
	Pipeline string  `json:"pipeline"`
}

// Input references an existing data source
type Input struct {
	Dataset *string ` json:"dataset,omitempty"`
}

func newQuery(data *schema.ResourceData) (*gql.MultiStageQueryInput, diag.Diagnostics) {
	inputIds := make(map[string]string)
	for k, v := range data.Get("inputs").(map[string]interface{}) {
		is, _ := oid.NewOID(v.(string))
		inputIds[k] = is.Id
	}

	stages := make([]Stage, 0)
	for i := range data.Get("stage").([]interface{}) {
		var stage Stage

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
		stages = append(stages, stage)
	}

	var sortedNames []string
	inputs := make(map[string]*gql.InputDefinitionInput, len(inputIds))
	for name, input := range inputIds {
		input := input

		if input == "" {
			return nil, diag.FromErr(errInputEmpty)
		}
		if _, err := strconv.ParseInt(input, 10, 64); err != nil {
			diagErr := fmt.Errorf("invalid dataset %s: %w", input, errObjectIDInvalid)
			return nil, diag.FromErr(diagErr)
		}
		inputs[name] = &gql.InputDefinitionInput{
			InputName: name,
			DatasetId: &input,
		}
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	var defaultInput *gql.InputDefinitionInput
	switch len(inputs) {
	case 0:
		return nil, diag.FromErr(errInputsMissing)
	case 1:
		// if only one input is provided, us it as input for first stage
		defaultInput = inputs[sortedNames[0]]
	}

	var query gql.MultiStageQueryInput

	// We're now ready to convert stages
	// If a stage is named, it can be used as an input for every subsequent stage.
	// If a stage is anonymous, it can still be used as a default input on the next stage.
	for i, stage := range stages {
		stageInputId := fmt.Sprintf("stage-%d", i)
		// Each stage will be given an ID based on the hash of all preceeding pipelines
		stageInput := gql.StageQueryInput{
			Id:       &stageInputId,
			Pipeline: stage.Pipeline,
		}

		// if stage has a declared input, update defaultInput
		if stage.Input != nil {
			v, ok := inputs[*stage.Input]
			if !ok {
				diagErr := fmt.Errorf("stage-%d: %q: %w", i, *stage.Input, errStageInputUnresolved)
				return nil, diag.FromErr(diagErr)
			}
			defaultInput = v
		}

		if defaultInput == nil {
			diagErr := fmt.Errorf("stage-%d: %w", i, errStageInputMissing)
			return nil, diag.FromErr(diagErr)
		}

		// construct stage inputs, first default, then any declared input that
		// is referenced in pipeline.
		stageInput.Input = append(stageInput.Input, *defaultInput)

		for _, name := range sortedNames {
			input := inputs[name]
			if input == defaultInput {
				continue
			}

			if strings.Contains(stage.Pipeline, fmt.Sprintf("@%s", input.InputName)) || strings.Contains(stage.Pipeline, fmt.Sprintf("@%q", input.InputName)) {
				stageInput.Input = append(stageInput.Input, *input)
			}
		}

		// stage is done, append to transform
		query.Stages = append(query.Stages, stageInput)
		query.OutputStage = stageInputId

		// prepare for next iteration of loop
		// this stage will become defaultInput for the next
		defaultInput = &gql.InputDefinitionInput{
			InputName: stageInputId,
			StageId:   &stageInputId,
		}

		// if explicitly named, this stage can be also be an input for the next
		if stage.Alias != nil {
			defaultInput.InputName = *stage.Alias
			// conflict?
			inputs[*stage.Alias] = defaultInput
			sortedNames = append(sortedNames, *stage.Alias)
		}
	}
	// a query must have at least one stage
	if query.OutputStage == "" {
		return nil, diag.FromErr(errStagesMissing)
	}

	return &query, nil
}

func newQueryConfig(data *schema.ResourceData) (query []*gql.StageInput, params *gql.QueryParams, diags diag.Diagnostics) {
	var (
		start, _ = time.Parse(time.RFC3339, data.Get("start").(string))
		limit, _ = data.Get("limit").(int)
	)

	end := time.Now().Truncate(time.Second).UTC()
	if v, ok := data.GetOk("end"); ok {
		end, _ = time.Parse(time.RFC3339, v.(string))
	}

	multiStageQueryInput, diags := newQuery(data)
	if diags.HasError() {
		return nil, nil, diags
	}

	// This is insane. StageQueryInput is a subset of StageInput, but differs
	// in the key of the input field: one has "input", the other "inputs".
	// Convert here rather than replicating all the conversion logic.
	stageInputs := make([]gql.StageInput, len(multiStageQueryInput.Stages))
	for i, s := range multiStageQueryInput.Stages {
		stageInputs[i] = gql.StageInput{
			Inputs:   s.Input,
			StageId:  *s.Id,
			Pipeline: s.Pipeline,
			Presentation: &gql.StagePresentationInput{
				ResultKinds: []gql.ResultKind{gql.ResultKindResultkindsuppress},
			},
		}
	}

	outputStage := stageInputs[len(stageInputs)-1]
	outputStage.Presentation.ResultKinds = []gql.ResultKind{gql.ResultKindResultkinddata, gql.ResultKindResultkindschema}
	limitParsed := types.Int64Scalar(limit)
	outputStage.Presentation.Limit = &limitParsed

	startParsed := types.TimeScalar(start)
	endParsed := types.TimeScalar(end)
	params = &gql.QueryParams{
		StartTime: &startParsed,
		EndTime:   &endParsed,
	}

	return query, params, nil
}

func dataSourceQueryRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	// TODO (OB-10912): Queries are currently broken
	return diag.Errorf("this feature is disabled until it can be updated to work with upstream API changes")

	// var (
	// 	client      = meta.(*observe.Client)
	// 	queryResult *observe.QueryResult
	// )

	// stages, params, diags := newQueryConfig(data)
	// if diags.HasError() {
	// 	return diags
	// }

	// var poller Poller

	// // if no interval is set, poller will run exactly once
	// if v, ok := data.GetOk("poll.0.interval"); ok && v != nil {
	// 	d, _ := time.ParseDuration(v.(string))
	// 	poller.Interval = &d
	// }

	// if v, ok := data.GetOk("poll.0.timeout"); ok && v != nil {
	// 	d, _ := time.ParseDuration(v.(string))
	// 	poller.Timeout = &d
	// }

	// err := poller.Run(ctx, func(ctx context.Context) error {
	// 	var err error

	// 	if _, ok := data.GetOk("end"); !ok {
	// 		// reset end time on every subsequent request
	// 		params.EndTime = types.TimeScalar(time.Now().Truncate(time.Second).UTC())
	// 	}

	// 	queryResult, err = client.Query(ctx, stages, params)
	// 	return err
	// }, func() bool {
	// 	return queryResult != nil && len(queryResult.Rows) > 0
	// })

	// if err != nil {
	// 	diags = diag.FromErr(err)
	// 	return
	// }

	// data.SetId(queryResult.ID)
	// if diags = queryToResourceData(queryResult, data); diags.HasError() {
	// 	return
	// }

	// if v, ok := data.GetOk("assert.0.golden_file"); ok {
	// 	var (
	// 		filename = v.(string)
	// 		update   = data.Get("assert.0.update").(bool)
	// 	)

	// 	if update {
	// 		// we indent only when writing to golden file, since we want pretty diffs
	// 		data, err := json.MarshalIndent(queryResult.Rows, "", "  ")
	// 		if err != nil {
	// 			return diag.Errorf("failed to marshal rows: %s", err)
	// 		}

	// 		if err := ioutil.WriteFile(filename, data, os.FileMode(0644)); err != nil {
	// 			return diag.Errorf("failed to write to golden file: %s", err)
	// 		}
	// 	} else {
	// 		golden_data, err := ioutil.ReadFile(v.(string))
	// 		if err != nil {
	// 			return diag.Errorf("failed to read golden file: %s", err)
	// 		}

	// 		// Unfortunately we need to marshal to JSON in order to compare
	// 		// correctly with golden file, otherwise types won't match.
	// 		// Fortunately perf is not an issue for the result sizes we'll be
	// 		// handling.
	// 		returned_rows, err := json.Marshal(queryResult.Rows)
	// 		if err != nil {
	// 			return diag.Errorf("failed to marshal returned rows: %s", err)
	// 		}

	// 		// compare JSON strings
	// 		transformJSON := cmp.FilterValues(func(x, y []byte) bool {
	// 			return json.Valid(x) && json.Valid(y)
	// 		}, cmp.Transformer("ParseJSON", func(in []byte) (out interface{}) {
	// 			_ = json.Unmarshal(in, &out)
	// 			return out
	// 		}))

	// 		// ... while ignoring timestamps
	// 		ignoreTimestamps := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
	// 			typerep, ok := queryResult.ColTypeRep(k)
	// 			return ok && typerep == "timestamp"
	// 		})

	// 		if diff := cmp.Diff(returned_rows, golden_data, transformJSON, ignoreTimestamps); diff != "" {
	// 			return diag.Errorf("query result does not match golden file: %s", diff)
	// 		}
	// 	}
	// }

	// return
}

// func queryToResourceData(q *observe.QueryResult, data *schema.ResourceData) (diags diag.Diagnostics) {
// 	rows, err := json.Marshal(q.Rows)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}

// 	if err := data.Set("result", string(rows)); err != nil {
// 		diags = append(diags, diag.FromErr(err)...)
// 	}

// 	return diags
// }

func flattenQuery(gqlStages []*gql.StageQuery) (*Query, error) {
	query := &Query{Inputs: make(map[string]*Input)}

	// first reconstruct all inputs
	stageIDs := make(map[string]string)
	for _, stageQuery := range gqlStages {
		for _, i := range stageQuery.Input {
			if i.GetDatasetId() != nil {
				datasetID := *i.GetDatasetId()
				query.Inputs[i.InputName] = &Input{Dataset: &datasetID}
			}
			if i.StageId != nil && *i.StageId != "" {
				stageIDs[*i.StageId] = i.InputName
			}
		}
	}

	for i, gqlStage := range gqlStages {
		stage := &Stage{
			Pipeline: gqlStage.Pipeline,
		}

		stageId := ""
		if gqlStage.Id != nil {
			stageId = *gqlStage.Id
		}
		if name, ok := stageIDs[stageId]; ok && name != stageId {
			stage.Alias = &name
		}

		inputName := gqlStage.Input[0].InputName

		switch {
		case i == 0 && len(query.Inputs) == 1:
			// defaulted to first input
		case i > 0 && query.Stages[i-1].Alias != nil && inputName == *(query.Stages[i-1].Alias):
			// follow on from aliased stage
		case stageIDs[inputName] != "":
			// follow on from anonymous stage
		default:
			stage.Input = &inputName
		}

		query.Stages = append(query.Stages, stage)
	}

	return query, nil
}
