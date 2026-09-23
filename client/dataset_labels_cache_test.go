package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"

	"github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

// labelServer serves /v1/datasets `id in [...]` lookups over visible and
// records each request's IDs. block, when set, holds requests it matches
// until release is closed or the request is canceled.
type labelServer struct {
	t       *testing.T
	visible map[int64]string
	baseURL string

	mu       sync.Mutex
	requests [][]int64
	status   int
	block    func(ids []int64) bool
	release  chan struct{}
	arrived  chan struct{}
}

func newLabelServer(t *testing.T, visible map[int64]string) *labelServer {
	s := &labelServer{t: t, visible: visible, release: make(chan struct{}), arrived: make(chan struct{}, 100)}
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	t.Cleanup(s.unblock)
	s.baseURL = server.URL
	return s
}

func (s *labelServer) unblock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.release:
	default:
		close(s.release)
	}
}

func (s *labelServer) client() *Client {
	return &Client{Config: &Config{}, Rest: rest.New(s.baseURL, http.DefaultClient)}
}

func (s *labelServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	inner := strings.TrimSuffix(strings.TrimPrefix(r.URL.Query().Get("filter"), "id in ["), "]")
	var ids []int64
	for _, part := range strings.Split(inner, ", ") {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			s.t.Errorf("bad filter %q", r.URL.Query().Get("filter"))
			return
		}
		ids = append(ids, id)
	}
	s.mu.Lock()
	s.requests = append(s.requests, ids)
	status, block := s.status, s.block
	s.mu.Unlock()
	s.arrived <- struct{}{}

	if block != nil && block(ids) {
		select {
		case <-s.release:
		case <-r.Context().Done():
			return
		}
	}
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	datasets := []map[string]string{}
	for _, id := range ids {
		if label, ok := s.visible[id]; ok {
			datasets = append(datasets, map[string]string{"id": strconv.FormatInt(id, 10), "label": label})
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"datasets": datasets, "meta": map[string]any{"totalCount": len(datasets)}})
}

func (s *labelServer) setStatus(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = code
}

func (s *labelServer) setBlock(block func([]int64) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.block = block
}

func (s *labelServer) snapshot() [][]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]int64, len(s.requests))
	for i, ids := range s.requests {
		out[i] = append([]int64(nil), ids...)
	}
	return out
}

func (s *labelServer) waitArrivals(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-s.arrived:
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for request %d", i+1)
		}
	}
}

func contains(ids []int64, id int64) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func lookup(t *testing.T, c *Client, ids ...string) map[string]string {
	t.Helper()
	got, err := c.LookupDatasetLabels(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestDatasetLabelCacheColdWarm(t *testing.T) {
	visible := map[int64]string{}
	var ids []string
	for i := int64(1); i <= 150; i++ {
		visible[i] = "ds" + strconv.FormatInt(i, 10)
		ids = append(ids, strconv.FormatInt(i, 10))
	}
	s := newLabelServer(t, visible)
	c := s.client()

	if got := lookup(t, c, ids...); len(got) != 150 {
		t.Fatalf("cold: got %d labels", len(got))
	}
	if n := len(s.snapshot()); n != 2 {
		t.Fatalf("cold: %d requests, want 2", n)
	}
	if got := lookup(t, c, ids...); len(got) != 150 || got["7"] != "ds7" {
		t.Fatalf("warm: got %d labels", len(got))
	}
	if n := len(s.snapshot()); n != 2 {
		t.Fatalf("warm: %d requests, want 2", n)
	}
}

func TestDatasetLabelCacheMissingIdRequeried(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a"})
	c := s.client()
	for range 2 {
		if got := lookup(t, c, "1", "2"); !reflect.DeepEqual(got, map[string]string{"1": "a"}) {
			t.Fatalf("got %v", got)
		}
	}
	if got := s.snapshot(); !reflect.DeepEqual(got, [][]int64{{1, 2}, {2}}) {
		t.Fatalf("requests %v", got)
	}
}

func TestDatasetLabelCacheFailureNotCached(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a"})
	c := s.client()
	s.setStatus(http.StatusForbidden)
	if _, err := c.LookupDatasetLabels(context.Background(), []string{"1"}); !rest.HasStatusCode(err, http.StatusForbidden) {
		t.Fatalf("err = %v", err)
	}
	s.setStatus(0)
	if got := lookup(t, c, "1"); got["1"] != "a" {
		t.Fatalf("got %v", got)
	}
	if n := len(s.snapshot()); n != 2 {
		t.Fatalf("%d requests, want 2", n)
	}
}

func TestDatasetLabelCacheClientsIsolated(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a"})
	a, b := s.client(), s.client()
	lookup(t, a, "1")
	lookup(t, b, "1")
	lookup(t, a, "1")
	if n := len(s.snapshot()); n != 2 {
		t.Fatalf("%d requests, want 2", n)
	}
}

func TestDatasetLabelCacheSharesOverlappingMisses(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a", 2: "b", 3: "c", 4: "d"})
	c := s.client()
	s.setBlock(func(ids []int64) bool { return contains(ids, 1) })

	var wg sync.WaitGroup
	results := make([]map[string]string, 2)
	errs := make([]error, 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		results[0], errs[0] = c.LookupDatasetLabels(context.Background(), []string{"1", "2"})
	}()
	s.waitArrivals(t, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		results[1], errs[1] = c.LookupDatasetLabels(context.Background(), []string{"2", "3", "4"})
	}()
	s.waitArrivals(t, 1) // second caller fetches only 3 and 4
	s.unblock()
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(results[1], map[string]string{"2": "b", "3": "c", "4": "d"}) {
		t.Fatalf("second result %v", results[1])
	}
	requests := s.snapshot()
	for _, ids := range requests {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}
	if !reflect.DeepEqual(requests, [][]int64{{1, 2}, {3, 4}}) {
		t.Fatalf("requests %v", requests)
	}
}

func TestDatasetLabelCacheOwnerCanceledWaiterRetries(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a", 2: "b"})
	c := s.client()
	first := true
	var mu sync.Mutex
	s.setBlock(func(ids []int64) bool {
		mu.Lock()
		defer mu.Unlock()
		if contains(ids, 1) && first {
			first = false
			return true
		}
		return false
	})

	ownerCtx, cancelOwner := context.WithCancel(context.Background())
	ownerErr := make(chan error, 1)
	go func() {
		_, err := c.LookupDatasetLabels(ownerCtx, []string{"1"})
		ownerErr <- err
	}()
	s.waitArrivals(t, 1)

	waiterResult := make(chan map[string]string, 1)
	waiterErr := make(chan error, 1)
	go func() {
		got, err := c.LookupDatasetLabels(context.Background(), []string{"1", "2"})
		waiterResult <- got
		waiterErr <- err
	}()
	s.waitArrivals(t, 1) // waiter's own fetch of 2: it is now waiting on 1
	cancelOwner()

	if err := <-ownerErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("owner err = %v", err)
	}
	if err := <-waiterErr; err != nil {
		t.Fatalf("waiter err = %v", err)
	}
	if got := <-waiterResult; !reflect.DeepEqual(got, map[string]string{"1": "a", "2": "b"}) {
		t.Fatalf("waiter got %v", got)
	}
	if got := s.snapshot(); !reflect.DeepEqual(got, [][]int64{{1}, {2}, {1}}) {
		t.Fatalf("requests %v", got)
	}
}

func TestDatasetLabelCacheCanceledWaiter(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a"})
	c := s.client()
	s.setBlock(func(ids []int64) bool { return contains(ids, 1) })

	ownerResult := make(chan map[string]string, 1)
	go func() {
		got, _ := c.LookupDatasetLabels(context.Background(), []string{"1"})
		ownerResult <- got
	}()
	s.waitArrivals(t, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.LookupDatasetLabels(ctx, []string{"1"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiter err = %v", err)
	}
	s.unblock()
	if got := <-ownerResult; got["1"] != "a" {
		t.Fatalf("owner got %v", got)
	}
	lookup(t, c, "1")
	if n := len(s.snapshot()); n != 1 {
		t.Fatalf("%d requests, want 1", n)
	}
}

func TestDatasetLabelCacheInvalidatedWhileInFlight(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a"})
	c := s.client()
	s.setBlock(func(ids []int64) bool { return true })

	done := make(chan struct{})
	go func() {
		defer close(done)
		lookup(t, c, "1")
	}()
	s.waitArrivals(t, 1)
	c.invalidateDatasetLabel("1")
	s.unblock()
	<-done

	lookup(t, c, "1")
	if n := len(s.snapshot()); n != 2 {
		t.Fatalf("%d requests, want 2", n)
	}
}

func TestDatasetLabelCacheInvalidatedByMutations(t *testing.T) {
	s := newLabelServer(t, map[int64]string{1: "a", 2: "b"})
	c := newClientWithMockGql(func(req *graphql.Request, resp *graphql.Response) error {
		if req.OpName == "saveDataset" {
			return errors.New("save failed")
		}
		return nil
	})
	c.Rest = rest.New(s.baseURL, http.DefaultClient)

	lookup(t, c, "1", "2")
	id := "1"
	_, _ = c.SaveDataset(context.Background(), "ws", &meta.DatasetInput{Id: &id}, &meta.MultiStageQueryInput{}, nil)
	if err := c.DeleteDataset(context.Background(), "2"); err != nil {
		t.Fatal(err)
	}
	lookup(t, c, "1", "2")
	requests := s.snapshot()
	if len(requests) != 2 || len(requests[1]) != 2 {
		t.Fatalf("requests %v, want both IDs refetched", requests)
	}

	lookup(t, c, "1", "2")
	c.clearDatasetLabels()
	lookup(t, c, "1", "2")
	if n := len(s.snapshot()); n != 3 {
		t.Fatalf("%d requests after clear, want 3", n)
	}
}
