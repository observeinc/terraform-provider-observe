package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

// TestDatasetConfig starts with a valid dataset and breaks it repeatedly to verify resulting error
func TestDatasetConfig(t *testing.T) {

	// this wil be our reference datasetConfig
	validDatasetConfig := func(t *testing.T) *DatasetConfig {
		jsonData := `{
			"id": "41000000",
			"name": "test",
			"workspace": "4100000",
			"freshness": 6000000,
			"inputs": {
				"input0": { "dataset": "410000" },
				"input1": { "dataset": "410001" }
			},
			"stages": [
				{ "name": "stage0", "input": "input0", "pipeline": "filter true\n" },
				{ "name": "stage1", "pipeline": "filter true\n" },
				{ "input": "input1", "pipeline": "filter true\n" }
			]
		}`
		var config DatasetConfig
		if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
			t.Fatalf("failed to read json: %s", err)
		}
		return &config
	}

	testcases := []struct {
		ErrorType error
		Mutation  func(*testing.T, *DatasetConfig)
	}{
		{
			// verify base case is valid
			Mutation:  func(*testing.T, *DatasetConfig) {},
			ErrorType: nil,
		},
		{
			Mutation:  func(t *testing.T, d *DatasetConfig) { d.Name = "" },
			ErrorType: errNameMissing,
		},
		{
			Mutation:  func(t *testing.T, d *DatasetConfig) { d.Inputs = nil },
			ErrorType: errInputsMissing,
		},
		{
			Mutation:  func(t *testing.T, d *DatasetConfig) { d.Stages = nil },
			ErrorType: errStagesMissing,
		},
		{
			Mutation: func(t *testing.T, d *DatasetConfig) {
				id := "not an id"
				d.Inputs["input0"].Dataset = &id
			},
			ErrorType: errObjectIDInvalid,
		},
		{
			Mutation:  func(t *testing.T, d *DatasetConfig) { d.Stages[0].Pipeline = "" },
			ErrorType: errStagePipelineMissing,
		},
		{
			Mutation:  func(t *testing.T, d *DatasetConfig) { d.Inputs["input0"].Dataset = nil },
			ErrorType: errInputEmpty,
		},
	}

	for i, testcase := range testcases {
		tt := testcase
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			config := validDatasetConfig(t)
			tt.Mutation(t, config)
			if err := config.Validate(); !errors.Is(err, tt.ErrorType) {
				t.Fatalf("error %q does not match %q", err, tt.ErrorType)
			}
		})
	}

}

// TestDatasetConfigToGQL verifies we can convert from Dataset to native GQL inputs
func TestDatasetConfigToGQL(t *testing.T) {
	testcases := []struct {
		InputFile  string
		OutputFile string
	}{
		{
			InputFile:  "testdata/togql/simple.tfjson",
			OutputFile: "testdata/togql/simple.json",
		},
		{
			InputFile:  "testdata/togql/multistage.tfjson",
			OutputFile: "testdata/togql/multistage.json",
		},
		{
			InputFile:  "testdata/togql/refstage.tfjson",
			OutputFile: "testdata/togql/refstage.json",
		},
	}

	for i, testcase := range testcases {
		tt := testcase

		type tuple struct {
			D *meta.DatasetInput         `json:"datasetInput"`
			T *meta.MultiStageQueryInput `json:"queryInput"`
		}

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			var config DatasetConfig
			readJSONFile(t, tt.InputFile, &config)

			datasetInput, transformInput, err := config.toGQL()
			if err != nil {
				t.Fatal(err)
			}

			resultTuple := tuple{D: datasetInput, T: transformInput}

			if *update {
				writeJSONFile(t, tt.InputFile, config)
				writeJSONFile(t, tt.OutputFile, resultTuple)
			}

			var expected tuple
			readJSONFile(t, tt.OutputFile, &expected)
			if s := cmp.Diff(expected, resultTuple); s != "" {
				t.Fatalf("GQL outputs do not match: %s", s)
			}
		})
	}
}

// TestDatasetFromGQL verifies we can convert from native GQL outputs to shim.Dataset
func TestDatasetFromGQL(t *testing.T) {
	testcases := []struct {
		InputFile  string
		OutputFile string
	}{
		{
			InputFile:  "testdata/fromgql/full.json",
			OutputFile: "testdata/fromgql/full.tfjson",
		},
		{
			InputFile:  "testdata/fromgql/example3.json",
			OutputFile: "testdata/fromgql/example3.tfjson",
		},
		{
			InputFile:  "testdata/fromgql/refstage.json",
			OutputFile: "testdata/fromgql/refstage.tfjson",
		},
	}

	for i, testcase := range testcases {
		tt := testcase

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			// Read dataset in, we need to rely on mapstructure
			var dataset meta.Dataset
			var raw map[string]interface{}
			readJSONFile(t, tt.InputFile, &raw)
			if err := dataset.Decode(&raw); err != nil {
				t.Fatal(err)
			}

			result, err := newDataset(&dataset)
			if err != nil {
				t.Fatal(err)
			}

			if *update {
				writeJSONFile(t, tt.InputFile, dataset)
				writeJSONFile(t, tt.OutputFile, result.Config)
			}

			var expected DatasetConfig
			readJSONFile(t, tt.OutputFile, &expected)

			if err := expected.Validate(); err != nil {
				t.Fatalf("expected dataset is invalid: %s", err)
			}

			if s := cmp.Diff(&expected, result.Config); s != "" {
				t.Fatalf("Terraform outputs do not match: %s", s)
			}
		})
	}
}
