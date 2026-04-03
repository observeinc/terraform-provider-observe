package observe

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

func TestBuildMultiStageQuery_SingleInput(t *testing.T) {
	inputs := map[string]string{
		"test": "o:::dataset:100",
	}
	stages := []fwStageModel{
		{
			Pipeline:    types.StringValue("filter true"),
			Alias:       types.StringNull(),
			Input:       types.StringNull(),
			OutputStage: types.BoolValue(false),
		},
	}

	query, diags := buildMultiStageQuery(inputs, stages)
	if diags.HasError() {
		t.Fatalf("expected no errors, got: %s", diags.Errors())
	}
	if query.OutputStage != "stage-0" {
		t.Fatalf("expected output stage stage-0, got %s", query.OutputStage)
	}
	if len(query.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(query.Stages))
	}
	if query.Stages[0].Pipeline != "filter true" {
		t.Fatalf("expected pipeline 'filter true', got %q", query.Stages[0].Pipeline)
	}
	if len(query.Stages[0].Input) != 1 {
		t.Fatalf("expected 1 input on stage, got %d", len(query.Stages[0].Input))
	}
	if *query.Stages[0].Input[0].DatasetId != "100" {
		t.Fatalf("expected dataset id 100, got %s", *query.Stages[0].Input[0].DatasetId)
	}
}

func TestBuildMultiStageQuery_MultipleStagesWithAlias(t *testing.T) {
	inputs := map[string]string{
		"src": "o:::dataset:200",
	}
	stages := []fwStageModel{
		{
			Alias:       types.StringValue("filtered"),
			Input:       types.StringNull(),
			Pipeline:    types.StringValue("filter true"),
			OutputStage: types.BoolValue(false),
		},
		{
			Alias:       types.StringNull(),
			Input:       types.StringValue("filtered"),
			Pipeline:    types.StringValue("make_col x:1"),
			OutputStage: types.BoolValue(false),
		},
	}

	query, diags := buildMultiStageQuery(inputs, stages)
	if diags.HasError() {
		t.Fatalf("expected no errors, got: %s", diags.Errors())
	}
	if len(query.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(query.Stages))
	}
	if query.OutputStage != "stage-1" {
		t.Fatalf("expected output stage stage-1, got %s", query.OutputStage)
	}
}

func TestBuildMultiStageQuery_NoInputs(t *testing.T) {
	_, diags := buildMultiStageQuery(map[string]string{}, []fwStageModel{
		{Pipeline: types.StringValue("filter true"), Alias: types.StringNull(), Input: types.StringNull(), OutputStage: types.BoolValue(false)},
	})
	if !diags.HasError() {
		t.Fatal("expected error for no inputs")
	}
}

func TestBuildMultiStageQuery_NoStages(t *testing.T) {
	_, diags := buildMultiStageQuery(
		map[string]string{"test": "o:::dataset:100"},
		[]fwStageModel{},
	)
	if !diags.HasError() {
		t.Fatal("expected error for no stages")
	}
}

func TestBuildMultiStageQuery_InvalidInputOID(t *testing.T) {
	_, diags := buildMultiStageQuery(
		map[string]string{"test": "not-an-oid"},
		[]fwStageModel{
			{Pipeline: types.StringValue("filter true"), Alias: types.StringNull(), Input: types.StringNull(), OutputStage: types.BoolValue(false)},
		},
	)
	if !diags.HasError() {
		t.Fatal("expected error for invalid input OID")
	}
}

func TestBuildMultiStageQuery_UnresolvedStageInput(t *testing.T) {
	_, diags := buildMultiStageQuery(
		map[string]string{"a": "o:::dataset:100", "b": "o:::dataset:200"},
		[]fwStageModel{
			{Pipeline: types.StringValue("filter true"), Alias: types.StringNull(), Input: types.StringValue("nonexistent"), OutputStage: types.BoolValue(false)},
		},
	)
	if !diags.HasError() {
		t.Fatal("expected error for unresolved stage input")
	}
}

func TestBuildMultiStageQuery_ExplicitOutputStage(t *testing.T) {
	inputs := map[string]string{"src": "o:::dataset:100"}
	stages := []fwStageModel{
		{Pipeline: types.StringValue("filter true"), Alias: types.StringValue("first"), Input: types.StringNull(), OutputStage: types.BoolValue(true)},
		{Pipeline: types.StringValue("make_col x:1"), Alias: types.StringNull(), Input: types.StringValue("first"), OutputStage: types.BoolValue(false)},
	}

	query, diags := buildMultiStageQuery(inputs, stages)
	if diags.HasError() {
		t.Fatalf("expected no errors, got: %s", diags.Errors())
	}
	if query.OutputStage != "stage-0" {
		t.Fatalf("expected output stage stage-0, got %s", query.OutputStage)
	}
}

func TestFlattenQueryToModel_PreservesExistingInputVersion(t *testing.T) {
	ctx := context.Background()

	stageId := "stage-0"
	dsId := "100"
	gqlStages := []gql.StageQuery{
		{
			Id:       &stageId,
			Pipeline: "filter true",
			Input: []gql.StageQueryInputInputDefinition{
				{InputName: "test", DatasetId: &dsId},
			},
		},
	}

	existingInputs := map[string]string{
		"test": "o:::dataset:100/2024-01-01T00:00:00Z",
	}

	inputs, stages, diags := flattenQueryToModel(ctx, gqlStages, "stage-0", existingInputs, "")
	if diags.HasError() {
		t.Fatalf("expected no errors, got: %s", diags.Errors())
	}

	if inputs["test"] != "o:::dataset:100/2024-01-01T00:00:00Z" {
		t.Fatalf("expected version to be preserved, got %s", inputs["test"])
	}
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(stages))
	}
	if stages[0].Pipeline.ValueString() != "filter true" {
		t.Fatalf("expected pipeline 'filter true', got %q", stages[0].Pipeline.ValueString())
	}
}
