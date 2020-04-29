package api

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
	IconURL          *string        `json:"iconUrl"`
	Transform        *Transform     `json:"transform"`
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
	Transform   *Transform    `json:"transform"`
	OutputStage string        `json:"outputStage"`
	Stages      []*StageQuery `json:"stages"`
}

type StageQuery struct {
	StageID  string             `json:"stageID"`
	Input    []*InputDefinition `json:"input"`
	Pipeline string             `json:"pipeline"`
}

type InputDefinition struct {
	InputName   string          `json:"inputName"`
	InputRole   *InputRole      `json:"inputRole"`
	DatasetID   *ObjectIdScalar `json:"datasetId,omitempty"`
	DatasetPath *string         `json:"datasetPath"`
	StageID     string          `json:"stageId"`
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
