package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func newDatasetLabelTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *atomic.Int32) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return &Client{Config: &Config{}, Rest: rest.New(server.URL, server.Client())}, &requests
}

func TestClientLookupDatasetLabels(t *testing.T) {
	c, requests := newDatasetLabelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter"); got != "id in [7, 41000001, 41000002]" {
			t.Errorf("filter = %q", got)
		}
		_, _ = io.WriteString(w, `{"datasets":[{"id":"7","label":"seven"},{"id":"41000001","label":"a"}],"meta":{"totalCount":2}}`)
	})

	got, err := c.LookupDatasetLabels(context.Background(), []string{"41000001", "007", "41000002", "41000001", "7"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"41000001": "a", "007": "seven", "7": "seven"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("made %d requests", n)
	}
}

func TestClientLookupDatasetLabelsNoRequest(t *testing.T) {
	c, requests := newDatasetLabelTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	for _, ids := range [][]string{nil, {}} {
		got, err := c.LookupDatasetLabels(context.Background(), ids)
		if err != nil || len(got) != 0 {
			t.Fatalf("%v: got %v, %v", ids, got, err)
		}
	}
	for _, ids := range [][]string{{""}, {"0"}, {"-5"}, {"abc"}, {"41000001", "o:::dataset:41000001"}, {"9223372036854775808"}, {" 41000001"}} {
		if _, err := c.LookupDatasetLabels(context.Background(), ids); err == nil {
			t.Errorf("%q: expected error", ids)
		}
	}
	if n := requests.Load(); n != 0 {
		t.Fatalf("made %d requests", n)
	}
}

type closeCountingBody struct {
	io.Reader
	closed *atomic.Int32
}

func (b closeCountingBody) Close() error {
	b.closed.Add(1)
	return nil
}

func TestMiddlewareRetriesAreBoundedAndCloseBodies(t *testing.T) {
	var attempts, closed atomic.Int32
	c := &Client{Config: &Config{RetryCount: 2, RetryWait: time.Millisecond}}
	transport := c.withMiddleware(RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		attempts.Add(1)
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       closeCountingBody{Reader: strings.NewReader(""), closed: &closed},
			Request:    req,
		}, nil
	}))
	c.Rest = rest.New("http://example.invalid", &http.Client{Transport: transport})

	_, err := c.LookupDatasetLabels(context.Background(), []string{"41000001"})
	if !rest.HasStatusCode(err, http.StatusTooManyRequests) {
		t.Fatalf("err = %v", err)
	}
	if attempts.Load() != 3 || closed.Load() != 3 {
		t.Fatalf("attempts %d, closed %d", attempts.Load(), closed.Load())
	}
}

func TestMiddlewareRetryHonorsContext(t *testing.T) {
	var attempts atomic.Int32
	c := &Client{Config: &Config{RetryCount: 5, RetryWait: time.Hour}}
	transport := c.withMiddleware(RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		attempts.Add(1)
		return &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	}))
	c.Rest = rest.New("http://example.invalid", &http.Client{Transport: transport})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.LookupDatasetLabels(ctx, []string{"41000001"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second || attempts.Load() != 1 {
		t.Fatalf("elapsed %s, attempts %d", elapsed, attempts.Load())
	}
}
