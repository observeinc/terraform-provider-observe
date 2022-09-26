package meta

import "context"

type configOverrideResponse interface {
	GetLayeredSettingRecord() LayeredSettingRecord
}

func configOverrideOrError(c configOverrideResponse, err error) (*LayeredSettingRecord, error) {
	if err != nil {
		return nil, err
	}
	result := c.GetLayeredSettingRecord()
	return &result, nil
}

func (client *Client) CreateLayeredSettingRecord(ctx context.Context, config *LayeredSettingRecordInput) (*LayeredSettingRecord, error) {
	resp, err := createLayeredSettingRecord(ctx, client.Gql, *config)
	return configOverrideOrError(resp, err)
}

func (client *Client) UpdateLayeredSettingRecord(ctx context.Context, config *LayeredSettingRecordInput) (*LayeredSettingRecord, error) {
	resp, err := updateLayeredSettingRecord(ctx, client.Gql, *config)
	return configOverrideOrError(resp, err)
}

func (client *Client) GetLayeredSettingRecord(ctx context.Context, id string) (*LayeredSettingRecord, error) {
	resp, err := getLayeredSettingRecord(ctx, client.Gql, id)
	return configOverrideOrError(resp, err)
}

func (client *Client) DeleteLayeredSettingRecord(ctx context.Context, id string) error {
	resp, err := deleteLayeredSettingRecord(ctx, client.Gql, id)
	rs := resp.GetDeleteLayeredSettingRecord()
	return resultStatusError(&rs, err)
}
