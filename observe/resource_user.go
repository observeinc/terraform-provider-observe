package observe

import (
	"context"
	"errors"

	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

const (
	schemaUserResourceEmailDescription            = "User email. Cannot be changed after creation; changing it forces a new user."
	schemaUserResourceLabelDescription            = "User display name."
	schemaUserResourceIdentityProviderDescription = "SSO identity provider the user logs in with."
	schemaUserResourceIdentityDescription         = "SSO identity the user is matched on at login. For `SAML2` this must equal the NameID the IdP sends; for `GOOGLE` it is the OAuth2 subject."
	schemaUserResourceRoleDescription             = "Legacy user role. Defaults to `reader` when unset. Prefer `observe_rbac_group_member` for access control."
	schemaUserResourceExpirationTimeDescription   = "Time (RFC3339) at which the user expires."
	schemaUserResourceDisabledDescription         = "Whether the user is disabled. Destroying the resource also disables the user."
	schemaUserResourceStatusDescription           = "Current user status as reported by Observe."
)

var userIdentityProviders = []string{
	string(gql.SSOIdentityProviderSaml2),
	string(gql.SSOIdentityProviderGoogle),
}

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an Observe user bound to an SSO identity, so the user can be provisioned " +
			"before their first login. Observe has no user delete operation: destroying this resource disables the user. " +
			"Requires an admin token; disabling needs `user:delete` and re-enabling needs `user:invite`.",
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceUserImport,
		},
		CustomizeDiff: resourceUserCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: schemaUserResourceEmailDescription,
			},
			"label": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaUserResourceLabelDescription,
			},
			"identity_provider": {
				Type:     schema.TypeString,
				Required: true,

				ValidateDiagFunc: validateStringInSlice(userIdentityProviders, false),
				Description:      fmt.Sprintf("%s Accepted values: `%s`.", schemaUserResourceIdentityProviderDescription, strings.Join(userIdentityProviders, "`, `")),
			},
			"identity": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaUserResourceIdentityDescription,
			},
			"role": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: schemaUserResourceRoleDescription,
			},
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaUserCommentDescription,
			},
			"expiration_time": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimestamp,
				Description:      schemaUserResourceExpirationTimeDescription,
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: schemaUserResourceDisabledDescription,
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaUserOIDDescription,
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaUserResourceStatusDescription,
			},
		},
	}
}

func userSsoIdentityInput(data *schema.ResourceData) []gql.SSOIdentityInput {
	return []gql.SSOIdentityInput{{
		IdentityProvider: gql.SSOIdentityProvider(data.Get("identity_provider").(string)),
		Identity:         data.Get("identity").(string),
	}}
}

func userExpirationTime(data *schema.ResourceData) (*types.TimeScalar, error) {
	v, ok := data.GetOk("expiration_time")
	if !ok {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v.(string))
	if err != nil {
		return nil, err
	}
	ts := types.TimeScalar(t)
	return &ts, nil
}

func newUserCreateInput(data *schema.ResourceData) (*gql.UserInput, diag.Diagnostics) {
	input := &gql.UserInput{
		Email:         stringPtr(data.Get("email").(string)),
		Label:         stringPtr(data.Get("label").(string)),
		SsoIdentities: userSsoIdentityInput(data),
	}
	if v, ok := data.GetOk("role"); ok {
		input.Role = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("comment"); ok {
		input.Comment = stringPtr(v.(string))
	}
	exp, err := userExpirationTime(data)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	input.ExpirationTime = exp
	return input, nil
}

// newUserUpdateInput only sets changed fields: updateUser rejects email, and
// several fields need different privileges, so we never resend unchanged ones.
// The second return is false when nothing besides status changed.
func newUserUpdateInput(data *schema.ResourceData) (*gql.UserInput, bool, diag.Diagnostics) {
	input := &gql.UserInput{}
	changed := false
	if data.HasChange("label") {
		input.Label = stringPtr(data.Get("label").(string))
		changed = true
	}
	if data.HasChange("identity") || data.HasChange("identity_provider") {
		input.SsoIdentities = userSsoIdentityInput(data)
		changed = true
	}
	if data.HasChange("role") {
		if v, ok := data.GetOk("role"); ok {
			input.Role = stringPtr(v.(string))
			changed = true
		}
	}
	if data.HasChange("comment") {
		input.Comment = stringPtr(data.Get("comment").(string))
		changed = true
	}
	if data.HasChange("expiration_time") {
		exp, err := userExpirationTime(data)
		if err != nil {
			return nil, false, diag.FromErr(err)
		}
		// Removal is rejected at plan time by resourceUserCustomizeDiff; if a nil
		// gets here anyway, skip it rather than block the other changes.
		if exp != nil {
			input.ExpirationTime = exp
			changed = true
		}
	}
	return input, changed, nil
}

// resourceUserCustomizeDiff rejects removing expiration_time at plan time:
// UserInput fields are omitempty, so null cannot be sent to clear it.
func resourceUserCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	if d.Id() == "" || !d.HasChange("expiration_time") {
		return nil
	}
	if old, new := d.GetChange("expiration_time"); old.(string) != "" && new.(string) == "" {
		return fmt.Errorf("removing expiration_time is not supported; keep the current value (%s) or set a new one", old)
	}
	return nil
}

func resourceUserCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newUserCreateInput(data)
	if diags.HasError() {
		return diags
	}

	// There is no user delete, so a destroyed (or replaced) user still exists
	// as disabled. Adopt it rather than creating a second user for one email.
	existing, err := client.LookupUser(ctx, data.Get("email").(string))
	switch {
	case err == nil && existing.Status == gql.UserStatusUserstatusdisabled:
		data.SetId(existing.Id.String())
		// updateUser rejects email, and it is unchanged anyway.
		input.Email = nil
		if _, err := client.UpdateUser(ctx, data.Id(), input); err != nil {
			return diag.Errorf("failed to update existing disabled user %s: %s", data.Id(), err.Error())
		}
		if !data.Get("disabled").(bool) {
			if _, err := client.EnableUser(ctx, data.Id()); err != nil {
				return diag.Errorf("failed to re-enable existing user %s: %s", data.Id(), err.Error())
			}
		}
		return append(diags, resourceUserRead(ctx, data, meta)...)
	case err == nil:
		return diag.Errorf("user with email %q already exists (id %s); import it instead", existing.Email, existing.Id.String())
	case !errors.Is(err, gql.ErrUserNotFound):
		return diag.Errorf("failed to check for existing user: %s", err.Error())
	}

	result, err := client.CreateSSOUser(ctx, input)
	if err != nil {
		return diag.Errorf("failed to create user: %s", err.Error())
	}
	data.SetId(result.Id.String())

	if data.Get("disabled").(bool) {
		// Don't fail Create: that taints a user that exists, and the replacement
		// would collide with it. Read records disabled = false, so the next plan
		// retries only the disable.
		if err := client.DisableUser(ctx, data.Id()); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "user created but failed to disable",
				Detail:   err.Error(),
			})
		}
	}
	return append(diags, resourceUserRead(ctx, data, meta)...)
}

func resourceUserRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	u, err := client.GetUser(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read user: %s", err.Error())
	}
	if u == nil || u.Status == gql.UserStatusUserstatusdeleted {
		data.SetId("")
		return nil
	}
	return userResourceToResourceData(u, data)
}

func userResourceToResourceData(u *gql.User, data *schema.ResourceData) (diags diag.Diagnostics) {
	set := func(k string, v interface{}) {
		if err := data.Set(k, v); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	set("email", u.Email)
	set("label", u.Label)
	set("role", u.Role)
	set("status", string(u.Status))
	set("disabled", u.Status == gql.UserStatusUserstatusdisabled)
	set("oid", u.Oid().String())

	if u.Comment != nil {
		set("comment", *u.Comment)
	} else {
		set("comment", nil)
	}
	if u.ExpirationTime != nil {
		set("expiration_time", u.ExpirationTime.String())
	} else {
		set("expiration_time", nil)
	}
	if len(u.SsoIdentities) > 0 {
		set("identity_provider", string(u.SsoIdentities[0].IdentityProvider))
		set("identity", u.SsoIdentities[0].Identity)
	}
	return diags
}

func resourceUserUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, changed, diags := newUserUpdateInput(data)
	if diags.HasError() {
		return diags
	}
	if changed {
		if _, err := client.UpdateUser(ctx, data.Id(), input); err != nil {
			return diag.Errorf("failed to update user: %s", err.Error())
		}
	}

	// Status changes go separately: they need user:delete / user:invite
	// rather than the admin privilege the other fields use.
	if data.HasChange("disabled") {
		var err error
		if data.Get("disabled").(bool) {
			err = client.DisableUser(ctx, data.Id())
		} else {
			_, err = client.EnableUser(ctx, data.Id())
		}
		if err != nil {
			return diag.Errorf("failed to change user status: %s", err.Error())
		}
	}
	return append(diags, resourceUserRead(ctx, data, meta)...)
}

func resourceUserDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	// No delete mutation exists for users; disabling is the removal path.
	// Decide from live status: state may be stale (-refresh=false, saved plans).
	u, err := client.GetUser(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			return nil
		}
		return diag.Errorf("failed to read user: %s", err.Error())
	}
	if u == nil || u.Status == gql.UserStatusUserstatusdisabled || u.Status == gql.UserStatusUserstatusdeleted {
		return nil
	}
	if err := client.DisableUser(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to disable user: %s", err.Error())
	}
	return nil
}

// resourceUserImport accepts either a user ID or an email address, so users
// that were already JIT-provisioned by a login can be brought under management.
func resourceUserImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if !strings.Contains(data.Id(), "@") {
		return []*schema.ResourceData{data}, nil
	}
	client := meta.(*observe.Client)
	u, err := client.LookupUser(ctx, data.Id())
	if err != nil {
		return nil, fmt.Errorf("failed to look up user %q: %w", data.Id(), err)
	}
	data.SetId(u.Id.String())
	return []*schema.ResourceData{data}, nil
}
