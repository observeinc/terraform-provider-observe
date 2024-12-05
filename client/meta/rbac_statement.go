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

func (client *Client) MutateRbacStatements(ctx context.Context, toCreate []RbacStatementInput, toUpdate []UpdateRbacStatementInput, toDelete []string) (*MutateRbacStatementsResponse, error) {
	resp, err := mutateRbacStatements(ctx, client.Gql, toCreate, toUpdate, toDelete)
	if err != nil {
		return nil, err
	}
	return &resp.MutateRbacStatements, err
}

func (client *Client) GetRbacResourceStatements(ctx context.Context, ids []string) ([]RbacStatement, error) {
	resp, err := getRbacResourceStatements(ctx, client.Gql, ids)
	if err != nil {
		return nil, err
	}
	return resp.RbacResourceStatements, err
}

func (r *RbacStatement) Oid() *oid.OID {
	rbacStatementOid := oid.RbacStatementOid(r.Id)
	return &rbacStatementOid
}
