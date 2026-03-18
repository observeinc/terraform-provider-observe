package observe

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

type fwStageModel struct {
	Alias       types.String `tfsdk:"alias"`
	Input       types.String `tfsdk:"input"`
	Pipeline    types.String `tfsdk:"pipeline"`
	OutputStage types.Bool   `tfsdk:"output_stage"`
}

// buildMultiStageQuery converts framework model inputs + stages into the GQL MultiStageQueryInput.
// This mirrors the SDKv2 newQuery function in data_source_query.go.
func buildMultiStageQuery(inputs map[string]string, stages []fwStageModel) (*gql.MultiStageQueryInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	inputIds := make(map[string]string, len(inputs))
	for k, v := range inputs {
		parsed, err := oid.NewOID(v)
		if err != nil {
			diags.AddError("Invalid input OID", fmt.Sprintf("input %q: %s", k, err))
			return nil, diags
		}
		inputIds[k] = parsed.Id
	}

	localStages := make([]Stage, len(stages))
	for i, s := range stages {
		if !s.Alias.IsNull() && !s.Alias.IsUnknown() && s.Alias.ValueString() != "" {
			v := s.Alias.ValueString()
			localStages[i].Alias = &v
		}
		if !s.Input.IsNull() && !s.Input.IsUnknown() && s.Input.ValueString() != "" {
			v := s.Input.ValueString()
			localStages[i].Input = &v
		}
		if !s.Pipeline.IsNull() && !s.Pipeline.IsUnknown() {
			localStages[i].Pipeline = s.Pipeline.ValueString()
		}
		if !s.OutputStage.IsNull() && !s.OutputStage.IsUnknown() {
			localStages[i].OutputStage = s.OutputStage.ValueBool()
		}
	}

	outputStagesCount := getOutputStagesCount(localStages)
	if outputStagesCount > 1 {
		diags.AddError("Invalid stages", errMoreThanOneOutputStages.Error())
		return nil, diags
	}

	var sortedNames []string
	gqlInputs := make(map[string]*gql.InputDefinitionInput, len(inputIds))
	for name, id := range inputIds {
		id := id
		if id == "" {
			diags.AddError("Invalid input", fmt.Sprintf("input %q: %s", name, errInputEmpty.Error()))
			return nil, diags
		}
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			diags.AddError("Invalid input", fmt.Sprintf("input %q: %s", name, errObjectIDInvalid.Error()))
			return nil, diags
		}
		gqlInputs[name] = &gql.InputDefinitionInput{
			InputName: name,
			DatasetId: &id,
		}
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	var defaultInput *gql.InputDefinitionInput
	switch len(gqlInputs) {
	case 0:
		diags.AddError("Invalid inputs", errInputsMissing.Error())
		return nil, diags
	case 1:
		defaultInput = gqlInputs[sortedNames[0]]
	}

	var query gql.MultiStageQueryInput

	for i, stage := range localStages {
		stageInputId := fmt.Sprintf("stage-%d", i)
		stageInput := gql.StageQueryInput{
			Id:       &stageInputId,
			Pipeline: stage.Pipeline,
		}

		if stage.Input != nil {
			v, ok := gqlInputs[*stage.Input]
			if !ok {
				diags.AddError("Invalid stage", fmt.Sprintf("stage-%d: %q: %s", i, *stage.Input, errStageInputUnresolved.Error()))
				return nil, diags
			}
			defaultInput = v
		}

		if defaultInput == nil {
			diags.AddError("Invalid stage", fmt.Sprintf("stage-%d: %s", i, errStageInputMissing.Error()))
			return nil, diags
		}

		stageInput.Input = append(stageInput.Input, *defaultInput)

		for _, name := range sortedNames {
			input := gqlInputs[name]
			if input == defaultInput {
				continue
			}
			if strings.Contains(stage.Pipeline, fmt.Sprintf("@%s", input.InputName)) || strings.Contains(stage.Pipeline, fmt.Sprintf("@%q", input.InputName)) {
				stageInput.Input = append(stageInput.Input, *input)
			}
		}

		query.Stages = append(query.Stages, stageInput)

		if outputStagesCount == 0 || stage.OutputStage {
			query.OutputStage = stageInputId
		}

		defaultInput = &gql.InputDefinitionInput{
			InputName: stageInputId,
			StageId:   &stageInputId,
		}

		if stage.Alias != nil {
			defaultInput.InputName = *stage.Alias
			gqlInputs[*stage.Alias] = defaultInput
			sortedNames = append(sortedNames, *stage.Alias)
		}
	}

	if query.OutputStage == "" {
		diags.AddError("Invalid stages", errStagesMissing.Error())
		return nil, diags
	}

	return &query, diags
}

// flattenQueryToModel converts GQL stage data back into framework model types.
// Returns the inputs map (name -> OID string) and the stage models.
// existingInputs is used to preserve OID versions from the prior state.
func flattenQueryToModel(_ context.Context, gqlStages []gql.StageQuery, outputStage string, existingInputs map[string]string, existingFirstStageInput string) (map[string]string, []fwStageModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	query, err := flattenQuery(gqlStages, outputStage)
	if err != nil {
		diags.AddError("Failed to flatten query", err.Error())
		return nil, nil, diags
	}

	inputs := make(map[string]string, len(query.Inputs))
	for name, input := range query.Inputs {
		id := oid.OID{
			Type: oid.TypeDataset,
			Id:   *input.Dataset,
		}
		// Preserve version from existing state to avoid spurious diffs
		if existing, ok := existingInputs[name]; ok {
			prv, err := oid.NewOID(existing)
			if err == nil && id.Id == prv.Id {
				id.Version = prv.Version
			}
		}
		inputs[name] = id.String()
	}

	stages := make([]fwStageModel, len(query.Stages))
	for i, s := range query.Stages {
		stages[i] = fwStageModel{
			Pipeline:    types.StringValue(s.Pipeline),
			OutputStage: types.BoolValue(s.OutputStage),
		}
		if s.Alias != nil {
			stages[i].Alias = types.StringValue(*s.Alias)
		} else {
			stages[i].Alias = types.StringValue("")
		}
		if s.Input != nil {
			stages[i].Input = types.StringValue(*s.Input)
		} else if i == 0 {
			// Preserve the first stage's input from prior state if not set by API
			stages[i].Input = types.StringValue(existingFirstStageInput)
		} else {
			stages[i].Input = types.StringValue("")
		}
	}

	return inputs, stages, diags
}
