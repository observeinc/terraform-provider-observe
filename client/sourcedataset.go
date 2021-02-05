package client

import (
	"errors"
	"fmt"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type SourceDataset struct {
	ID          string               `json:"id"`
	WorkspaceID string               `json:"workspace_id"`
	Version     string               `json:"version"`
	Config      *SourceDatasetConfig `json:"config"`
}

func (d *SourceDataset) OID() *OID {
	return &OID{
		Type:    TypeDataset,
		ID:      d.ID,
		Version: &d.Version,
	}
}

type SourceDatasetConfig struct {
	Name string `json:"name"`

	// SourceTable configuration
	Schema                string  `json:"schema"`
	TableName             string  `json:"table_name"`
	SourceUpdateTableName *string `json:"source_update_table_name"`
	ValidFromField        *string `json:"valid_from_field"`
	BatchSeqField         *string `json:"batch_seq_field"`
	IsInsertOnly          bool    `json:"isInsertOnly"`

	// Fields configures both the dataset typedef and the sourceTable fields
	Fields []SourceDatasetFieldConfig `json:"fields"`

	// Generic Dataset fields
	Description *string        `json:"description"`
	IconURL     *string        `json:"icon_url"`
	Freshness   *time.Duration `json:"freshness"`
}

type SourceDatasetFieldConfig struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	SqlType      string `json:"sql_type"`
	IsEnum       *bool  `json:"is_enum"`
	IsSearchable *bool  `json:"is_searchable"`
	IsHidden     *bool  `json:"is_hidden"`
	IsConst      *bool  `json:"is_const"`
	IsMetric     *bool  `json:"is_metric"`
}

func newSourceDataset(gqlDataset *meta.Dataset) (*SourceDataset, error) {
	if gqlDataset.SourceTable == nil {
		return nil, errors.New("input GQL dataset has nil 'sourceTable'")
	}
	eds := &SourceDataset{
		ID:          gqlDataset.ID.String(),
		WorkspaceID: gqlDataset.WorkspaceId.String(),
		Version:     gqlDataset.Version,
		Config: &SourceDatasetConfig{
			Name:                  gqlDataset.Label,
			Schema:                gqlDataset.SourceTable.Schema,
			TableName:             gqlDataset.SourceTable.TableName,
			SourceUpdateTableName: gqlDataset.SourceTable.SourceUpdateTableName,
			ValidFromField:        gqlDataset.SourceTable.ValidFromField,
			BatchSeqField:         gqlDataset.SourceTable.BatchSeqField,
			IsInsertOnly:          gqlDataset.SourceTable.IsInsertOnly,
			Description:           gqlDataset.Description,
			IconURL:               gqlDataset.IconURL,
			Freshness:             gqlDataset.FreshnessDesired,
		},
	}

	var fields []SourceDatasetFieldConfig
	fieldDefs := gqlDataset.Typedef.Definition["fields"].([]interface{})
	fieldDefsMap := make(map[string]map[string]interface{})
	for _, d := range fieldDefs {
		defMap := d.(map[string]interface{})
		name := defMap["name"].(string)
		fieldDefsMap[name] = defMap
	}
	for _, col := range gqlDataset.SourceTable.Fields {
		fieldDef := fieldDefsMap[col.Name]
		colType := fieldDef["type"].(map[string]interface{})
		typeRep := colType["rep"].(string)
		fieldCfg := SourceDatasetFieldConfig{
			Name:    col.Name,
			Type:    typeRep,
			SqlType: col.SqlType,
		}

		if isEnum, ok := fieldDef["isEnum"]; ok {
			b := isEnum.(bool)
			fieldCfg.IsEnum = &b
		}
		if isSearchable, ok := fieldDef["isSearchable"]; ok {
			b := isSearchable.(bool)
			fieldCfg.IsSearchable = &b
		}
		if isHidden, ok := fieldDef["isHidden"]; ok {
			b := isHidden.(bool)
			fieldCfg.IsHidden = &b
		}
		if isConst, ok := fieldDef["isConst"]; ok {
			b := isConst.(bool)
			fieldCfg.IsConst = &b
		}
		if isMetric, ok := fieldDef["isMetric"]; ok {
			b := isMetric.(bool)
			fieldCfg.IsMetric = &b
		}
		fields = append(fields, fieldCfg)
	}

	return eds, nil
}

func (f *SourceDatasetFieldConfig) toGQL() (meta.DatasetFieldDefInput, meta.SourceTableFieldDefinitionInput) {
	datasetField := meta.DatasetFieldDefInput{
		Name: f.Name,
		Type: meta.DatasetFieldTypeInput{
			Rep: f.Type,
		},
		IsEnum:       f.IsEnum,
		IsSearchable: f.IsSearchable,
		IsHidden:     f.IsHidden,
		IsConst:      f.IsConst,
		IsMetric:     f.IsMetric,
	}

	tableField := meta.SourceTableFieldDefinitionInput{
		Name:    f.Name,
		SqlType: f.SqlType,
	}

	return datasetField, tableField
}

func (c *SourceDatasetConfig) toGQL() (*meta.DatasetDefinitionInput, *meta.SourceTableDefinitionInput) {
	datasetInput := &meta.DatasetDefinitionInput{
		Dataset: meta.DatasetInput{
			Label:       c.Name,
			Description: c.Description,
			IconURL:     c.IconURL,
		},
	}
	tableInput := &meta.SourceTableDefinitionInput{
		Schema:                c.Schema,
		TableName:             c.TableName,
		SourceUpdateTableName: c.SourceUpdateTableName,
		ValidFromField:        c.ValidFromField,
		BatchSeqField:         c.BatchSeqField,
		IsInsertOnly:          c.IsInsertOnly,
	}

	var datasetFields []meta.DatasetFieldDefInput
	var tableFields []meta.SourceTableFieldDefinitionInput
	for _, fieldCfg := range c.Fields {
		datasetField, tableField := fieldCfg.toGQL()
		datasetFields = append(datasetFields, datasetField)
		tableFields = append(tableFields, tableField)
	}
	datasetInput.Schema = datasetFields
	tableInput.Fields = tableFields

	if c.Freshness != nil {
		i := fmt.Sprintf("%d", c.Freshness.Nanoseconds())
		datasetInput.Dataset.FreshnessDesired = &i
	}

	return datasetInput, tableInput
}
