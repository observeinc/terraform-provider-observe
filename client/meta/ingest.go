package meta

import "context"

func (client *Client) GetIngestInfo(ctx context.Context) (*IngestInfo, error) {
	resp, err := getIngestInfo(ctx, client.Gql)
	if err != nil {
		return nil, err
	}

	info := resp.GetIngest().GetIngestInfo()
	return &info, nil
}
