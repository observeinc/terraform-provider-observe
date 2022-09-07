package meta

import (
	"context"
	//oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type terraformResponse interface {
	GetTerraform() TerraformDefinition
}

func terraformOrError(b terraformResponse, err error) (*TerraformDefinition, error) {
	if err != nil {
		return nil, err
	}
	result := b.GetTerraform()
	return &result, nil
}

func (client *Client) GetTerraform(ctx context.Context, id string, objectType TerraformObjectType) (*TerraformDefinition, error) {
	resp, err := getTerraform(ctx, client.Gql, id, objectType)
	return terraformOrError(resp, err)
}
