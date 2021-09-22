package meta

import (
	"context"
)

var (
	backendDatastreamTokenFragment = `
	fragment datastreamTokenFields on DatastreamToken {
	    id
	    name
		description
		disabled
		datastreamId
		secret
	}`
)

func (c *Client) GetDatastreamToken(ctx context.Context, id string) (*DatastreamToken, error) {
	result, err := c.Run(ctx, backendDatastreamTokenFragment+`
	query datastreamToken($id: String!) {
		datastreamToken(id: $id) {
			...datastreamTokenFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var t DatastreamToken
	if err = decodeStrict(getNested(result, "datastreamToken"), &t); err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *Client) CreateDatastreamToken(ctx context.Context, datastreamID string, config *DatastreamTokenInput) (*DatastreamToken, error) {
	result, err := c.Run(ctx, backendDatastreamTokenFragment+`
	mutation createDatastreamToken($datastreamId: ObjectId!, $token: DatastreamTokenInput!) {
		createDatastreamToken(datastreamId:$datastreamId, token: $token) {
			...datastreamTokenFields
		}
	}`, map[string]interface{}{
		"datastreamId": datastreamID,
		"token":        config,
	})
	if err != nil {
		return nil, err
	}

	var t DatastreamToken
	if err = decodeStrict(getNested(result, "createDatastreamToken"), &t); err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *Client) UpdateDatastreamToken(ctx context.Context, id string, config *DatastreamTokenInput) (*DatastreamToken, error) {
	result, err := c.Run(ctx, backendDatastreamTokenFragment+`
	mutation updateDatastreamToken($id: String!, $token: DatastreamTokenInput!) {
		updateDatastreamToken(id:$id, token: $token) {
			...datastreamTokenFields
		}
	}`, map[string]interface{}{
		"id":    id,
		"token": config,
	})
	if err != nil {
		return nil, err
	}

	var t DatastreamToken
	if err = decodeStrict(getNested(result, "updateDatastreamToken"), &t); err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *Client) DeleteDatastreamToken(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: String!) {
        deleteDatastreamToken(id: $id) {
            success
            errorMessage
            detailedInfo
        }
    }`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "deleteDatastreamToken")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
