package meta

import (
	"context"
	"fmt"
)

func (client *Client) GetCloudInfo(ctx context.Context) (*CloudInfo, error) {
	resp, err := getCloudInfo(ctx, client.Gql)
	if err != nil {
		return nil, err
	}

	cloud := resp.GetCloud()
	if cloud == nil || cloud.GetCloudInfo() == nil {
		return nil, fmt.Errorf("cloud info not available")
	}
	return cloud.GetCloudInfo(), nil
}
