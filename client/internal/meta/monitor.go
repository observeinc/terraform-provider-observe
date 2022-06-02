package meta

import (
	"context"
)

var (
	backendMonitorFragment = `
	fragment monitorFields on Monitor {
		workspaceId
		id
		name
		description
		iconUrl
		disabled
		freshnessGoal
		useDefaultFreshness
		source
		query {
			outputStage
			stages {
				id
				pipeline
				input {
					inputName
					inputRole
					datasetId
					datasetPath
					stageId
				}
			}
		}

		rule {
			__typename
			sourceColumn
			groupByGroups {
				groupName
				columns
			}
			... on MonitorRuleCount {
			  compareFunction
			  compareValues
			  lookbackTime
			}
			... on MonitorRuleChange {
			  changeType
			  compareFunction
			  compareValues
			  aggregateFunction
			  lookbackTime
			  baselineTime
			}
			... on MonitorRuleFacet {
			  facetFunction
			  facetValues
			  timeFunction
			  timeValue
			  lookbackTime
			}
			... on MonitorRuleThreshold {
			  compareFunction
			  compareValues
			  lookbackTime
			}
			... on MonitorRulePromote {
			  kindField
			  descriptionField
			  primaryKey
			}
		}

		notificationSpec {
			merge
			importance
		}
	}`
)

// CreateMonitor creates a monitor
func (c *Client) CreateMonitor(ctx context.Context, workspaceID string, m *MonitorInput) (*Monitor, error) {
	result, err := c.Run(ctx, backendMonitorFragment+`
	mutation createMonitor($workspaceId: ObjectId!, $monitor: MonitorInput!) {
		createMonitor(workspaceId:$workspaceId, monitor:$monitor) {
			monitor {
				...monitorFields
			}
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceID,
		"monitor":     m,
	})

	if err != nil {
		return nil, err
	}

	var r MonitorUpdateResult
	err = decodeStrict(getNested(result, "createMonitor"), &r)
	return r.Monitor, err
}

// GetMonitor retrieves monitor.
func (c *Client) GetMonitor(ctx context.Context, id string) (*Monitor, error) {
	result, err := c.Run(ctx, backendMonitorFragment+`
			query getMonitor($id: ObjectId!) {
		        monitor(id: $id) {
		            ...monitorFields
		        }
		    }`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return nil, err
	}

	var m Monitor
	err = decodeStrict(getNested(result, "monitor"), &m)
	return &m, err
}

// LookupMonitor retrieves monitor by name.
func (c *Client) LookupMonitor(ctx context.Context, workspaceId string, name string) (*Monitor, error) {
	result, err := c.Run(ctx, backendMonitorFragment+`
			query lookupMonitor($workspaceId: ObjectId!, $name: String!) {
		        workspace(id: $workspaceId) {
					monitor(name: $name) {
						...monitorFields
					}
				}
		    }`, map[string]interface{}{
		"workspaceId": workspaceId,
		"name":        name,
	})

	if err != nil {
		return nil, err
	}

	var m Monitor
	err = decodeStrict(getNested(result, "workspace", "monitor"), &m)
	return &m, err
}

// UpdateMonitor does what you'd expect
func (c *Client) UpdateMonitor(ctx context.Context, id string, m *MonitorInput) (*Monitor, error) {
	result, err := c.Run(ctx, backendMonitorFragment+`
	mutation createMonitor($id: ObjectId!, $monitor: MonitorInput!) {
		updateMonitor(id:$id, monitor:$monitor) {
			monitor {
				...monitorFields
			}
		}
	}`, map[string]interface{}{
		"id":      id,
		"monitor": m,
	})

	if err != nil {
		return nil, err
	}

	var r MonitorUpdateResult
	err = decodeStrict(getNested(result, "updateMonitor"), &r)
	return r.Monitor, err
}

// DeleteMonitor deletes monitor by ID.
func (c *Client) DeleteMonitor(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
		    mutation ($id: ObjectId!) {
		        deleteMonitor(id: $id) {
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
	nested := getNested(result, "deleteMonitor")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
