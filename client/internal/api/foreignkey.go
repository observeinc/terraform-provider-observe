package api

var (
	backendForeignKeyFragment = `
	fragment foreignKeyFields on DeferredForeignKey {
	    id
		workspaceId
		sourceDataset { datasetId }
		targetDataset { datasetId }
		srcFields
		dstFields
		label
		resolution { sourceId targetId }
		status { errorText }
	}`
)

func (c *Client) GetDeferredForeignKey(id string) (*DeferredForeignKey, error) {
	result, err := c.Run(backendForeignKeyFragment+`
	query getDeferredForeignKey($id: ObjectId!) {
		deferredForeignKey(id: $id) {
			...foreignKeyFields
		}
	}`, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var dfk DeferredForeignKey
	if err = decodeStrict(getNested(result, "deferredForeignKey"), &dfk); err != nil {
		return nil, err
	}

	return &dfk, nil
}

func (c *Client) CreateDeferredForeignKey(workspaceid string, config *DeferredForeignKeyInput) (*DeferredForeignKey, error) {
	result, err := c.Run(backendForeignKeyFragment+`
	mutation createDeferredForeignKey($workspaceId: ObjectId!, $data: DeferredForeignKeyInput!) {
		createDeferredForeignKey(workspaceId:$workspaceId, data: $data) {
			...foreignKeyFields
		}
	}`, map[string]interface{}{
		"workspaceId": workspaceid,
		"data":        config,
	})
	if err != nil {
		return nil, err
	}

	var dfk DeferredForeignKey
	if err = decodeStrict(getNested(result, "createDeferredForeignKey"), &dfk); err != nil {
		return nil, err
	}

	return &dfk, nil
}

func (c *Client) UpdateDeferredForeignKey(id string, config *DeferredForeignKeyInput) (*DeferredForeignKey, error) {
	result, err := c.Run(backendForeignKeyFragment+`
	mutation updateDeferredForeignKey($id: ObjectId!, $data: DeferredForeignKeyInput!) {
		updateDeferredForeignKey(id:$id, data: $data) {
			...foreignKeyFields
		}
	}`, map[string]interface{}{
		"id":   id,
		"data": config,
	})
	if err != nil {
		return nil, err
	}

	var dfk DeferredForeignKey
	if err = decodeStrict(getNested(result, "updateDeferredForeignKey"), &dfk); err != nil {
		return nil, err
	}

	return &dfk, nil
}

// DeleteDeferredForeignKey deletes dfk by ID.
func (c *Client) DeleteDeferredForeignKey(id string) error {
	result, err := c.Run(`
    mutation ($id: ObjectId!) {
		deleteDeferredForeignKey(id: $id) {
        	success
            errorMessage
        }
    }`, map[string]interface{}{
		"id": id,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "deleteDeferredForeignKey")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
