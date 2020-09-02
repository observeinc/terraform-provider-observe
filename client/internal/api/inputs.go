package api

type DatasetInput struct {
	ID               *ObjectIdScalar `json:"id,omitempty"`
	Label            string          `json:"label"`
	Deleted          bool            `json:"deleted"`
	LatencyDesired   *string         `json:"latencyDesired"`
	FreshnessDesired *string         `json:"freshnessDesired"`
	IconURL          *string         `json:"iconUrl"`
	PathCost         *string         `json:"pathCost"`
}

type DeferredDatasetReferenceInput struct {
	DatasetID   *ObjectIdScalar `json:"datasetId,omitempty"`
	DatasetPath *string         `json:"datasetPath,omitempty"`
}

type DeferredForeignKeyInput struct {
	SourceDataset DeferredDatasetReferenceInput `json:"sourceDataset"`
	TargetDataset DeferredDatasetReferenceInput `json:"targetDataset"`
	SrcFields     []string                      `json:"srcFields"`
	DstFields     []string                      `json:"dstFields"`
	Label         *string                       `json:"label,omitempty"`
}

type TransformInput struct {
	QueryLanguage string             `json:"queryLanguage"`
	OutputStage   string             `json:"outputStage"`
	Stages        []*StageQueryInput `json:"stages"`
}

type StageQueryInput struct {
	Input    []InputDefinitionInput `json:"input"`
	StageID  string                 `json:"stageID"`
	Pipeline string                 `json:"pipeline"`
}

type InputDefinitionInput struct {
	InputName   string          `json:"inputName"`
	InputRole   *InputRole      `json:"inputRole"`
	DatasetID   *ObjectIdScalar `json:"datasetId,omitempty"`
	DatasetPath *string         `json:"datasetPath"`
	StageID     string          `json:"stageID"`
}

type DependencyHandlingInput struct {
	SaveMode             SaveMode         `json:"saveMode"`
	IgnoreSpecificErrors []ObjectIdScalar `json:"ignoreSpecificErrors"`
}

type SaveMode string

const (
	SaveModeUpdateDataset                                 = "UpdateDataset"
	SaveModeUpdateDatasetAndDependenciesUnlessNewErrors   = "UpdateDatasetAndDependenciesUnlessNewErrors"
	SaveModeUpdateDatasetAndDependenciesIgnoringAllErrors = "UpdateDatasetAndDependenciesIgnoringAllErrors"
	SaveModePreflightDataset                              = "PreflightDataset"
	SaveModePreflightDatasetAndDependencies               = "PreflightDatasetAndDependencies"
)

type BookmarkGroupInput struct {
	Name         *string                    `json:"name,omitempty"`
	IconURL      *string                    `json:"iconUrl,omitempty"`
	WorkspaceID  *ObjectIdScalar            `json:"workspaceId,omitempty"`
	Presentation *BookmarkGroupPresentation `json:"presentation,omitempty"`
}

// BookmarkGroupPresentation is an int in backend definition, but we'd have to
// convert it from string to int, back into string when serializing to GQL.
// Might as well just define it as a string enum.
type BookmarkGroupPresentation string

const (
	BookmarkGroupPresentationHidden               BookmarkGroupPresentation = "Hidden"
	BookmarkGroupPresentationPerUser              BookmarkGroupPresentation = "PerUser"
	BookmarkGroupPresentationPerUserWorkspace     BookmarkGroupPresentation = "PerUserWorkspace"
	BookmarkGroupPresentationPerCustomerWorkspace BookmarkGroupPresentation = "PerCustomerWorkspace"
)
