package meta

import (
	"context"
)

var (
	backendDatastreamFragment = `
	fragment datastreamFields on Datastream {
	    id
	    name
	    iconUrl
		description
		workspaceId
		datasetId
	}`
)

func (c *Client) GetDatastream(ctx context.Context, id string) (*Datastream, error) {
	result, err := c.Run(ctx, backendDatastreamFragment+`
	query datastream($id: ObjectId!) {
		datastream(id: $id) {
			...datastreamFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var ch Datastream
	if err = decodeStrict(getNested(result, "datastream"), &ch); err != nil {
		return nil, err
	}

	return &ch, nil
}

func (c *Client) CreateDatastream(ctx context.Context, workspaceID string, config *DatastreamInput) (*Datastream, error) {
	result, err := c.Run(ctx, backendDatastreamFragment+`
	mutation createDatastream($workspaceId: ObjectId!, $datastream: DatastreamInput!) {
		createDatastream(workspaceId:$workspaceId, datastream: $datastream) {
			...datastreamFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"datastream":  config,
	})
	if err != nil {
		return nil, err
	}

	var ch Datastream
	if err = decodeStrict(getNested(result, "createDatastream"), &ch); err != nil {
		return nil, err
	}

	return &ch, nil
}

func (c *Client) UpdateDatastream(ctx context.Context, id string, config *DatastreamInput) (*Datastream, error) {
	result, err := c.Run(ctx, backendDatastreamFragment+`
	mutation updateDatastream($id: ObjectId!, $datastream: DatastreamInput!) {
		updateDatastream(id:$id, datastream: $datastream) {
			...datastreamFields
		}
	}`, map[string]interface{}{
		"id":         id,
		"datastream": config,
	})
	if err != nil {
		return nil, err
	}

	var ch Datastream
	if err = decodeStrict(getNested(result, "updateDatastream"), &ch); err != nil {
		return nil, err
	}

	return &ch, nil
}

func (c *Client) DeleteDatastream(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteDatastream(id: $id) {
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
	nested := getNested(result, "deleteDatastream")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}

// LookupDatastream retrieves datastream by name.
func (c *Client) LookupDatastream(ctx context.Context, workspaceId, name string) (*Datastream, error) {
	result, err := c.Run(ctx, backendDatastreamFragment+`
	query lookupDatastream($workspaceId: ObjectId!, $name: String!) {
		workspace(id: $workspaceId) {
			datastream(name: $name) {
            	...datastreamFields
        	}
		}
    }`, map[string]interface{}{
		"workspaceId": workspaceId,
		"name":        name,
	})

	if err != nil {
		return nil, err
	}

	var datastream Datastream
	if err := decodeStrict(getNested(result, "workspace", "datastream"), &datastream); err != nil {
		return nil, err
	}
	return &datastream, nil
}
