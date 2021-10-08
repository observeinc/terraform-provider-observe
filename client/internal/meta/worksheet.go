package meta

import (
	"context"
)

var (
	backendWorksheetFragment = `
	fragment worksheetFields on Worksheet {
		id
		label
		icon
		workspace {
			id
		}
		queries {
			id:label
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
	}`
)

func (c *Client) GetWorksheet(ctx context.Context, id string) (*Worksheet, error) {
	result, err := c.Run(ctx, backendWorksheetFragment+`
	query getWorksheet($id: ObjectId!) {
		worksheet(id: $id) {
			...worksheetFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var w Worksheet
	if err = decodeStrict(getNested(result, "worksheet"), &w); err != nil {
		return nil, err
	}

	return &w, nil
}

func (c *Client) SaveWorksheet(ctx context.Context, config *WorksheetInput) (*Worksheet, error) {
	result, err := c.Run(ctx, backendWorksheetFragment+`
	mutation saveWorksheet($worksheetInput: WorksheetInput!) {
		saveWorksheet(wks:$worksheetInput) {
			...worksheetFields
		}
	}`, map[string]interface{}{
		"worksheetInput": config,
	})
	if err != nil {
		return nil, err
	}

	var w Worksheet
	if err = decodeStrict(getNested(result, "saveWorksheet"), &w); err != nil {
		return nil, err
	}

	return &w, nil
}

func (c *Client) DeleteWorksheet(ctx context.Context, id string) error {
	result, err := c.Run(ctx, `
    mutation ($id: ObjectId!) {
        deleteWorksheet(wks: $id) {
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
	nested := getNested(result, "deleteWorksheet")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
