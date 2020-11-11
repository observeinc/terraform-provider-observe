package meta

import (
	"context"
)

var (
	backendChannelFragment = `
	fragment channelFields on Channel {
	    id
	    name
	    iconUrl
		description
		workspaceId
		actions {
			id
		}
	}`
)

func (c *Client) GetChannel(ctx context.Context, id string) (*Channel, error) {
	result, err := c.Run(ctx, backendChannelFragment+`
	query getChannel($id: ObjectId!) {
		getChannel(id: $id) {
			...channelFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var ch Channel
	if err = decodeStrict(getNested(result, "getChannel"), &ch); err != nil {
		return nil, err
	}

	return &ch, nil
}

func (c *Client) CreateChannel(ctx context.Context, workspaceID string, config *ChannelInput, actions []string) (*Channel, error) {
	result, err := c.Run(ctx, backendChannelFragment+`
	mutation createChannel($workspaceId: ObjectId!, $channel: ChannelInput!, $actions: [ObjectId!]) {
		createChannel(workspaceId:$workspaceId, channel: $channel, actions: $actions) {
			...channelFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"channel":     config,
		"actions":     actions,
	})
	if err != nil {
		return nil, err
	}

	var ch Channel
	if err = decodeStrict(getNested(result, "createChannel"), &ch); err != nil {
		return nil, err
	}

	return &ch, nil
}

func (c *Client) UpdateChannel(ctx context.Context, id string, config *ChannelInput, actions []string) (*Channel, error) {
	result, err := c.Run(ctx, backendChannelFragment+`
	mutation updateChannel($id: ObjectId!, $channel: ChannelInput!, $actions: [ObjectId!]) {
		updateChannel(id:$id, channel: $channel, actions: $actions) {
			...channelFields
		}
	}`, map[string]interface{}{
		"id":      id,
		"channel": config,
		"actions": actions,
	})
	if err != nil {
		return nil, err
	}

	var ch Channel
	if err = decodeStrict(getNested(result, "updateChannel"), &ch); err != nil {
		panic(err)
		return nil, err
	}

	return &ch, nil
}

func (c *Client) DeleteChannel(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteChannel(id: $id) {
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
	nested := getNested(result, "deleteChannel")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
