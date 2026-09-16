package meta

import (
	"context"
	"testing"

	"github.com/observeinc/terraform-provider-observe/internal/gqltest"
)

// TestDryRunRequestsDematerializationOnlyWhenAsked pins down what each preflight ("dry run")
// save actually asks the backend for.
//
// dematerializedDatasets is the expensive field: the backend computes it by walking the
// transformer graph synchronously, and it decides whether to do that work by looking for the
// field in the response field mask, which it derives from the selection set of the query it was
// sent. A terraform plan issues one preflight per dataset, so a preflight that requests the
// field when no caller reads the answer is pure cost - and re-adding it to the shared
// DatasetDryRunSaveResult fragment would be a silent regression.
//
// Every preflight must still select errorDatasets, both because it carries the validation
// outcome and because GraphQL requires a non-empty selection set.
func TestDryRunRequestsDematerializationOnlyWhenAsked(t *testing.T) {
	// Answers any of the three preflights below; genqlient ignores the fields a given
	// operation did not select.
	const response = `{"data":{"datasetSaveResult":{"errorDatasets":[],"dematerializedDatasets":[]}}}`

	for _, tt := range []struct {
		name                       string
		send                       func(context.Context, *Client) error
		wantOperationName          string
		wantDematerializedDatasets bool
	}{
		{
			// The default preflight. No caller reads a dematerialization list from it.
			name: "dataset",
			send: func(ctx context.Context, client *Client) error {
				_, err := client.SaveDatasetDryRun(ctx, "41000215", &DatasetInput{}, &MultiStageQueryInput{})
				return err
			},
			wantOperationName:          "saveDatasetDryRun",
			wantDematerializedDatasets: false,
		},
		{
			// The log-derived metric resource discards everything but the error, so it
			// must not ask for the list either.
			name: "log derived metric dataset",
			send: func(ctx context.Context, client *Client) error {
				_, err := client.SaveLogDerivedMetricDatasetDryRun(ctx, "41000215", &DatasetInput{}, &LogDerivedMetricDefinitionInput{})
				return err
			},
			wantOperationName:          "saveLogDerivedMetricDatasetDryRun",
			wantDematerializedDatasets: false,
		},
		{
			// Used only for rematerialization_mode = must_skip_rematerialization, which
			// turns a non-empty list into a hard error.
			name: "dataset with rematerialization",
			send: func(ctx context.Context, client *Client) error {
				_, err := client.SaveDatasetDryRunWithRematerialization(ctx, "41000215", &DatasetInput{}, &MultiStageQueryInput{})
				return err
			},
			wantOperationName:          "saveDatasetDryRunWithRematerialization",
			wantDematerializedDatasets: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := gqltest.New(t, response)

			if err := tt.send(context.Background(), &Client{Gql: recorder.Client()}); err != nil {
				t.Fatalf("preflight save: %s", err)
			}

			request := recorder.OnlyRequest()
			if request.OperationName != tt.wantOperationName {
				t.Errorf("sent operation %q, want %q", request.OperationName, tt.wantOperationName)
			}

			fields := request.RequestedFields(t)
			if got := fields["dematerializedDatasets"]; got != tt.wantDematerializedDatasets {
				t.Errorf("requests dematerializedDatasets = %t, want %t\noperation:\n%s",
					got, tt.wantDematerializedDatasets, request.Query)
			}
			if !fields["errorDatasets"] {
				t.Errorf("does not request errorDatasets, so it no longer reports the validation outcome\noperation:\n%s",
					request.Query)
			}
			// All three are selection-set variants of the same backend mutation.
			if !fields["saveDataset"] {
				t.Errorf("does not call saveDataset\noperation:\n%s", request.Query)
			}
		})
	}
}
