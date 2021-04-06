package customer

import (
	"context"
)

func (c *Client) Login(ctx context.Context, user, password string) (string, error) {
	var result struct {
		AccessKey string `json:"access_key"`
		Ok        bool   `json:"ok"`
	}

	err := c.do(ctx, "POST", "/v1/login", map[string]interface{}{
		"user_email":    user,
		"user_password": password,
	}, &result)
	if err != nil {
		return "", err
	}

	return result.AccessKey, nil
}
