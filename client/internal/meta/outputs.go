package meta

// File outputs.go contains copies of the defintions of the input types in directory
// meta/metatypes of our go monorepo.

import (
	"errors"
	"fmt"
	"time"
)

type Workspace struct {
	ID       ObjectIdScalar `json:"id"`
	Label    string         `json:"label"`
	Datasets []*Dataset     `json:"datasets"`
}

type DatasetSaveResult struct {
	Dataset *Dataset `json:"dataset"`
}

type Dataset struct {
	ID               ObjectIdScalar `json:"id"`
	WorkspaceId      ObjectIdScalar `json:"workspaceId"`
	Version          string         `json:"version"`
	Label            string         `json:"label"`
	LatencyDesired   *time.Duration `json:"latencyDesired"`
	FreshnessDesired *time.Duration `json:"freshnessDesired"`
	Typedef          Typedef        `json:"typedef"`
	Description      *string        `json:"description"`
	IconURL          *string        `json:"iconUrl"`
	PathCost         *int64         `json:"pathCost"`
	Transform        *Transform     `json:"transform"`
	SourceTable      *SourceTable   `json:"sourceTable"`
	Source           *string        `json:"source"`
	ForeignKeys      []ForeignKey   `json:"foreignKeys"`
	LastSaved        string         `json:"lastSaved"`
}

func (d *Dataset) Decode(v interface{}) error {
	return decodeStrict(v, d)
}

type Typedef struct {
	Definition map[string]interface{} `json:"definition"`
}

type Transform struct {
	Dataset *Dataset          `json:"dataset"`
	ID      ObjectIdScalar    `json:"id"`
	Current *TransformVersion `json:"current"`
}

type TransformVersion struct {
	Transform *Transform       `json:"transform"`
	Query     *MultiStageQuery `json:"query"`
}

type MultiStageQuery struct {
	OutputStage string        `json:"outputStage"`
	Stages      []*StageQuery `json:"stages"`
}

type StageQuery struct {
	ID       string             `json:"id"`
	Input    []*InputDefinition `json:"input"`
	Pipeline string             `json:"pipeline"`
}

type InputDefinition struct {
	InputName   string          `json:"inputName"`
	InputRole   *InputRole      `json:"inputRole"`
	DatasetID   *ObjectIdScalar `json:"datasetId,omitempty"`
	DatasetPath *string         `json:"datasetPath,omitempty"`
	StageID     string          `json:"stageId,omitempty"`
}

type InputRole string

const (
	InputRoleDefault   InputRole = ""
	InputRoleData      InputRole = "Data"
	InputRoleReference InputRole = "Reference"
)

func (e InputRole) IsValid() bool {
	switch e {
	case InputRoleDefault,
		InputRoleData,
		InputRoleReference:
		return true
	}
	return false
}

func (e InputRole) String() string {
	return string(e)
}

type ResultStatus struct {
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"errorMessage"`
	DetailedInfo map[string]interface{} `json:"detailedInfo"`
}

func (s *ResultStatus) Error() error {
	if s.Success {
		return nil
	}
	if s.ErrorMessage != "" {
		return fmt.Errorf("request failed: %q", s.ErrorMessage)
	}
	return errors.New("request failed")
}

type ForeignKey struct {
	TargetDataset        *int64   `json:"targetDataset,string"`
	TargetStageLabel     *string  `json:"targetStageLabel"`
	Label                *string  `json:"label"`
	TargetLabelFieldName *string  `json:"targetLabelFieldName"`
	SrcFields            []string `json:"srcFields"`
	DstFields            []string `json:"dstFields"`
}

type DeferredForeignKey struct {
	ID            ObjectIdScalar           `json:"id"`
	WorkspaceID   ObjectIdScalar           `json:"workspaceId"`
	SourceDataset DeferredDatasetReference `json:"source"`
	TargetDataset DeferredDatasetReference `json:"target"`
	SrcFields     []string                 `json:"srcFields"`
	DstFields     []string                 `json:"dstFields"`
	Label         *string                  `json:"label,omitempty"`
	Resolution    *ResolvedForeignKey      `json:"resolution,omitempty"`
	Status        DeferredForeignKeyStatus `json:"status"`
}

type ResolvedForeignKey struct {
	SourceID ObjectIdScalar `json:"sourceId"`
	TargetID ObjectIdScalar `json:"targetId"`
}

//  If the foreign key doesn't match some datasets, that may be because of a
//  number of reasons. A "true" in a status field means that that part of the
//  resolution is A-OK, a "false" means that an error happened in that part.
type DeferredForeignKeyStatus struct {
	ID                      ObjectIdScalar `json:"id"`
	FoundSource             bool           `json:"foundSource"`
	FoundTarget             bool           `json:"foundTarget"`
	MatchedSourceFields     bool           `json:"matchedSourceFields"`
	MatchedTargetFields     bool           `json:"matchedTargetFields"`
	FieldTypesAreComparable bool           `json:"fieldTypesAreComparable"`
	//  EnglishError is empty if there's no error, else it's a human-readable
	//  string describing what things prevent the key from resolving.
	ErrorText string `json:"errorText"`
}

type DeferredDatasetReference struct {
	DatasetID   *ObjectIdScalar `json:"datasetId,omitempty"`
	DatasetPath *string         `json:"datasetPath,omitempty"`
}

type BookmarkGroup struct {
	ID           ObjectIdScalar            `json:"id"`
	Name         string                    `json:"name"`
	IconURL      string                    `json:"iconUrl"`
	WorkspaceID  ObjectIdScalar            `json:"workspaceId"`
	Presentation BookmarkGroupPresentation `json:"presentation"`
}

type Bookmark struct {
	ID       ObjectIdScalar `json:"id"`
	Name     string         `json:"name"`
	IconURL  string         `json:"iconUrl"`
	TargetID ObjectIdScalar `json:"targetId"`
	// TODO: use ObjectKind
	TargetIDKind string         `json:"targetIdKind"`
	GroupID      ObjectIdScalar `json:"groupId"`
}

type ChannelAction struct {
	ID            ObjectIdScalar `json:"id"`
	Name          string         `json:"name"`
	IconURL       *string        `json:"iconUrl"`
	Description   *string        `json:"description"`
	WorkspaceId   ObjectIdScalar `json:"workspaceId"`
	NotifyOnClose *bool          `json:"notifyOnClose"`
	RateLimit     *time.Duration `json:"rateLimit"`
	Channels      []struct {
		ID ObjectIdScalar `json:"id"`
	} `json:"channels"`
	//CreatedBy   UserIdScalar   `json:"createdBy"`
	//CreatedDate TimeScalar     `json:"createdDate"`
	//UpdatedBy   UserIdScalar   `json:"updatedBy"`
	//UpdatedDate TimeScalar     `json:"updatedDate"`

	Webhook *WebhookChannelAction `json:"webhook"`
	Email   *EmailChannelAction   `json:"email"`
}

type WebhookChannelAction struct {
	URLTemplate  *string          `json:"urlTemplate"`
	Method       *string          `json:"method"`
	BodyTemplate *string          `json:"bodyTemplate"`
	Headers      []*WebhookHeader `json:"headers"`
}

type EmailChannelAction struct {
	TargetAddresses []string `json:"targetAddresses"`
	SubjectTemplate *string  `json:"subjectTemplate"`
	BodyTemplate    *string  `json:"bodyTemplate"`
	IsHTML          bool     `json:"isHtml"`
}

type Channel struct {
	ID          ObjectIdScalar `json:"id"`
	Name        string         `json:"name"`
	IconURL     *string        `json:"iconUrl"`
	Description *string        `json:"description"`
	WorkspaceId ObjectIdScalar `json:"workspaceId"`
	Monitors    []struct {
		ID ObjectIdScalar `json:"id"`
	} `json:"monitors"`
}

type TaskResult struct {
	QueryID string `json:"queryId"`
	StageID string `json:"stageId"`
	// The time range which this set of results cover.
	StartTime *time.Time `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
	//ResultProgress   *TaskResultProgress
	ResultCursor *SnowflakeCursor `json:"resultCursor"`
	//PaginatedResults *metatypes.PaginatedResults
	ResultKind   *ResultKind       `json:"resultKind"`
	ResultSchema *TaskResultSchema `json:"resultSchema"`
	//ParsedPipeline   *metatypes.ParsedPipeline
	Error *string `json:"error"`
	//EstimatedCost    []CostMetric
}

type TaskResultSchema struct {
	TypedefDefinition struct {
		Fields []map[string]interface{} `json:"fields"`
	} `json:"typedefDefinition"`
}

type SnowflakeCursor struct {
	QueryID       string                   `json:"queryId,omitempty"`
	TotalRowCount int64                    `json:"totalRowCount,omitempty"`
	ColumnDesc    []map[string]interface{} `json:"columnDesc,omitempty"`
	Columns       [][]*string              `json:"columns,omitempty"`
	Chunks        interface{}              `json:"chunks,omitempty"`
	HttpHeaders   map[string]string        `json:"httpHeaders,omitempty"`
	Format        string                   `json:"format,omitempty"`

	ArrowColumnsBase64 string `json:"arrowColumnsBase64,omitempty"`
}

type Monitor struct {
	Id          ObjectIdScalar `json:"id"`
	Name        string         `json:"name"`
	IconUrl     string         `json:"icon_url"`
	Description string         `json:"description"`
	Disabled    bool           `json:"disabled"`
	WorkspaceId ObjectIdScalar `json:"workspaceId"`
	//CreatedBy           UserIdScalar
	//CreatedDate         TimeScalar
	//UpdatedBy           UserIdScalar
	//UpdatedDate         TimeScalar
	//GeneratedDatasetIds []ObjectIdScalar
	//Status        MonitorStatus
	//StatusMessage string

	Query            *MultiStageQuery          `json:"query"`
	NotificationSpec NotificationSpecification `json:"notificationSpec"`

	Rule *MonitorRule `json:"rule"`
}

type MonitorRule struct {
	Type           string                 `mapstructure:"__typename"`
	SourceColumn   *string                `json:"sourceColumn"`
	GroupBy        *MonitorGrouping       `json:"groupBy"`
	GroupByColumns []string               `json:"groupByColumns"`
	Other          map[string]interface{} `mapstructure:",remain"`
}

// DecodeType is a helper to decode to correct struct
func (r *MonitorRule) DecodeType(dst interface{}) error {
	return decodeStrict(r.Other, dst)
}

type MonitorStatus string

const (
	MonitorStatusCreating   MonitorStatus = "Creating"
	MonitorStatusMonitoring MonitorStatus = "Monitoring"
	MonitorStatusStopped    MonitorStatus = "Stopped"
	MonitorStatusMissing    MonitorStatus = ""
)

type NotificationSpecification struct {
	Importance NotificationImportance `json:"importance"`
	Merge      NotificationMerge      `json:"merge"`
	Selection  NotificationSelection  `json:"selection"`
	// NOTE: api will always return a value here :(
	SelectionValue *NumberScalar `json:"selectionValue,omitEmpty"`
}

type MonitorUpdateResult struct {
	Monitor       *Monitor       `json:"monitor"`
	MonitorErrors []string       `json:"monitorErrors"`
	ErrorDatasets []DatasetError `json:"errorDatasets"`
}

type DatasetError struct {
	CustomerId    ObjectIdScalar `json:"customerId"`
	DatasetId     ObjectIdScalar `json:"datasetId"`
	WorkspaceName string         `json:"workspaceName"`
	DatasetName   string         `json:"datasetName"`
	Time          *time.Time     `json:"time"`
	Location      string         `json:"location"`
	Text          []string       `json:"text"`
}

type SourceTable struct {
	Schema                string                       `json:"schema"`
	TableName             string                       `json:"tableName"`
	Fields                []SourceTableFieldDefinition `json:"fields"`
	ValidFromField        *string                      `json:"validFromField,omitempty"`
	BatchSeqField         *string                      `json:"batchSeqField,omitempty"`
	IsInsertOnly          bool                         `json:"isInsertOnly"`
	SourceUpdateTableName *string                      `json:"sourceUpdateTableName,omitempty"`
}

type SourceTableFieldDefinition struct {
	Name    string `json:"name"`
	SqlType string `json:"sqlType"`
}

type Board struct {
	ID        ObjectIdScalar `json:"id"`
	Name      string         `json:"name"`
	DatasetID ObjectIdScalar `json:"datasetId"`
	IsDefault bool           `json:"isDefault"`
	Board     interface{}    `json:"board"`
	Type      BoardType      `json:"type"`
}

type Poller struct {
	ID          ObjectIdScalar `json:"id"`
	WorkspaceId ObjectIdScalar `json:"workspaceId"`
	CustomerID  ObjectIdScalar `json:"customerId"`
	Disabled    bool           `json:"disabled"`
	Kind        string         `json:"kind"`
	Config      PollerConfig   `json:"config"`
}

//TODO: vikramr revisit as needed
type PollerConfig struct {
	Name     string                 `json:"name"`
	Retries  *int64                 `json:"retries"`
	Interval *time.Duration         `json:"interval"`
	Chunk    *PollerChunkConfig     `json:"chunk"`
	Tags     map[string]interface{} `json:"tags"`

	HTTPConfig         *PollerHTTPConfig          `json:"httpConfig"`
	PubSubConfig       *PollerPubSubConfig        `json:"pubsubConfig"`
	GCPConfig          *PollerGCPMonitoringConfig `json:"gcpConfig"`
	MongoDBAtlasConfig *PollerMongoDBAtlasConfig  `json:"mongoDBAtlasConfig"`

	Other map[string]interface{} `mapstructure:",remain"`
}

type PollerChunkConfig struct {
	Enabled bool   `json:"enabled"`
	Size    *int64 `json:"size"`
}

type PollerHTTPConfig struct {
	Endpoint    string                 `json:"endpoint"`
	ContentType string                 `json:"contentType"`
	Headers     map[string]interface{} `json:"headers"`
}

type PollerPubSubConfig struct {
	ProjectID      string                 `json:"projectId"`
	JSONKey        map[string]interface{} `json:"jsonKey"`
	SubscriptionID string                 `json:"subscriptionId"`
}

type PollerGCPMonitoringConfig struct {
	ProjectID                 string                 `json:"projectId"`
	JSONKey                   map[string]interface{} `json:"jsonKey"`
	IncludeMetricTypePrefixes []string               `json:"includeMetricTypePrefixes"`
	ExcludeMetricTypePrefixes []string               `json:"excludeMetricTypePrefixes"`
	RateLimit                 *int64                 `json:"rateLimit"`
	TotalLimit                *int64                 `json:"totalLimit"`
}

type PollerMongoDBAtlasConfig struct {
	PublicKey     string   `json:"publicKey"`
	PrivateKey    string   `json:"privateKey"`
	IncludeGroups []string `json:"includeGroups"`
	ExcludeGroups []string `json:"excludeGroups"`
}

type Datastream struct {
	ID          ObjectIdScalar `json:"id"`
	Name        string         `json:"name"`
	IconURL     *string        `json:"iconUrl"`
	Description *string        `json:"description"`
	WorkspaceId ObjectIdScalar `json:"workspaceId"`
	DatasetId   ObjectIdScalar `json:"datasetId"`
}

type DatastreamToken struct {
	ID           string         `json:"id"`
	DatastreamID ObjectIdScalar `json:"datastreamId"`
	Name         string         `json:"name"`
	Description  *string        `json:"description"`
	Disabled     bool           `json:"disabled"`
	Secret       *string        `json:"secret"`
}

type Worksheet struct {
	ID        ObjectIdScalar         `json:"id"`
	Label     string                 `json:"label"`
	Workspace *Workspace             `json:"workspace"`
	Layout    map[string]interface{} `json:"layout,omitempty"`
	Icon      *string                `json:"icon,omitempty"`
	Queries   []*WorksheetQuery      `json:"queries"`
}

type WorksheetQuery struct {
	ID       string                 `json:"id,omitempty"`
	Input    []*InputDefinition     `json:"input"`
	Params   map[string]interface{} `json:"params,omitempty"`
	Layout   map[string]interface{} `json:"layout,omitempty"`
	Pipeline string                 `json:"pipeline"`
}
