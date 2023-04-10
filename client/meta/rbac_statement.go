package meta

import (
	"context"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type rbacStatementResponse interface {
	GetRbacStatement() RbacStatement
}

func rbacStatementOrError(r rbacStatementResponse, err error) (*RbacStatement, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetRbacStatement()
	return &result, nil
}

func (client *Client) CreateRbacStatement(ctx context.Context, input *RbacStatementInput) (*RbacStatement, error) {
	resp, err := createRbacStatement(ctx, client.Gql, *input)
	return rbacStatementOrError(resp, err)
}

func (client *Client) GetRbacStatement(ctx context.Context, id string) (*RbacStatement, error) {
	resp, err := getRbacStatement(ctx, client.Gql, id)
	return rbacStatementOrError(resp, err)
}

func (client *Client) UpdateRbacStatement(ctx context.Context, id string, input *RbacStatementInput) (*RbacStatement, error) {
	resp, err := updateRbacStatement(ctx, client.Gql, id, *input)
	return rbacStatementOrError(resp, err)
}

func (client *Client) DeleteRbacStatement(ctx context.Context, id string) error {
	resp, err := deleteRbacStatement(ctx, client.Gql, id)
	return resultStatusError(resp, err)
}

func (r *RbacStatement) Oid() *oid.OID {
	rbacStatementOid := oid.RbacStatementOid(r.Id)
	return &rbacStatementOid
}
