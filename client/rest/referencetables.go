package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// TODO: generate types from OpenAPI spec
type ReferenceTableInput struct {
	Metadata       ReferenceTableMetadataInput `json:"metadata"`
	SourceFilePath string                      `json:"-"`
}

// All fields are nullable + omitempty to support PATCH semantics (excluding field means leave as is)
type ReferenceTableMetadataInput struct {
	Label       *string   `json:"label,omitempty"`
	Description *string   `json:"description,omitempty"`
	PrimaryKey  *[]string `json:"primaryKey,omitempty"`
	LabelField  *string   `json:"labelField,omitempty"`
}

type ReferenceTable struct {
	Id          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Checksum    string `json:"checksum"`
	DatasetId   string `json:"datasetId"`
}

type ReferenceTableListResponse struct {
	TotalCount      int              `json:"totalCount"`
	ReferenceTables []ReferenceTable `json:"referenceTables"`
}

func (r *ReferenceTable) Oid() oid.OID {
	return oid.OID{
		Id:   r.Id,
		Type: oid.TypeReferenceTable,
	}
}

func (r *ReferenceTableInput) RequestBody() (body *bytes.Buffer, contentType string, err error) {
	body = &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileName := filepath.Base(r.SourceFilePath)
	uploadPart, err := writer.CreateFormFile("upload", fileName)
	if err != nil {
		return nil, "", err
	}
	file, err := os.Open(r.SourceFilePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	_, err = io.Copy(uploadPart, file)
	if err != nil {
		return nil, "", err
	}

	metadata, err := json.Marshal(r.Metadata)
	if err != nil {
		return nil, "", err
	}
	writer.WriteField("metadata", string(metadata))

	contentType = writer.FormDataContentType()
	writer.Close()
	return body, contentType, nil
}

func (client *Client) CreateReferenceTable(ctx context.Context, input *ReferenceTableInput) (*ReferenceTable, error) {
	body, contentType, err := input.RequestBody()
	if err != nil {
		return nil, err
	}

	resp, err := client.Post("/v1/referencetables/", contentType, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}
	return refTable, nil
}

func (client *Client) GetReferenceTable(ctx context.Context, id string) (*ReferenceTable, error) {
	resp, err := client.Get("/v1/referencetables/" + id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}

	return refTable, nil
}

func (client *Client) UpdateReferenceTable(ctx context.Context, id string, input *ReferenceTableInput) (*ReferenceTable, error) {
	body, contentType, err := input.RequestBody()
	if err != nil {
		return nil, err
	}
	resp, err := client.Put("/v1/referencetables/"+id, contentType, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}
	return refTable, nil
}

func (client *Client) UpdateReferenceTableMetadata(ctx context.Context, id string, input *ReferenceTableMetadataInput) (*ReferenceTable, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch("/v1/referencetables/"+id, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}
	return refTable, nil
}

func (client *Client) DeleteReferenceTable(ctx context.Context, id string) error {
	resp, err := client.Delete("/v1/referencetables/" + id)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (client *Client) LookupReferenceTable(ctx context.Context, label string) (*ReferenceTable, error) {
	resp, err := client.Get("/v1/referencetables?label=" + label)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTableList := &ReferenceTableListResponse{}
	if err := json.NewDecoder(resp.Body).Decode(refTableList); err != nil {
		return nil, err
	}

	// the API does a substring match, we want an exact match
	var refTable *ReferenceTable
	for _, t := range refTableList.ReferenceTables {
		if t.Label == label {
			refTable = &t
			break
		}
	}

	if refTable == nil {
		return nil, fmt.Errorf("reference table not found")
	}

	return refTable, nil
}
