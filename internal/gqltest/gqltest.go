// Package gqltest stubs the Observe GraphQL backend for tests. It records the requests a client
// sends and answers each one with a canned response, so a test can assert what a client asked
// the backend for, not only what it did with the answer.
//
// It sits outside the packages under test so that both the GraphQL client (client/meta) and the
// resource layer (observe) can share one stub.
package gqltest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// Request is one recorded GraphQL request.
type Request struct {
	OperationName string          `json:"operationName"`
	Query         string          `json:"query"`
	Variables     json.RawMessage `json:"variables"`
}

// RequestedFields returns the name of every field the request selects, resolving fragment
// spreads against the fragment definitions genqlient emits into the same document.
//
// This is the meaningful assertion to make about an outgoing request, because the Observe
// backend decides how much work to do from the fields a query asks for: it turns the selection
// set into a response field mask and skips expensive work for fields nobody requested. It is
// also more precise than searching the query text, which matches comments and fields of
// unrelated types too.
func (r Request) RequestedFields(tb testing.TB) map[string]bool {
	tb.Helper()

	doc, err := parser.ParseQuery(&ast.Source{Input: r.Query})
	if err != nil {
		tb.Fatalf("gqltest: parsing operation %q: %s", r.OperationName, err)
	}

	fields := make(map[string]bool)
	var walk func(ast.SelectionSet)
	walk = func(set ast.SelectionSet) {
		for _, selection := range set {
			switch s := selection.(type) {
			case *ast.Field:
				fields[s.Name] = true
				walk(s.SelectionSet)
			case *ast.InlineFragment:
				walk(s.SelectionSet)
			case *ast.FragmentSpread:
				if definition := doc.Fragments.ForName(s.Name); definition != nil {
					walk(definition.SelectionSet)
				}
			}
		}
	}
	for _, operation := range doc.Operations {
		walk(operation.SelectionSet)
	}
	return fields
}

// Recorder is a stub GraphQL backend.
type Recorder struct {
	tb     testing.TB
	server *httptest.Server

	mu       sync.Mutex
	requests []Request
}

// New starts a stub GraphQL backend that records every request and answers each one with
// response, a raw GraphQL response body such as `{"data":{...}}`. The server is shut down when
// the test finishes.
func New(tb testing.TB, response string) *Recorder {
	tb.Helper()

	recorder := &Recorder{tb: tb}
	recorder.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			tb.Errorf("gqltest: decoding request: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		recorder.mu.Lock()
		recorder.requests = append(recorder.requests, request)
		recorder.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, response); err != nil {
			tb.Errorf("gqltest: writing response: %s", err)
		}
	}))
	tb.Cleanup(recorder.server.Close)

	return recorder
}

// Client returns a GraphQL client that talks to the stub.
func (r *Recorder) Client() graphql.Client {
	return graphql.NewClient(r.server.URL, r.server.Client())
}

// Requests returns the recorded requests, in the order they were received.
func (r *Recorder) Requests() []Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Request(nil), r.requests...)
}

// OnlyRequest returns the single request the client sent, failing the test if it sent any other
// number of them.
func (r *Recorder) OnlyRequest() Request {
	r.tb.Helper()

	requests := r.Requests()
	if len(requests) != 1 {
		r.tb.Fatalf("gqltest: got %d requests, want exactly 1", len(requests))
	}
	return requests[0]
}
