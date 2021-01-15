package meta

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type NumberScalar float64

func (n *NumberScalar) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%f", *n))
}

func (n *NumberScalar) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*n = NumberScalar(f)
	return nil
}

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

type MonitorInput struct {
	Name             *string                         `json:"name"`
	IconUrl          *string                         `json:"iconUrl"`
	Description      *string                         `json:"description"`
	Query            *MultiStageQueryInput           `json:"query"`
	Rule             *MonitorRuleInput               `json:"rule"`
	NotificationSpec *NotificationSpecificationInput `json:"notificationSpec"`
	Channels         []ObjectIdScalar                `json:"channels"`
}

type MonitorRuleInput struct {
	SourceColumn   *string          `json:"sourceColumn"`
	GroupBy        *MonitorGrouping `json:"groupBy"`
	GroupByColumns []string         `json:"groupByColumns"`

	CountRule  *MonitorRuleCountInput  `json:"countRule,omitempty"`
	ChangeRule *MonitorRuleChangeInput `json:"changeRule,omitempty"`
	//FacetRule     *MonitorRuleFacetInput     `json:"facetRule,omitempty"`
	//ThresholdRule *MonitorRuleThresholdInput `json:"thresholdRule,omitempty"`
}

type MonitorGrouping string

const (
	MonitorGroupingNone       MonitorGrouping = "None"
	MonitorGroupingValue      MonitorGrouping = "Value"
	MonitorGroupingResource   MonitorGrouping = "Resource"
	MonitorGroupingLinkTarget MonitorGrouping = "LinkTarget"
	MonitorGroupingMissing    MonitorGrouping = ""
)

func (mg MonitorGrouping) String() string {
	return string(mg)
}

type NotificationSpecificationInput struct {
	Importance     *NotificationImportance `json:"importance"`
	Merge          *NotificationMerge      `json:"merge"`
	Selection      *NotificationSelection  `json:"selection"`
	SelectionValue NumberScalar            `json:"selectionValue"`
}

type NotificationImportance string

const (
	NotificationImportanceInformational NotificationImportance = "Informational"
	NotificationImportanceImportant     NotificationImportance = "Important"
	NotificationImportanceMissing                              = ""
)

func (ni NotificationImportance) String() string {
	return string(ni)
}

//  Notification merge tells us how to generate notifications
//  when more than one resource triggers the condition -- one notification
//  per resource (separate) or one notification overall?
type NotificationMerge string

const (
	NotificationMergeMerged   NotificationMerge = "Merged"
	NotificationMergeSeparate NotificationMerge = "Separate"
	NotificationMergeMissing  NotificationMerge = ""
)

func (nm NotificationMerge) String() string {
	return string(nm)
}

type NotificationSelection string

const (
	NotificationSelectionAny        NotificationSelection = "Any"
	NotificationSelectionAll        NotificationSelection = "All"
	NotificationSelectionPercentage NotificationSelection = "Percentage"
	NotificationSelectionCount      NotificationSelection = "Count"
	NotificationSelectionMissing    NotificationSelection = ""
)

func (ns NotificationSelection) String() string {
	return string(ns)
}

type MonitorRuleCountInput struct {
	CompareFunction *CompareFunction `json:"compareFunction"`
	CompareValues   []NumberScalar   `json:"compareValues"`
	LookbackTime    *string          `json:"lookbackTime"`
}

type MonitorRuleChangeInput struct {
	ChangeType        *ChangeType        `json:"changeType"`
	CompareFunction   *CompareFunction   `json:"compareFunction"`
	CompareValues     []NumberScalar     `json:"compareValues"`
	AggregateFunction *AggregateFunction `json:"aggregateFunction"`
	LookbackTime      *string            `json:"lookbackTime"`
	BaselineTime      *string            `json:"baselineTime"`
}

type AggregateFunction string

const (
	AggregateFunctionAvg     AggregateFunction = "Avg"
	AggregateFunctionSum     AggregateFunction = "Sum"
	AggregateFunctionMin     AggregateFunction = "Min"
	AggregateFunctionMax     AggregateFunction = "Max"
	AggregateFunctionMissing AggregateFunction = ""
)

func (fn AggregateFunction) String() string {
	return string(fn)
}

type ChangeType string

const (
	ChangeTypeAbsolute ChangeType = "Absolute"
	ChangeTypeRelative ChangeType = "Relative"
	ChangeTypeMissing  ChangeType = ""
)

func (ct ChangeType) String() string {
	return string(ct)
}

type CompareFunction string

const (
	CompareFunctionEqual              CompareFunction = "Equal"
	CompareFunctionNotEqual           CompareFunction = "NotEqual"
	CompareFunctionGreater            CompareFunction = "Greater"
	CompareFunctionGreaterOrEqual     CompareFunction = "GreaterOrEqual"
	CompareFunctionLess               CompareFunction = "Less"
	CompareFunctionLessOrEqual        CompareFunction = "LessOrEqual"
	CompareFunctionBetweenHalfOpen    CompareFunction = "BetweenHalfOpen"
	CompareFunctionNotBetweenHalfOpen CompareFunction = "NotBetweenHalfOpen"
	CompareFunctionIsNull             CompareFunction = "IsNull"
	CompareFunctionIsNotNull          CompareFunction = "IsNotNull"
	CompareFunctionMissing            CompareFunction = ""
)

func (fn CompareFunction) String() string {
	return string(fn)
}
