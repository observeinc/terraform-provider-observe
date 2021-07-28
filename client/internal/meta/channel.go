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
		monitors {
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

func (c *Client) CreateChannel(ctx context.Context, workspaceID string, config *ChannelInput) (*Channel, error) {
	result, err := c.Run(ctx, backendChannelFragment+`
	mutation createChannel($workspaceId: ObjectId!, $channel: ChannelInput!) {
		createChannel(workspaceId:$workspaceId, channel: $channel) {
			...channelFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"channel":     config,
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

func (c *Client) UpdateChannel(ctx context.Context, id string, config *ChannelInput) (*Channel, error) {
	result, err := c.Run(ctx, backendChannelFragment+`
	mutation updateChannel($id: ObjectId!, $channel: ChannelInput!) {
		updateChannel(id:$id, channel: $channel) {
			...channelFields
		}
	}`, map[string]interface{}{
		"id":      id,
		"channel": config,
	})
	if err != nil {
		return nil, err
	}

	var ch Channel
	if err = decodeStrict(getNested(result, "updateChannel"), &ch); err != nil {
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

func (c *Client) SetMonitorsForChannel(ctx context.Context, id string, monitors []string) error {
	// endpoint does not accept null, set explicitly to empty list
	if monitors == nil {
		monitors = make([]string, 0)
	}

	result, err := c.Run(ctx, `
	mutation ($channelId: ObjectId!, $monitorIds: [ObjectId!]!) {
	    setMonitorsForChannel(channelId: $channelId, monitorIds: $monitorIds) {
	        success
	        errorMessage
	        detailedInfo
        }
    }`, map[string]interface{}{
		"channelId":  id,
		"monitorIds": monitors,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "setMonitorsForChannel")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
