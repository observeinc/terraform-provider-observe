package meta

import (
	"context"
)

var (
	backendDashboardFragment = `
	fragment primitiveValueFields on Value {
		bool
		float64
		int64
		string
	}
	fragment valueFields on Value {
		...primitiveValueFields
		array {
			value {
				# We only allow array elements to be be primitives right now
				...primitiveValueFields
			}
		}
		link {
			datasetId
			primaryKeyValue {
				name
				value {
					# We only allow primary key values to be primitives right now
					...primitiveValueFields
				}
			}
			storedLabel
		}
		datasetref {
			datasetId
			datasetPath
			stageId
		}
	}
	fragment dashboardFields on Dashboard {
		id
		name
		iconUrl
		workspaceId
		managedById
		folderId
		layout
		stages {
			id
			input {
				inputName
				inputRole
				datasetId
				datasetPath
				stageId
			}
			params
			layout
			pipeline
		}
		parameters {
			id
			name
			defaultValue {
				...valueFields
			}
			valueKind {
				type
				keyForDatasetId
				arrayItemType {
					type
					keyForDatasetId
					# We don't support nested arrays; no need to query arrayItemType at this level
				}
			}
		}
		parameterValues {
			id
			value {
				...valueFields
			}
		}
	}`
)

func (c *Client) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	result, err := c.Run(ctx, backendDashboardFragment+`
	query getDashboard($id: ObjectId!) {
		dashboard(id: $id) {
			...dashboardFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var d Dashboard
	if err = decodeStrict(getNested(result, "dashboard"), &d); err != nil {
		return nil, err
	}

	return &d, nil
}

func (c *Client) SaveDashboard(ctx context.Context, config *DashboardInput) (*Dashboard, error) {
	result, err := c.Run(ctx, backendDashboardFragment+`
	mutation saveDashboard($dashboardInput: DashboardInput!) {
		saveDashboard(dash:$dashboardInput) {
			...dashboardFields
		}
	}`, map[string]interface{}{
		"dashboardInput": config,
	})
	if err != nil {
		return nil, err
	}

	var d Dashboard
	if err = decodeStrict(getNested(result, "saveDashboard"), &d); err != nil {
		return nil, err
	}

	return &d, nil
}

func (c *Client) DeleteDashboard(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteDashboard(id: $id) {
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
	nested := getNested(result, "deleteDashboard")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
