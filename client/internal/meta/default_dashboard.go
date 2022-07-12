package meta

import (
	"context"
)

func (c *Client) GetDefaultDashboard(ctx context.Context, dsid string) (*ObjectIdScalar, error) {
	result, err := c.Run(ctx, `
	query getDefaultDashboard($dsid: ObjectId!) {
		defaultDashboard(dsid: $dsid)
	}`, map[string]interface{}{
		"dsid": dsid,
	})
	if err != nil {
		return nil, err
	}

	nested := getNested(result, "defaultDashboard")
	if nested == nil {
		return nil, nil
	}

	var dashid ObjectIdScalar
	if err = decodeStrict(nested, &dashid); err != nil {
		return nil, err
	}

	return &dashid, nil
}

func (c *Client) SetDefaultDashboard(ctx context.Context, dsid string, dashid string) error {
	result, err := c.Run(ctx, `
	mutation setDefaultDashboard($dsid:ObjectId!,$dashid:ObjectId!){
		setDefaultDashboard(dsid:$dsid,dashid:$dashid){
	  		detailedInfo
	  		success
	  		errorMessage
		}
  	}`, map[string]interface{}{
		"dsid":   dsid,
		"dashid": dashid,
	})
	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "setDefaultDashboard")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}

func (c *Client) DeleteDefaultDashboard(ctx context.Context, dsid string) error {
	result, err := c.Run(ctx, `
    mutation ($dsid: ObjectId!) {
        clearDefaultDashboard(dsid: $dsid) {
            success
            errorMessage
            detailedInfo
        }
    }`, map[string]interface{}{
		"dsid": dsid,
	})

	if err != nil {
		return err
	}

	var status ResultStatus
	nested := getNested(result, "clearDefaultDashboard")
	if err := decodeStrict(nested, &status); err != nil {
		return err
	}
	return status.Error()
}
