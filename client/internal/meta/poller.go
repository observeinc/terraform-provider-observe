package meta

import (
	"context"
	"fmt"
)

var (
	backendPollerFragment = `
    fragment pollerFields on Poller {
        id
        workspaceId
        customerId
		datastreamId
        disabled
        kind
		config {
			name
			retries
			interval
			tags
			chunk {
			  enabled
			  size
			}
			... on PollerPubSubConfig {
				projectId
				jsonKey
				subscriptionId
			}
			... on PollerHTTPConfig {
				method
				body
				endpoint
				contentType
				headers
			}
			... on PollerGCPMonitoringConfig {
				projectId
				jsonKey
				includeMetricTypePrefixes
				excludeMetricTypePrefixes
				rateLimit
				totalLimit
			}
			... on PollerMongoDBAtlasConfig {
				publicKey
				privateKey
				includeGroups
				excludeGroups
			}
		}
	}`
)

// pubsub or http config is returned inline in the GQL output
// json decoding dumps these into an "Other" mapstructure
// this function extracts those fields and sets the relevant HTTPConfig or PubsubConfig
func sanitizePoller(in *Poller) (*Poller, error) {
	var err error
	switch in.Kind {
	case "PubSub":
		err = decodeStrict(in.Config.Other, &in.Config.PubSubConfig)
	case "HTTP":
		err = decodeStrict(in.Config.Other, &in.Config.HTTPConfig)
	case "GCPMonitoring":
		err = decodeStrict(in.Config.Other, &in.Config.GCPConfig)
	case "MongoDBAtlas":
		err = decodeStrict(in.Config.Other, &in.Config.MongoDBAtlasConfig)
	default:
		err = fmt.Errorf("unknown kind: %s", in.Kind)
	}
	in.Config.Other = nil
	return in, err
}

func (c *Client) GetPoller(ctx context.Context, id string) (*Poller, error) {
	result, err := c.Run(ctx, backendPollerFragment+`
	query poller($id: ObjectId!) {
		poller(id: $id) {
			...pollerFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}
	var p Poller
	if err = decodeStrict(getNested(result, "poller"), &p); err != nil {
		return nil, err
	}
	return sanitizePoller(&p)
}

func (c *Client) CreatePoller(ctx context.Context, workspaceID string, config *PollerInput) (*Poller, error) {
	result, err := c.Run(ctx, backendPollerFragment+`
    mutation createPoller($workspaceId: ObjectId!, $poller: PollerInput!) {
		createPoller(workspaceId:$workspaceId, poller: $poller) {
			...pollerFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"poller":      config,
	})
	if err != nil {
		return nil, err
	}
	var p Poller
	if err = decodeStrict(getNested(result, "createPoller"), &p); err != nil {
		return nil, err
	}
	return sanitizePoller(&p)
}

func (c *Client) UpdatePoller(ctx context.Context, id string, config *PollerInput) (*Poller, error) {
	result, err := c.Run(ctx, backendPollerFragment+`
    mutation updatePoller($id: ObjectId!, $poller: PollerInput!) {
		updatePoller(id:$id, poller: $poller) {
			...pollerFields
		}
	}`, map[string]interface{}{
		"id":     id,
		"poller": config,
	})
	if err != nil {
		return nil, err
	}
	var p Poller
	if err = decodeStrict(getNested(result, "updatePoller"), &p); err != nil {
		return nil, err
	}
	return sanitizePoller(&p)
}

func (c *Client) DeletePoller(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
	mutation ($id: ObjectId!) {
		deletePoller(id: $id) {
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
	nested := getNested(result, "deletePoller")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
