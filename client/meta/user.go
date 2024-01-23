package meta

import (
	"context"
	"fmt"

	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

type userResponse interface {
	GetUser() *User
}

func userOrError(u userResponse, err error) (*User, error) {
	if err != nil {
		return nil, err
	}
	return u.GetUser(), nil
}

func (client *Client) GetUser(ctx context.Context, id string) (*User, error) {
	uid, err := types.StringToUserIdScalar(id)
	if err != nil {
		return nil, err
	}
	resp, err := getUser(ctx, client.Gql, uid)
	return userOrError(resp, err)
}

func (client *Client) LookupUser(ctx context.Context, email string) (*User, error) {
	resp, err := getCurrentCustomer(ctx, client.Gql)
	if err != nil {
		return nil, err
	}
	if resp.Customer != nil {
		for i, u := range resp.Customer.Users {
			if u.Email == email {
				return &resp.Customer.Users[i], nil
			}
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (u *User) Oid() *oid.OID {
	userOid := oid.UserOid(u.Id)
	return &userOid
}
