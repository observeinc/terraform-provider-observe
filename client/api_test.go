package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

type mockGqlClient struct {
	handler func(req *graphql.Request, resp *graphql.Response) error
}

func (m *mockGqlClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	return m.handler(req, resp)
}

func newClientWithMockGql(handler func(req *graphql.Request, resp *graphql.Response) error) *Client {
	return &Client{
		Config: &Config{Flags: map[string]bool{}},
		Meta:   &meta.Client{Gql: &mockGqlClient{handler: handler}},
	}
}

func workspaceListHandler(workspaces []meta.Workspace, err error) func(*graphql.Request, *graphql.Response) error {
	return func(req *graphql.Request, resp *graphql.Response) error {
		if err != nil {
			return err
		}
		payload := map[string]interface{}{
			"workspaces": workspaces,
		}
		b, _ := json.Marshal(payload)
		return json.Unmarshal(b, resp.Data)
	}
}

func TestResolveWorkspaceID_WithOID(t *testing.T) {
	c := &Client{Config: &Config{Flags: map[string]bool{}}}
	wsOID := oid.WorkspaceOid("12345").String()

	id, err := c.ResolveWorkspaceID(context.Background(), wsOID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "12345" {
		t.Fatalf("expected 12345, got %s", id)
	}
}

func TestResolveWorkspaceID_InvalidOID(t *testing.T) {
	c := &Client{Config: &Config{Flags: map[string]bool{}}}

	_, err := c.ResolveWorkspaceID(context.Background(), "not-a-valid-oid")
	if err == nil {
		t.Fatal("expected error for invalid OID")
	}
}

func TestResolveWorkspaceID_AutoResolve(t *testing.T) {
	c := newClientWithMockGql(workspaceListHandler([]meta.Workspace{{Id: "ws-99"}}, nil))

	id, err := c.ResolveWorkspaceID(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "ws-99" {
		t.Fatalf("expected ws-99, got %s", id)
	}
}

func TestResolveWorkspaceID_CachesResult(t *testing.T) {
	calls := 0
	c := newClientWithMockGql(func(req *graphql.Request, resp *graphql.Response) error {
		calls++
		return workspaceListHandler([]meta.Workspace{{Id: "ws-cached"}}, nil)(req, resp)
	})

	for i := 0; i < 3; i++ {
		id, err := c.ResolveWorkspaceID(context.Background(), "")
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if id != "ws-cached" {
			t.Fatalf("call %d: expected ws-cached, got %s", i, id)
		}
	}
	if calls != 1 {
		t.Fatalf("expected 1 ListWorkspaces call, got %d", calls)
	}
}

func TestResolveWorkspaceID_NoWorkspaces(t *testing.T) {
	c := newClientWithMockGql(workspaceListHandler([]meta.Workspace{}, nil))

	_, err := c.ResolveWorkspaceID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty workspace list")
	}
	if got := err.Error(); got != "no workspaces found for customer" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestResolveWorkspaceID_ListError(t *testing.T) {
	c := newClientWithMockGql(workspaceListHandler(nil, fmt.Errorf("network error")))

	_, err := c.ResolveWorkspaceID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error when ListWorkspaces fails")
	}
}

func TestResolveWorkspaceID_OIDBypassesCache(t *testing.T) {
	c := newClientWithMockGql(workspaceListHandler([]meta.Workspace{{Id: "ws-auto"}}, nil))

	// Auto-resolve first to populate the cache
	id, err := c.ResolveWorkspaceID(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "ws-auto" {
		t.Fatalf("expected ws-auto, got %s", id)
	}

	// Explicit OID should return the OID's ID, not the cached value
	wsOID := oid.WorkspaceOid("99999").String()
	id, err = c.ResolveWorkspaceID(context.Background(), wsOID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "99999" {
		t.Fatalf("expected 99999, got %s", id)
	}
}

func TestResolveWorkspaceID_ConcurrentSafety(t *testing.T) {
	calls := 0
	var mu sync.Mutex
	c := newClientWithMockGql(func(req *graphql.Request, resp *graphql.Response) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return workspaceListHandler([]meta.Workspace{{Id: "ws-concurrent"}}, nil)(req, resp)
	})

	var wg sync.WaitGroup
	errs := make([]error, 50)
	ids := make([]string, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ids[idx], errs[idx] = c.ResolveWorkspaceID(context.Background(), "")
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
		if ids[i] != "ws-concurrent" {
			t.Fatalf("goroutine %d: expected ws-concurrent, got %s", i, ids[i])
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected 1 ListWorkspaces call under concurrency, got %d", calls)
	}
}
