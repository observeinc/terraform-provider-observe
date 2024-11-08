package meta

import (
	"context"
)

type rbacDefaultGroupResponse interface {
	GetRbacDefaultGroup() RbacGroup
}

func rbacDefaultGroupOrError(r rbacDefaultGroupResponse, err error) (*RbacGroup, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetRbacDefaultGroup()
	return &result, nil
}

func (client *Client) GetRbacDefaultGroup(ctx context.Context) (*RbacGroup, error) {
	resp, err := getRbacDefaultGroup(ctx, client.Gql)
	return rbacDefaultGroupOrError(resp, err)
}

func (client *Client) SetRbacDefaultGroup(ctx context.Context, id string) error {
	resp, err := setRbacDefaultGroup(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (client *Client) UnsetRbacDefaultGroup(ctx context.Context) error {
	resp, err := unsetRbacDefaultGroup(ctx, client.Gql)
	return resultStatusError(resp, err)
}
