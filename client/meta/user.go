package meta

import (
	"context"
	"errors"

	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

// ErrUserNotFound is returned by LookupUser when no user has the given email.
var ErrUserNotFound = errors.New("user not found")

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
	return nil, ErrUserNotFound
}

func (client *Client) ListUsers(ctx context.Context) ([]User, error) {
	resp, err := listUsers(ctx, client.Gql)
	if err != nil {
		return nil, err
	}
	return resp.Users.Users, nil
}

type userMutationResponse interface {
	GetUser() User
}

func userMutationOrError(u userMutationResponse, err error) (*User, error) {
	if err != nil {
		return nil, err
	}
	result := u.GetUser()
	return &result, nil
}

// CreateSSOUser creates a user bound to a single SSO identity (SAML2 or
// GOOGLE) ahead of their first login. Requires admin privilege.
func (client *Client) CreateSSOUser(ctx context.Context, input *UserInput) (*User, error) {
	resp, err := createSSOUser(ctx, client.Gql, *input)
	return userMutationOrError(resp, err)
}

// UpdateUser updates mutable fields of a user. Email cannot be changed.
func (client *Client) UpdateUser(ctx context.Context, id string, input *UserInput) (*User, error) {
	uid, err := types.StringToUserIdScalar(id)
	if err != nil {
		return nil, err
	}
	resp, err := updateUser(ctx, client.Gql, uid, *input)
	return userMutationOrError(resp, err)
}

// SetUserStatus changes a user's status. Disabling requires user:delete,
// re-activating requires user:invite.
func (client *Client) SetUserStatus(ctx context.Context, id string, status UserStatus) (*User, error) {
	return client.UpdateUser(ctx, id, &UserInput{Status: &status})
}

// DisableUser disables a user. There is no delete mutation; this is the
// API's removal path, and it is reversible via EnableUser.
func (client *Client) DisableUser(ctx context.Context, id string) error {
	_, err := client.SetUserStatus(ctx, id, UserStatusUserstatusdisabled)
	return err
}

// EnableUser re-activates a disabled user.
func (client *Client) EnableUser(ctx context.Context, id string) (*User, error) {
	return client.SetUserStatus(ctx, id, UserStatusUserstatusactive)
}

func (client *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	resp, err := currentUser(ctx, client.Gql)
	return userOrError(resp, err)
}

func (u *User) Oid() *oid.OID {
	userOid := oid.UserOid(u.Id)
	return &userOid
}
