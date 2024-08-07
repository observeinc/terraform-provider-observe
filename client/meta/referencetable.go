package meta

import (
    "encoding/json"
    "fmt"
	"context"
	"io"
    "net/http"
    "net/url"

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

type Upload struct {
	File     io.ReadSeeker
	Filename string
	Size     int64
}

type ReferenceTableInput struct {
	Upload      *Upload                `json:"upload"`
	Schema      []DatasetFieldDefInput `json:"schema"`
	PrimaryKey  []string               `json:"primaryKey"`
	Name        *string                `json:"name"`
	IconUrl     *string                `json:"iconUrl"`
	Description *string                `json:"description"`
	ManagedById *string                `json:"managedById"`
	FolderId    *string                `json:"folderId"`
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

func (client *Client) CreateReferenceTable(ctx context.Context, workspaceId string, input *ReferenceTableInput) (*ReferenceTable, error) {
	return nil, nil
	// resp, err := createReferenceTable(ctx, client.Gql, workspaceId, *input)
	// return referenceTableOrError(resp, err)
}

func (client *Client) GetReferenceTable(ctx context.Context, id string) (*ReferenceTable, error) {
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", client.RestEndpoint(), id), nil)
    if err != nil {
        return nil, err
    }
    resp, err := client.Do(req)
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
    req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", client.RestEndpoint(), params.Encode()), nil)
    if err != nil {
        return nil, err
    }
    resp, err := client.Do(req)
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
