package meta

import (
	"context"
	"fmt"

	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type rbacGroupResponse interface {
	GetRbacGroup() RbacGroup
}

func rbacGroupOrError(r rbacGroupResponse, err error) (*RbacGroup, error) {
	if err != nil {
		return nil, err
	}
	result := r.GetRbacGroup()
	return &result, nil
}

func (client *Client) GetRbacGroup(ctx context.Context, id string) (*RbacGroup, error) {
	resp, err := getRbacGroup(ctx, client.Gql, id)
	return rbacGroupOrError(resp, err)
}

func (client *Client) LookupRbacGroup(ctx context.Context, name string) (*RbacGroup, error) {
	//TODO: refine once there is a better api
	// currently we need to fetch all groups and filter by name
	resp, err := getRbacGroups(ctx, client.Gql)
	if err != nil {
		return nil, err
	}

	var out *RbacGroup
	for i, g := range resp.RbacGroups {
		if g.Name == name {
			out = &resp.RbacGroups[i]
			break
		}
	}
	if out == nil {
		return nil, fmt.Errorf("rbacgroup not found")
	}
	return out, nil
}

func (r *RbacGroup) Oid() *oid.OID {
	rbacGroupOid := oid.RbacGroupOid(r.Id)
	return &rbacGroupOid
}
