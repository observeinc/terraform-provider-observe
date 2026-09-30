package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeDatasets serves /v1/datasets for an `id in [...]` filter over visible.
type fakeDatasets struct {
	t        *testing.T
	visible  map[int64]string
	pageSize int // server-side page cap; 0 means honor limit

	mu       sync.Mutex
	requests []url.Values
	// override, when set, replaces the response for every request.
	override func(w http.ResponseWriter, r *http.Request)
}

func (f *fakeDatasets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.URL.Query())
	f.mu.Unlock()

	if r.Method != http.MethodGet || r.URL.Path != "/v1/datasets" {
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}
	if f.override != nil {
		f.override(w, r)
		return
	}
	ids, err := parseIdFilter(r.URL.Query().Get("filter"))
	if err != nil {
		f.t.Errorf("bad filter: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var matched []int64
	for _, id := range ids {
		if _, ok := f.visible[id]; ok {
			matched = append(matched, id)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i] < matched[j] })
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if f.pageSize > 0 && f.pageSize < limit {
		limit = f.pageSize
	}
	page := matched[min(offset, len(matched)):min(offset+limit, len(matched))]
	datasets := make([]map[string]any, 0, len(page))
	for _, id := range page {
		datasets = append(datasets, map[string]any{
			"id":        strconv.FormatInt(id, 10),
			"label":     f.visible[id],
			"workspace": map[string]any{"id": "41000001"},
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"datasets": datasets,
		"meta":     map[string]any{"totalCount": len(matched)},
	})
}

func parseIdFilter(filter string) ([]int64, error) {
	inner, ok := strings.CutPrefix(filter, "id in [")
	if !ok {
		return nil, fmt.Errorf("unexpected filter %q", filter)
	}
	inner, ok = strings.CutSuffix(inner, "]")
	if !ok {
		return nil, fmt.Errorf("unexpected filter %q", filter)
	}
	var ids []int64
	for _, part := range strings.Split(inner, ", ") {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (f *fakeDatasets) snapshot() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.requests...)
}

func newFake(t *testing.T, visible map[int64]string) (*fakeDatasets, *Client) {
	f := &fakeDatasets{t: t, visible: visible}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	return f, New(server.URL, server.Client())
}

func idRange(first int64, n int) []int64 {
	ids := make([]int64, n)
	for i := range ids {
		ids[i] = first + int64(i)
	}
	return ids
}

func TestLookupDatasetLabelsBatching(t *testing.T) {
	for _, n := range []int{0, 1, 50, 100, 101, 250} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			ids := idRange(41000000, n)
			visible := map[int64]string{}
			for _, id := range ids {
				visible[id] = fmt.Sprintf("ds/%d", id)
			}
			f, client := newFake(t, visible)

			// Reverse and repeat to exercise dedup and ordering.
			input := make([]int64, 0, 2*n)
			for i := n - 1; i >= 0; i-- {
				input = append(input, ids[i], ids[i])
			}
			got, err := client.LookupDatasetLabels(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, visible) {
				t.Fatalf("got %d labels, want %d", len(got), len(visible))
			}

			reqs := f.snapshot()
			if want := (n + 99) / 100; len(reqs) != want {
				t.Fatalf("got %d requests, want %d", len(reqs), want)
			}
			var covered []int64
			for _, q := range reqs {
				assertLookupQuery(t, q)
				batch, err := parseIdFilter(q.Get("filter"))
				if err != nil {
					t.Fatal(err)
				}
				if len(batch) > 100 {
					t.Fatalf("batch of %d ids", len(batch))
				}
				covered = append(covered, batch...)
			}
			if n > 0 && !reflect.DeepEqual(covered, ids) {
				t.Fatalf("batches cover %v, want %v", covered, ids)
			}
		})
	}
}

func assertLookupQuery(t *testing.T, q url.Values) {
	t.Helper()
	if q.Get("filter") == "" {
		t.Fatal("request without filter")
	}
	for key, want := range map[string]string{"limit": "100", "offset": "0", "orderBy": "id"} {
		if got := q.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	for _, key := range []string{"query", "expand"} {
		if q.Has(key) {
			t.Errorf("unexpected %s parameter", key)
		}
	}
}

func TestLookupDatasetLabelsEncodedQuery(t *testing.T) {
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, `{"datasets":[],"meta":{"totalCount":0}}`)
	}))
	defer server.Close()

	if _, err := New(server.URL, server.Client()).LookupDatasetLabels(context.Background(), []int64{41000002, 41000001}); err != nil {
		t.Fatal(err)
	}
	want := "filter=id+in+%5B41000001%2C+41000002%5D&limit=100&offset=0&orderBy=id"
	if rawQuery != want {
		t.Fatalf("query = %q, want %q", rawQuery, want)
	}
}

func TestLookupDatasetLabelsLargeIds(t *testing.T) {
	// Above 2^53, where float64 decoding would lose precision.
	big := []int64{1<<53 + 1, 9223372036854775807}
	f, client := newFake(t, map[int64]string{big[0]: "a", big[1]: "b"})
	got, err := client.LookupDatasetLabels(context.Background(), big)
	if err != nil {
		t.Fatal(err)
	}
	if got[big[0]] != "a" || got[big[1]] != "b" {
		t.Fatalf("got %v", got)
	}
	if filter := f.snapshot()[0].Get("filter"); filter != "id in [9007199254740993, 9223372036854775807]" {
		t.Fatalf("filter = %q", filter)
	}
}

func TestLookupDatasetLabelsOmitsMissing(t *testing.T) {
	_, client := newFake(t, map[int64]string{41000001: "visible"})
	got, err := client.LookupDatasetLabels(context.Background(), []int64{41000001, 41000002, 41000003})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[int64]string{41000001: "visible"}) {
		t.Fatalf("got %v", got)
	}
}

func TestLookupDatasetLabelsInvalidIds(t *testing.T) {
	f, client := newFake(t, nil)
	for _, ids := range [][]int64{{0}, {-1}, {41000001, 0}} {
		if _, err := client.LookupDatasetLabels(context.Background(), ids); err == nil {
			t.Errorf("%v: expected error", ids)
		}
	}
	if n := len(f.snapshot()); n != 0 {
		t.Fatalf("made %d requests for invalid input", n)
	}
}

func TestLookupDatasetLabelsPaginatesShortPages(t *testing.T) {
	ids := idRange(41000000, 7)
	visible := map[int64]string{}
	for _, id := range ids {
		visible[id] = "x"
	}
	f, client := newFake(t, visible)
	f.pageSize = 3

	got, err := client.LookupDatasetLabels(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 7 {
		t.Fatalf("got %d labels", len(got))
	}
	var offsets []string
	for _, q := range f.snapshot() {
		offsets = append(offsets, q.Get("offset"))
	}
	if !reflect.DeepEqual(offsets, []string{"0", "3", "6"}) {
		t.Fatalf("offsets = %v", offsets)
	}
}

func TestLookupDatasetLabelsRejectsBadResponses(t *testing.T) {
	cases := map[string]string{
		"malformed json":     `{"datasets":`,
		"missing meta":       `{"datasets":[]}`,
		"missing totalCount": `{"datasets":[],"meta":{}}`,
		"missing datasets":   `{"meta":{"totalCount":0}}`,
		"negative total":     `{"datasets":[],"meta":{"totalCount":-1}}`,
		"total above batch":  `{"datasets":[],"meta":{"totalCount":5}}`,
		"numeric id":         `{"datasets":[{"id":41000001,"label":"a"}],"meta":{"totalCount":1}}`,
		"non-decimal id":     `{"datasets":[{"id":"o:::dataset:41000001","label":"a"}],"meta":{"totalCount":1}}`,
		"non-canonical id":   `{"datasets":[{"id":"041000001","label":"a"}],"meta":{"totalCount":1}}`,
		"missing id":         `{"datasets":[{"label":"a"}],"meta":{"totalCount":1}}`,
		"missing label":      `{"datasets":[{"id":"41000001"}],"meta":{"totalCount":1}}`,
		"unrequested id":     `{"datasets":[{"id":"41000009","label":"a"}],"meta":{"totalCount":1}}`,
		"repeated id":        `{"datasets":[{"id":"41000001","label":"a"},{"id":"41000001","label":"a"}],"meta":{"totalCount":2}}`,
		"empty page":         `{"datasets":[],"meta":{"totalCount":1}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			f, client := newFake(t, nil)
			f.override = func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }
			if _, err := client.LookupDatasetLabels(context.Background(), []int64{41000001, 41000002}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestLookupDatasetLabelsStatusErrors(t *testing.T) {
	for _, code := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			f, client := newFake(t, nil)
			f.override = func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = io.WriteString(w, `{"message":"nope"}`)
			}
			_, err := client.LookupDatasetLabels(context.Background(), []int64{41000001})
			var statusErr ErrorWithStatusCode
			if !errors.As(err, &statusErr) || statusErr.StatusCode != code {
				t.Fatalf("err = %v, want status %d", err, code)
			}
		})
	}
}

func TestLookupDatasetLabelsStopsAtFirstFailedBatch(t *testing.T) {
	f, client := newFake(t, nil)
	f.override = func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }
	if _, err := client.LookupDatasetLabels(context.Background(), idRange(41000000, 250)); err == nil {
		t.Fatal("expected error")
	}
	if n := len(f.snapshot()); n != 1 {
		t.Fatalf("made %d requests after failure", n)
	}
}

func TestLookupDatasetLabelsContext(t *testing.T) {
	release := make(chan struct{})
	f, client := newFake(t, nil)
	f.override = func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.LookupDatasetLabels(ctx, []int64{41000001}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: err = %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := client.LookupDatasetLabels(ctx, []int64{41000001}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline: err = %v", err)
	}
}

type trackedBody struct {
	io.Reader
	closed *atomic.Int32
}

func (b trackedBody) Close() error {
	b.closed.Add(1)
	return nil
}

func TestLookupDatasetLabelsClosesBodies(t *testing.T) {
	responses := map[string]struct {
		code int
		body string
	}{
		"ok":        {http.StatusOK, `{"datasets":[{"id":"41000001","label":"a"}],"meta":{"totalCount":1}}`},
		"malformed": {http.StatusOK, `{`},
		"invalid":   {http.StatusOK, `{"datasets":[{"id":"41000009","label":"a"}],"meta":{"totalCount":1}}`},
		"forbidden": {http.StatusForbidden, `{"message":"no"}`},
	}
	for name, r := range responses {
		t.Run(name, func(t *testing.T) {
			var opened, closed atomic.Int32
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				opened.Add(1)
				return &http.Response{
					StatusCode: r.code,
					Body:       trackedBody{Reader: strings.NewReader(r.body), closed: &closed},
					Request:    req,
				}, nil
			})
			client := New("http://example.invalid", &http.Client{Transport: transport})
			_, _ = client.LookupDatasetLabels(context.Background(), []int64{41000001})
			if opened.Load() != 1 || closed.Load() != 1 {
				t.Fatalf("opened %d, closed %d", opened.Load(), closed.Load())
			}
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
