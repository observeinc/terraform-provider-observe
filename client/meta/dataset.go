package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

const (
	SaveModeUpdateDataset                                 = SaveMode("UpdateDataset")
	SaveModeUpdateDatasetAndDependenciesUnlessNewErrors   = SaveMode("UpdateDatasetAndDependenciesUnlessNewErrors")
	SaveModeUpdateDatasetAndDependenciesIgnoringAllErrors = SaveMode("UpdateDatasetAndDependenciesIgnoringAllErrors")
	SaveModePreflightDataset                              = SaveMode("PreflightDataset")
	SaveModePreflightDatasetAndDependencies               = SaveMode("PreflightDatasetAndDependencies")
)

type datasetResponse interface {
	GetDataset() *Dataset
}

func datasetOrError(d datasetResponse, err error) (*Dataset, error) {
	if err != nil {
		return nil, err
	}
	return d.GetDataset(), nil
}

func DefaultDependencyHandling() *DependencyHandlingInput {
	mode := SaveModeUpdateDatasetAndDependenciesIgnoringAllErrors
	return &DependencyHandlingInput{SaveMode: &mode}
}

// SaveDataset creates and updates datasets
func (client *Client) SaveDataset(ctx context.Context, workspaceId string, input *DatasetInput, queryInput *MultiStageQueryInput, dependencyHandling *DependencyHandlingInput) (*Dataset, error) {
	resp, err := saveDataset(ctx, client.Gql, workspaceId, *input, *queryInput, dependencyHandling)
	return datasetOrError(resp.Dataset, err)
}

// SaveDatasetDryRun saves a dataset with pre-flight checks - this is useful when rematerialization_mode is set to "skip_rematerialization"
func (client *Client) SaveDatasetDryRun(ctx context.Context, workspaceId string, input *DatasetInput, queryInput *MultiStageQueryInput) ([]DatasetMaterialization, error) {
	saveMode := SaveModePreflightDatasetAndDependencies
	rematerializationMode := RematerializationModeSkiprematerialization
	dependencyHandling := &DependencyHandlingInput{
		SaveMode:              &saveMode,
		RematerializationMode: &rematerializationMode,
	}

	resp, err := saveDataset(ctx, client.Gql, workspaceId, *input, *queryInput, dependencyHandling)
	if err != nil {
		return nil, err
	}
	return resp.Dataset.GetDematerializedDatasets(), nil
}

// GetDataset retrieves dataset.
func (client *Client) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	resp, err := getDataset(ctx, client.Gql, id)
	return datasetOrError(resp, err)
}

// DeleteDataset deletes dataset by ID.
func (client *Client) DeleteDataset(ctx context.Context, id string) error {
	resp, err := deleteDataset(ctx, client.Gql, id, DefaultDependencyHandling())
	return optionalResultStatusError(resp, err)
}

// LookupDataset retrieves dataset by name.
func (client *Client) LookupDataset(ctx context.Context, workspaceId, name string) (*Dataset, error) {
	resp, err := lookupDataset(ctx, client.Gql, workspaceId, name)
	return datasetOrError(resp.Dataset, err)
}

// ListDatasets retrieves all datasets across workspaces. No filtering provided for now.
func (client *Client) ListDatasets(ctx context.Context) (ds []*Dataset, err error) {
	resp, err := listDatasets(ctx, client.Gql)
	if err != nil {
		return nil, err
	}
	result := make([]*Dataset, 0)
	for _, ds := range resp.Datasets {
		for _, d := range ds.Datasets {
			result = append(result, &d)
		}
	}
	return result, nil
}

func (client *Client) ListDatasetsIdNameOnly(ctx context.Context) (ds []*DatasetIdName, err error) {
	resp, err := listDatasetsIdNameOnly(ctx, client.Gql)
	if err != nil {
		return nil, err
	}
	result := make([]*DatasetIdName, 0)
	for _, ds := range resp.Datasets {
		d := ds.Dataset
		result = append(result, &d)
	}
	return result, nil
}

func (client *Client) SaveSourceDataset(ctx context.Context, workspaceId string, input *DatasetDefinitionInput, sourceInput *SourceTableDefinitionInput) (*Dataset, error) {
	resp, err := saveSourceDataset(ctx, client.Gql, workspaceId, *input, *sourceInput, DefaultDependencyHandling())
	return datasetOrError(resp.Dataset, err)
}

func (d *Dataset) Oid() *oid.OID {
	version := d.LastSaved.String()
	return &oid.OID{
		Id:      d.Id,
		Type:    oid.TypeDataset,
		Version: &version,
	}
}
