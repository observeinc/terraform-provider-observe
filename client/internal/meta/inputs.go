package meta

import (
	"time"
)

type DatasetInput struct {
	ID               *ObjectIdScalar `json:"id,omitempty"`
	Label            string          `json:"label"`
	Deleted          bool            `json:"deleted"`
	LatencyDesired   *string         `json:"latencyDesired"`
	FreshnessDesired *string         `json:"freshnessDesired"`
	Description      *string         `json:"description"`
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

type MultiStageQueryInput struct {
	OutputStage string             `json:"outputStage"`
	Stages      []*StageQueryInput `json:"stages"`
}

type StageQueryInput struct {
	ID       string                 `json:"id"`
	Input    []InputDefinitionInput `json:"input"`
	Pipeline string                 `json:"pipeline"`
}

type InputDefinitionInput struct {
	InputName   string          `json:"inputName"`
	InputRole   *InputRole      `json:"inputRole"`
	DatasetID   *ObjectIdScalar `json:"datasetId,omitempty"`
	DatasetPath *string         `json:"datasetPath"`
	StageID     string          `json:"stageId"`
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

type BookmarkInput struct {
	Name     *string         `json:"name,omitempty"`
	IconURL  *string         `json:"iconUrl,omitempty"`
	TargetID *ObjectIdScalar `json:"targetId,omitempty"`
	GroupID  *ObjectIdScalar `json:"groupId,omitempty"`
}

type ChannelActionInput struct {
	Name        *string `json:"name"`
	IconURL     *string `json:"iconUrl"`
	Description *string `json:"description"`
	//RateLimit   *string `json:"rateLimit"`

	Email   *EmailActionInput   `json:"email"`
	Webhook *WebhookActionInput `json:"webhook"`
}

type EmailActionInput struct {
	//TargetUsers     []UserIdScalar `json:"targetUsers"`
	TargetAddresses []string `json:"targetAddresses"`
	SubjectTemplate *string  `json:"subjectTemplate"`
	BodyTemplate    *string  `json:"bodyTemplate"`
	IsHTML          *bool    `json:"isHtml"`
}

type WebhookActionInput struct {
	URLTemplate  *string          `json:"urlTemplate"`
	Method       *string          `json:"method"`
	Headers      *[]WebhookHeader `json:"headers"`
	BodyTemplate *string          `json:"bodyTemplate"`
}

type WebhookHeader struct {
	Header        string `json:"header"`
	ValueTemplate string `json:"valueTemplate"`
}

type ChannelInput struct {
	Name        string  `json:"name"`
	IconURL     *string `json:"iconUrl"`
	Description *string `json:"description"`
}

type StageInput struct {
	Input        []InputDefinitionInput  `json:"inputs"`
	StageID      string                  `json:"stageId"`
	Pipeline     string                  `json:"pipeline"`
	Presentation *StagePresentationInput `json:"presentation"`
}

type StagePresentationInput struct {
	Limit       *int64        `json:"limit,string"`
	ResultKinds []*ResultKind `json:"resultKinds"`
}

type QueryParams struct {
	StartTime           *time.Time `json:"startTime"`
	EndTime             *time.Time `json:"endTime"`
	ProgressiveInterval int        `json:"progressiveInterval,omitempty"`
	ProgressiveSliceIdx string     `json:"progressiveSliceIdx,omitempty"`
}

type ResultKind string

const (
	ResultKindResultKindSchema   ResultKind = "ResultKindSchema"
	ResultKindResultKindData     ResultKind = "ResultKindData"
	ResultKindResultKindStats    ResultKind = "ResultKindStats"
	ResultKindResultKindSuppress ResultKind = "ResultKindSuppress"
	ResultKindResultKindProgress ResultKind = "ResultKindProgress"
)
