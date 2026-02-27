package collect

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Submit data
func (c *Client) Observe(ctx context.Context, s string, body io.Reader, tags map[string]string, options ...func(*http.Request)) error {
	u, _ := url.Parse(c.endpoint)
	u.Path = path.Join(u.Path, s)

	values := make(url.Values)
	for k, v := range tags {
		values.Add(k, v)
	}
	u.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return fmt.Errorf("failed to build new request: %s", err)
	}

	// set defaults before overriding with options
	req.Header.Set("Content-Type", "application/json")

	for _, o := range options {
		o(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("%s", strings.ToLower(http.StatusText(resp.StatusCode)))
	}
}
