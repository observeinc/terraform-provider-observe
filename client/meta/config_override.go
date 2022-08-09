package meta

import "context"

type configOverrideResponse0 interface {
	GetLayeredSettings() []LayeredSetting
}

type configOverrideResponse interface {
	GetLayeredSetting() LayeredSetting
}

func configOverrideOrError0(c configOverrideResponse0, err error) (*LayeredSetting, error) {
	if err != nil {
		return nil, err
	}
	result := c.GetLayeredSettings()
	return &result[0], nil
}

func configOverrideOrError(c configOverrideResponse, err error) (*LayeredSetting, error) {
	if err != nil {
		return nil, err
	}
	result := c.GetLayeredSetting()
	return &result, nil
}

func (client *Client) CreateLayeredSetting(ctx context.Context, config *LayeredSettingInput) (*LayeredSetting, error) {
	resp, err := createLayeredSetting(ctx, client.Gql, *config)
	return configOverrideOrError0(resp, err)
}

func (client *Client) UpdateLayeredSetting(ctx context.Context, config *LayeredSettingInput) (*LayeredSetting, error) {
	resp, err := updateLayeredSetting(ctx, client.Gql, *config)
	return configOverrideOrError0(resp, err)
}

func (client *Client) GetLayeredSetting(ctx context.Context, id string) (*LayeredSetting, error) {
	resp, err := getLayeredSetting(ctx, client.Gql, id)
	return configOverrideOrError(resp, err)
}

func (client *Client) DeleteLayeredSetting(ctx context.Context, id string) error {
	resp, err := deleteLayeredSetting(ctx, client.Gql, id)
	rs := resp.GetDeleteLayeredSettings()
	return resultStatusError(&rs, err)
}
