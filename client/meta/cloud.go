package meta

import "context"

func (client *Client) GetCloudInfo(ctx context.Context) (*CloudInfo, error) {
	resp, err := getCloudInfo(ctx, client.Gql)
	if err != nil {
		return nil, err
	}

	info := resp.GetCloud().GetCloudInfo()
	return &info, nil
}
