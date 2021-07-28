package meta

import (
	"context"
	"fmt"
	"time"
)

var (
	backendChannelActionFragment = `
	fragment channelActionFields on ChannelAction {
	    id
	    name
	    iconUrl
		description
		workspaceId
		channels {
			id
		}

		__typename
		... on WebhookAction {
		  urlTemplate
		  bodyTemplate
		  method
		  headers {
		    header
			valueTemplate
		  }
		}
		... on EmailAction {
		  targetAddresses
		  subjectTemplate
		  bodyTemplate
		  isHtml
		}

	}`
)

// on first pass, unmarshall common fields before deciding how to proceed based on type
type channelAction struct {
	ID          ObjectIdScalar `json:"id"`
	Name        string         `json:"name"`
	IconURL     *string        `json:"iconUrl"`
	Description *string        `json:"description"`
	WorkspaceId ObjectIdScalar `json:"workspaceId"`
	RateLimit   *time.Duration `json:"rateLimit"`
	Channels    []struct {
		ID ObjectIdScalar `json:"id"`
	} `json:"channels"`
	//CreatedBy   UserIdScalar   `json:"createdBy"`
	//CreatedDate TimeScalar     `json:"createdDate"`
	//UpdatedBy   UserIdScalar   `json:"updatedBy"`
	//UpdatedDate TimeScalar     `json:"updatedDate"`

	Type  string                 `mapstructure:"__typename"`
	Other map[string]interface{} `mapstructure:",remain"`
}

// unmarshall specific action attributes into appropriate struct
func (c *channelAction) ToChannelAction() (*ChannelAction, error) {
	var action ChannelAction
	err := decodeLoose(c, &action)
	if err != nil {
		return nil, err
	}

	switch c.Type {
	case "WebhookAction":
		err = decodeStrict(c.Other, &action.Webhook)
	case "EmailAction":
		err = decodeStrict(c.Other, &action.Email)
	default:
		err = fmt.Errorf("unknown action type %s", c.Type)
	}

	return &action, err
}

func (c *Client) GetChannelAction(ctx context.Context, id string) (*ChannelAction, error) {
	result, err := c.Run(ctx, backendChannelActionFragment+`
	query getChannelAction($id: ObjectId!) {
		getChannelAction(id: $id) {
			...channelActionFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var ca channelAction
	if err = decodeStrict(getNested(result, "getChannelAction"), &ca); err != nil {
		return nil, err
	}

	return ca.ToChannelAction()
}

func (c *Client) CreateChannelAction(ctx context.Context, workspaceID string, config *ChannelActionInput) (*ChannelAction, error) {
	result, err := c.Run(ctx, backendChannelActionFragment+`
	mutation createChannelAction($workspaceId: ObjectId!, $action: ActionInput!) {
		createChannelAction(workspaceId:$workspaceId, action: $action) {
			...channelActionFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"action":      config,
	})
	if err != nil {
		return nil, err
	}

	var ca channelAction
	if err = decodeStrict(getNested(result, "createChannelAction"), &ca); err != nil {
		return nil, err
	}

	return ca.ToChannelAction()
}

func (c *Client) UpdateChannelAction(ctx context.Context, id string, config *ChannelActionInput) (*ChannelAction, error) {
	result, err := c.Run(ctx, backendChannelActionFragment+`
	mutation updateChannelAction($id: ObjectId!, $action: ActionInput!) {
		updateChannelAction(id:$id, action: $action) {
			...channelActionFields
		}
	}`, map[string]interface{}{
		"id":     id,
		"action": config,
	})
	if err != nil {
		return nil, err
	}

	var ca channelAction
	if err = decodeStrict(getNested(result, "updateChannelAction"), &ca); err != nil {
		return nil, err
	}

	return ca.ToChannelAction()
}

func (c *Client) DeleteChannelAction(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteChannelAction(id: $id) {
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
	nested := getNested(result, "deleteChannelAction")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}

func (c *Client) SetChannelsForChannelAction(ctx context.Context, id string, channels []string) error {
	// endpoint does not accept null, set explicitly to empty list
	if channels == nil {
		channels = make([]string, 0)
	}

	result, err := c.Run(ctx, `
	mutation ($actionId: ObjectId!, $channelIds: [ObjectId!]!) {
	    setChannelsForChannelAction(actionId: $actionId, channelIds: $channelIds) {
	        success
	        errorMessage
	        detailedInfo
        }
    }`, map[string]interface{}{
		"actionId":   id,
		"channelIds": channels,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "setChannelsForChannelAction")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
