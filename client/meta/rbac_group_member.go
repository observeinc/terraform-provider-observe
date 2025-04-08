package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type rbacGroupmemberResponse interface {
	GetRbacGroupmember() RbacGroupmember
}

func rbacGroupmemberOrError(r rbacGroupmemberResponse, err error) (*RbacGroupmember, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetRbacGroupmember()
	return &result, nil
}

func (client *Client) CreateRbacGroupmember(ctx context.Context, input *RbacGroupmemberInput) (*RbacGroupmember, error) {
	resp, err := createRbacGroupmember(ctx, client.Gql, *input)
	return rbacGroupmemberOrError(resp, err)
}

func (client *Client) GetRbacGroupmember(ctx context.Context, id string) (*RbacGroupmember, error) {
	resp, err := getRbacGroupmember(ctx, client.Gql, id)
	return rbacGroupmemberOrError(resp, err)
}

func (client *Client) DeleteRbacGroupmember(ctx context.Context, id string) error {
	resp, err := deleteRbacGroupmember(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (r *RbacGroupmember) Oid() *oid.OID {
	rbacGroupmemberOid := oid.RbacGroupmemberOid(r.Id)
	return &rbacGroupmemberOid
}
