package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ListSharesParams contains parameters for listing shares
type ListSharesParams struct {
	Status       string // Filter by operational status (Pending, Creating, Active, Inactive, Error, Deleting)
	HealthStatus string // Filter by health status (Healthy, Unhealthy, Unknown)
	ProviderType string // Filter by provider type (Snowflake)
	Limit        int    // Maximum number of results (default: server-side default)
	Offset       int    // Number of results to skip
	OrderBy      string // Comma-separated list of fields to order by (e.g., "createdAt,-id")
}

// ListShares lists all external shares imported for the customer
func (c *Client) ListShares(ctx context.Context, params *ListSharesParams) (*ShareListResponse, error) {
	path := "/v1/shares/inbound"

	// Build query parameters
	if params != nil {
		query := url.Values{}
		if params.Status != "" {
			query.Add("status", params.Status)
		}
		if params.HealthStatus != "" {
			query.Add("healthStatus", params.HealthStatus)
		}
		if params.ProviderType != "" {
			query.Add("providerType", params.ProviderType)
		}
		if params.Limit > 0 {
			query.Add("limit", fmt.Sprintf("%d", params.Limit))
		}
		if params.Offset > 0 {
			query.Add("offset", fmt.Sprintf("%d", params.Offset))
		}
		if params.OrderBy != "" {
			query.Add("orderBy", params.OrderBy)
		}

		if len(query) > 0 {
			path = path + "?" + query.Encode()
		}
	}

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ShareListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode share list response: %w", err)
	}

	return &result, nil
}

// GetShare retrieves details for a specific external share
func (c *Client) GetShare(ctx context.Context, shareId string) (*Share, error) {
	path := fmt.Sprintf("/v1/shares/inbound/%s", url.PathEscape(shareId))

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var share Share
	if err := json.NewDecoder(resp.Body).Decode(&share); err != nil {
		return nil, fmt.Errorf("failed to decode share response: %w", err)
	}

	return &share, nil
}

// LookupShare finds a share by exact shareName and providerAccount match
// This is a convenience method that paginates through all shares and filters by exact match on both fields
// Both shareName and providerAccount are required for uniqueness
func (c *Client) LookupShare(ctx context.Context, shareName, providerAccount string) (*Share, error) {
	const pageSize = 100 // Use a larger page size to minimize API calls
	var allCandidates []Share
	offset := 0

	// Paginate through all shares to find candidates by shareName
	// The API doesn't support filtering by shareName, so we must iterate through all shares
	for {
		params := &ListSharesParams{
			Limit:  pageSize,
			Offset: offset,
		}

		result, err := c.ListShares(ctx, params)
		if err != nil {
			return nil, err
		}

		// Find shares by shareName in this page
		// NOTE: The list endpoint returns Share objects which include SnowflakeConfig,
		// but we still need to check provider account
		for _, share := range result.Shares {
			// Check that this is a Snowflake share
			if share.ProviderType != "Snowflake" {
				continue
			}

			// Match on shareName (top-level field is the Snowflake share name)
			if share.ShareName != shareName {
				continue
			}

			allCandidates = append(allCandidates, share)
		}

		// Check if we've seen all shares
		if len(result.Shares) < pageSize {
			// Last page (incomplete page means no more results)
			break
		}

		offset += len(result.Shares)

		// Safety check: If we've processed more shares than the total count, break
		if result.Meta.TotalCount > 0 && offset >= result.Meta.TotalCount {
			break
		}
	}

	// Now get full details for each candidate to check provider account
	// (In case SnowflakeConfig wasn't populated in the list response)
	var matches []Share
	for _, candidate := range allCandidates {
		// Check if we already have full details from list response
		if candidate.SnowflakeConfig != nil && candidate.SnowflakeConfig.ProviderAccount == providerAccount {
			matches = append(matches, candidate)
			continue
		}

		// Otherwise, fetch full details
		fullShare, err := c.GetShare(ctx, candidate.Id)
		if err != nil {
			continue
		}

		// Verify provider account matches
		if fullShare.SnowflakeConfig == nil {
			continue
		}

		if fullShare.SnowflakeConfig.ProviderAccount != providerAccount {
			continue
		}

		matches = append(matches, *fullShare)
	}

	// Validate exactly one match
	if len(matches) == 0 {
		return nil, ErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        fmt.Errorf("share with name %q and provider account %q not found", shareName, providerAccount),
		}
	}
	if len(matches) > 1 {
		// Build helpful error message listing the conflicting share IDs
		shareIDs := make([]string, len(matches))
		for i, share := range matches {
			shareIDs[i] = share.Id
		}
		return nil, ErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Err: fmt.Errorf(
				"multiple shares found with name %q and provider %q. "+
					"Share names may not be unique. Found %d shares with IDs: %v. "+
					"Use the share ID directly instead of name+provider lookup",
				shareName, providerAccount, len(matches), shareIDs),
		}
	}

	return &matches[0], nil
}
