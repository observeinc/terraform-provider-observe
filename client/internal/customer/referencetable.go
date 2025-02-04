package customer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

type ReferenceTable struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	IconUrl     *string `json:"iconUrl"`
	Description *string `json:"description"`
	WorkspaceId string  `json:"workspaceId"`
	DatasetID   string  `json:"datasetID"`
}

type ReferenceTableInput struct {
	// UploadFilePath and Schema will be included as files and therefore excluded from the remaining input
	UploadFilePath string                      `json:"-"`
	Schema         []meta.DatasetFieldDefInput `json:"-"`
	PrimaryKey     []string                    `json:"primaryKey"`
	Name           *string                     `json:"name"`
	IconUrl        *string                     `json:"iconUrl"`
	Description    *string                     `json:"description"`
	ManagedById    *string                     `json:"managedById"`
	FolderId       *string                     `json:"folderId"`
	WorkspaceId    string                      `json:"workspaceId"`
}

func (r *ReferenceTable) GetId() string {
	return r.Id
}

type referenceTableResponse interface {
	GetReferenceTable() *ReferenceTable
}

func referenceTableOrError(r referenceTableResponse, err error) (*ReferenceTable, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetReferenceTable()
	return result, nil
}

func (client *Client) CreateReferenceTable(ctx context.Context, input *ReferenceTableInput) (*ReferenceTable, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileName := filepath.Base(input.UploadFilePath)
	uploadPart, err := writer.CreateFormFile("upload", fileName)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(input.UploadFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	_, err = io.Copy(uploadPart, file) // TODO: can we not load the whole file into memory (at once)?
	if err != nil {
		return nil, err
	}

	schema, err := json.Marshal(input.Schema)
	if err != nil {
		return nil, err
	}
	schemaPart, err := writer.CreateFormFile("schema", "schema")
	if err != nil {
		return nil, err
	}
	schemaPart.Write(schema)

	inputData, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	writer.WriteField("input", string(inputData))

	contentType := writer.FormDataContentType()
	writer.Close()

	resp, err := client.httpClient.Post("/v1/meta/reftable", contentType, body)
	if err != nil {
		return nil, err
	}

	// TODO: return reference table

	return nil, nil
}

func (client *Client) GetReferenceTable(ctx context.Context, id string) (*ReferenceTable, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("/v1/meta/reftable/%s", id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ReferenceTable
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func (client *Client) UpdateReferenceTable(ctx context.Context, id string, input *ReferenceTableInput) (*ReferenceTable, error) {
	return nil, nil
	// resp, err := updateReferenceTable(ctx, client.Gql, id, *input)
	// return referenceTableOrError(resp, err)
}

func (client *Client) DeleteReferenceTable(ctx context.Context, id string) error {
	return nil
	// resp, err := deleteReferenceTable(ctx, client.Gql, id)
	// return resultStatusError(resp, err)
}

func (client *Client) LookupReferenceTable(ctx context.Context, workspaceId *string, nameExact *string) (*ReferenceTable, error) {
	params := url.Values{}
	params.Add("name", *nameExact)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", "/v1/meta/reftable", params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ReferenceTable
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func (r *ReferenceTable) Oid() *oid.OID {
	return &oid.OID{
		Id:   r.GetId(),
		Type: oid.TypeReferenceTable,
	}
}
