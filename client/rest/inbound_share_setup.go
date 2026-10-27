package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// GetSetupInfo returns the Observe consumer account that a share must be
// granted to, and the provider-side setup SQL.
func (c *Client) GetSetupInfo(ctx context.Context) (*SetupInfo, error) {
	resp, err := c.requestWithContext(ctx, http.MethodGet, "/v1/shares/inbound/setup", "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		SetupInfo
		Guidance *SetupGuidance `json:"guidance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode setup info response: %w", err)
	}
	if raw.Guidance == nil {
		return nil, errors.New("setup info response is missing guidance")
	}

	info := raw.SetupInfo
	info.Guidance = *raw.Guidance
	return &info, nil
}

// CreateRegistrationToken issues a single-use secret for the provider account,
// to be set as the share comment before RegisterShare. Issuing a new token
// retires any earlier outstanding one for the same account.
func (c *Client) CreateRegistrationToken(ctx context.Context, req *CreateRegistrationTokenRequest) (*RegistrationToken, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.requestWithContext(ctx, http.MethodPost, "/v1/shares/inbound/registration-tokens", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token RegistrationToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode registration token response: %w", err)
	}
	if token.Token == "" {
		return nil, errors.New("registration token response is missing the token")
	}

	return &token, nil
}
