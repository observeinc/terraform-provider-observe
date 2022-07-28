package meta

// File inputs.go contains a subset of the definitions of the input types in directory
// meta/metatypes of our go monorepo.

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
	ID                            *ObjectIdScalar `json:"id,omitempty"`
	Label                         string          `json:"label"`
	Deleted                       bool            `json:"deleted"`
	LatencyDesired                *string         `json:"latencyDesired"`
	FreshnessDesired              *string         `json:"freshnessDesired"`
	OnDemandMaterializationLength *string         `json:"onDemandMaterializationLength"`
	Description                   *string         `json:"description"`
	IconURL                       *string         `json:"iconUrl"`
	PathCost                      *string         `json:"pathCost"`
	Source                        *string         `json:"source"`
	OverwriteSource               bool            `json:"overwriteSource"`
	ManagedByID                   *ObjectIdScalar `json:"managedById,omitempty"`
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
	Name          *string `json:"name"`
	IconURL       *string `json:"iconUrl"`
	Description   *string `json:"description"`
	NotifyOnClose *bool   `json:"notifyOnClose"`
	RateLimit     *string `json:"rateLimit"`

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
	Name                *string                         `json:"name"`
	IconUrl             *string                         `json:"iconUrl"`
	Description         *string                         `json:"description"`
	Disabled            bool                            `json:"disabled"`
	FreshnessGoal       *string                         `json:"freshnessGoal"`
	UseDefaultFreshness *bool                           `json:"useDefaultFreshness"`
	Source              *string                         `json:"source"`
	OverwriteSource     bool                            `json:"overwriteSource"`
	ManagedByID         *ObjectIdScalar                 `json:"managedById,omitempty"`
	Query               *MultiStageQueryInput           `json:"query"`
	Rule                *MonitorRuleInput               `json:"rule"`
	NotificationSpec    *NotificationSpecificationInput `json:"notificationSpec"`
	Channels            []ObjectIdScalar                `json:"channels"`
}

type MonitorRuleInput struct {
	SourceColumn  *string            `json:"sourceColumn"`
	GroupByGroups []MonitorGroupInfo `json:"groupByGroups"`

	CountRule     *MonitorRuleCountInput     `json:"countRule,omitempty"`
	ChangeRule    *MonitorRuleChangeInput    `json:"changeRule,omitempty"`
	FacetRule     *MonitorRuleFacetInput     `json:"facetRule,omitempty"`
	ThresholdRule *MonitorRuleThresholdInput `json:"thresholdRule,omitempty"`
	PromoteRule   *MonitorRulePromoteInput   `json:"promoteRule,omitempty"`
}

type MonitorGroupInfo struct {
	Columns   []string `json:"columns"`
	GroupName string   `json:"groupName"`
}

type NotificationSpecificationInput struct {
	Importance *NotificationImportance `json:"importance"`
	Merge      *NotificationMerge      `json:"merge"`
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

type MonitorRuleFacetInput struct {
	FacetFunction *FacetFunction `json:"facetFunction"`
	FacetValues   []string       `json:"facetValues"`
	TimeFunction  *TimeFunction  `json:"timeFunction"`
	TimeValue     *NumberScalar  `json:"timeValue,omitempty"`
	LookbackTime  *string        `json:"lookbackTime"`
}

type MonitorRuleThresholdInput struct {
	CompareFunction      *CompareFunction      `json:"compareFunction"`
	CompareValues        []NumberScalar        `json:"compareValues"`
	LookbackTime         *string               `json:"lookbackTime"`
	ThresholdAggFunction *ThresholdAggFunction `json:"thresholdAggFunction"`
}

type MonitorRulePromoteInput struct {
	KindField        *string  `json:"kindField"`
	DescriptionField *string  `json:"descriptionField"`
	PrimaryKey       []string `json:"primaryKey"`
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

type ThresholdAggFunction string

const (
	ThresholdAggFunctionAggFunctionAtAllTimes  ThresholdAggFunction = "AtAllTimes"
	ThresholdAggFunctionAggFunctionAtLeastOnce ThresholdAggFunction = "AtLeastOnce"
	ThresholdAggFunctionAggFunctionOnAverage   ThresholdAggFunction = "OnAverage"
	ThresholdAggFunctionAggFunctionInTotal     ThresholdAggFunction = "InTotal"
	ThresholdAggFunctionAggFunctionMissing     ThresholdAggFunction = ""
)

func (tfn ThresholdAggFunction) String() string {
	return string(tfn)
}

type DatasetDefinitionInput struct {
	Dataset DatasetInput           `json:"dataset"`
	Schema  []DatasetFieldDefInput `json:"schema"`
}

type DatasetFieldDefInput struct {
	Name         string                `json:"name"`
	Type         DatasetFieldTypeInput `json:"type"`
	IsEnum       *bool                 `json:"isEnum,omitempty"`
	IsSearchable *bool                 `json:"isSearchable,omitempty"`
	IsHidden     *bool                 `json:"isHidden,omitempty"`
	IsConst      *bool                 `json:"isConst,omitempty"`
	IsMetric     *bool                 `json:"isMetric,omitempty"`
	Label        *string               `json:"label,omitempty"`
}

type DatasetFieldTypeInput struct {
	Rep      string `json:"rep"`
	Nullable *bool  `json:"nullable,omitempty"`
}

type SourceTableDefinitionInput struct {
	Schema                string                            `json:"schema"`
	TableName             string                            `json:"tableName"`
	Fields                []SourceTableFieldDefinitionInput `json:"fields"`
	ValidFromField        *string                           `json:"validFromField,omitempty"`
	BatchSeqField         *string                           `json:"batchSeqField,omitempty"`
	IsInsertOnly          bool                              `json:"isInsertOnly"`
	SourceUpdateTableName *string                           `json:"sourceUpdateTableName,omitempty"`
}

type SourceTableFieldDefinitionInput struct {
	Name    string `json:"name"`
	SqlType string `json:"sqlType"`
}

type FacetFunction string

const (
	FacetFunctionEquals         FacetFunction = "Equals"
	FacetFunctionNotEqual       FacetFunction = "NotEqual"
	FacetFunctionContains       FacetFunction = "Contains"
	FacetFunctionDoesNotContain FacetFunction = "DoesNotContain"
	FacetFunctionIsNull         FacetFunction = "IsNull"
	FacetFunctionIsNotNull      FacetFunction = "IsNotNull"
	FacetFunctionMissing        FacetFunction = ""
)

func (fn FacetFunction) String() string {
	return string(fn)
}

type TimeFunction string

const (
	TimeFunctionNever                  TimeFunction = "Never"
	TimeFunctionAtLeastOnce            TimeFunction = "AtLeastOnce"
	TimeFunctionAtAllTimes             TimeFunction = "AtAllTimes"
	TimeFunctionAtLeastPercentageTime  TimeFunction = "AtLeastPercentageTime"
	TimeFunctionLessThanPercentageTime TimeFunction = "LessThanPercentageTime"
	TimeFunctionMissing                TimeFunction = ""
)

func (fn TimeFunction) String() string {
	return string(fn)
}

type BoardInput struct {
	Name      *string `json:"name"`
	IsDefault *bool   `json:"isDefault,omitempty"`
	Board     *string `json:"board"`
	Source    *string `json:"source"`
}

type BoardType string

const (
	BoardTypeSet       BoardType = "Set"
	BoardTypeSingleton BoardType = "Singleton"
)

var AllBoardType = []BoardType{
	BoardTypeSet,
	BoardTypeSingleton,
}

func (e BoardType) String() string {
	return string(e)
}

type PollerInput struct {
	Name               string                    `json:"name"`
	Retries            *int64                    `json:"retries,string,omitempty"`
	Interval           *string                   `json:"interval,omitempty"`
	Tags               *string                   `json:"tags,omitempty"`
	DatastreamID       string                    `json:"datastreamId,omitempty"`
	Chunk              *PollerChunkInput         `json:"chunk,omitempty"`
	PubsubConfig       *PollerPubSubInput        `json:"pubsubConfig,omitempty"`
	HTTPConfig         *PollerHTTPInput          `json:"httpConfig,omitempty"`
	GcpConfig          *PollerGCPMonitoringInput `json:"gcpConfig,omitempty"`
	MongoDBAtlasConfig *PollerMongoDBAtlasInput  `json:"mongoDBAtlasConfig,omitempty"`
}

type PollerChunkInput struct {
	Enabled bool   `json:"enabled"`
	Size    *int64 `json:"size,string,omitempty"`
}

type PollerPubSubInput struct {
	ProjectID      string `json:"projectId"`
	JSONKey        string `json:"jsonKey"`
	SubscriptionID string `json:"subscriptionId"`
}

type PollerHTTPInput struct {
	Method      *string              `json:"method"`
	Body        *string              `json:"body"`
	Endpoint    string               `json:"endpoint,omitempty"`
	ContentType string               `json:"contentType,omitempty"`
	Headers     *string              `json:"headers,omitempty"`
	Template    *PollerHTTPRequest   `json:"template,omitempty"`
	Requests    []*PollerHTTPRequest `json:"requests,omitempty"`
	Rules       []*PollerHTTPRule    `json:"rules,omitempty"`
}

type PollerGCPMonitoringInput struct {
	ProjectID                 string   `json:"projectId"`
	JSONKey                   string   `json:"jsonKey"`
	IncludeMetricTypePrefixes []string `json:"includeMetricTypePrefixes"`
	ExcludeMetricTypePrefixes []string `json:"excludeMetricTypePrefixes"`
	RateLimit                 *int64   `json:"rateLimit,string,omitempty"`
	TotalLimit                *int64   `json:"totalLimit,string,omitempty"`
}

type PollerMongoDBAtlasInput struct {
	PublicKey     string   `json:"publicKey"`
	PrivateKey    string   `json:"privateKey"`
	IncludeGroups []string `json:"includeGroups"`
	ExcludeGroups []string `json:"excludeGroups"`
}

type WorkspaceInput struct {
	Label *string `json:"label"`
}

type DatastreamInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IconURL     *string `json:"iconUrl"`
}

type DatastreamTokenInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Disabled    *bool   `json:"disabled"`
}

type WorksheetInput struct {
	ID          *ObjectIdScalar `json:"id"`
	Label       string          `json:"label"`
	WorkspaceID *ObjectIdScalar `json:"workspaceId"`
	// XXX: layout at top level not currently used
	//Layout      map[string]interface{} `json:"layout"`
	Icon        *string                `json:"icon"`
	Stages      []*WorksheetQueryInput `json:"stages"`
	ManagedByID *ObjectIdScalar        `json:"managedById"`
}

func (i *WorksheetInput) SetWorkspaceID(id string) error {
	var o ObjectIdScalar
	if err := o.UnmarshalJSON([]byte(id)); err != nil {
		return fmt.Errorf("failed to unmarshal workspace ID: %w", err)
	}
	i.WorkspaceID = &o
	return nil
}

func (i *WorksheetInput) SetID(id string) error {
	var o ObjectIdScalar
	if err := o.UnmarshalJSON([]byte(id)); err != nil {
		return fmt.Errorf("failed to unmarshal worksheet ID: %w", err)
	}
	i.ID = &o
	return nil
}

type WorksheetQueryInput struct {
	ID       string                  `json:"id"`
	Input    []*InputDefinitionInput `json:"input"`
	Layout   map[string]interface{}  `json:"layout"`
	Pipeline string                  `json:"pipeline"`
}

type DashboardInput struct {
	ID              *ObjectIdScalar          `json:"id"`
	Name            string                   `json:"name"`
	WorkspaceID     *ObjectIdScalar          `json:"workspaceId"`
	Icon            *string                  `json:"iconUrl,omitempty"`
	Stages          []*DashboardStageInput   `json:"stages"`
	ManagedById     *ObjectIdScalar          `json:"managedById"`
	Layout          map[string]interface{}   `json:"layout,omitempty"`
	Parameters      []map[string]interface{} `json:"parameters,omitempty"`
	ParameterValues []map[string]interface{} `json:"parameterValues,omitempty"`
}

func (i *DashboardInput) SetWorkspaceID(id string) error {
	var o ObjectIdScalar
	if err := o.UnmarshalJSON([]byte(id)); err != nil {
		return fmt.Errorf("failed to unmarshal workspace ID: %w", err)
	}
	i.WorkspaceID = &o
	return nil
}

func (i *DashboardInput) SetID(id string) error {
	var o ObjectIdScalar
	if err := o.UnmarshalJSON([]byte(id)); err != nil {
		return fmt.Errorf("failed to unmarshal dashboard ID: %w", err)
	}
	i.ID = &o
	return nil
}

type DashboardStageInput struct {
	ID       string                  `json:"id,omitempty"`
	Input    []*InputDefinitionInput `json:"input"`
	Layout   map[string]interface{}  `json:"layout"`
	Pipeline string                  `json:"pipeline"`
}

type FolderInput struct {
	Name        string  `json:"name"`
	IconURL     *string `json:"iconUrl"`
	Description *string `json:"description"`
}

type AppInput struct {
	ModuleId  string             `json:"moduleId"`
	Version   string             `json:"version"`
	Variables []AppVariableInput `json:"variables"`
	FolderID  *ObjectIdScalar    `json:"folderId"`
}

type AppVariableInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PreferredPathInput struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	FolderID    *ObjectIdScalar `json:"folderId,omitempty"`

	SourceDataset *ObjectIdScalar          `json:"sourceDataset,omitempty"`
	Path          []PreferredPathStepInput `json:"path"`
}

type PreferredPathStepInput struct {
	LinkName           *string         `json:"linkName,omitempty"`
	ReverseFromDataset *ObjectIdScalar `json:"reverseFromDataset,omitempty"`
	LinkId             *ObjectIdScalar `json:"linkId,omitempty"`
	Reverse            bool            `json:"reverse"`
}
