package meta

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	// TODO: It may be possible to map UserIdScalar to a string in genqlient.yaml and avoid this.
	// But as an artifact of current configuration:
	// `id` read from the api is json encoded.
	// `id` may also be an unquoted string when user supplied in tf manifest.
	// we sanitize the id string before converting into a UserIdScalar
	id = strconv.Quote(strings.Trim(id, "\""))

	var uid types.UserIdScalar
	if err := uid.UnmarshalJSON([]byte(id)); err != nil {
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
				return resp.Customer.Users[i], nil
			}
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (u *User) Oid() *oid.OID {
	userOid := oid.UserOid(u.Id)
	return &userOid
}
