// Package bindingtest provides an HTTP-backed fake of the Observe APIs used by
// export binding generation.
package bindingtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

// Object is an id/label pair served by the fake.
type Object struct {
	ID    string
	Label string
}

// User is a user served by the fake.
type User struct {
	ID    int64
	Email string
	Label string
}

// Tenant is the fake's content. Slices are served in the given order.
type Tenant struct {
	Workspace        Object
	Datasets         []Object
	Worksheets       []Object
	Users            []User
	MonitorV2Actions []Object
}

// Failure selects how a request fails.
type Failure int

const (
	FailNone Failure = iota
	// FailUnauthorized responds 401.
	FailUnauthorized
	// FailMalformed responds 200 with an unparseable body.
	FailMalformed
	// FailNetwork closes the connection without responding.
	FailNetwork
)

// Operation names accepted by Fail and Count: GraphQL operation names, plus
// OpRestDatasets for the one REST endpoint.
const (
	// OpRestDatasets is GET /v1/datasets.
	OpRestDatasets = "rest:datasets"

	OpListWorkspaces            = "listWorkspaces"
	OpListDatasetsIdNameOnly    = "listDatasetsIdNameOnly"
	OpListWorksheetsIdLabelOnly = "listWorksheetsIdLabelOnly"
	OpListUsers                 = "listUsers"
	OpSearchMonitorV2Action     = "searchMonitorV2Action"
)

// Server is a fake Observe API. Unknown operations fail the test.
type Server struct {
	*httptest.Server
	t testing.TB

	mu       sync.Mutex
	tenant   Tenant
	failures map[string]Failure
	counts   map[string]int
	queries  []string // raw queries of GET /v1/datasets
}

// New starts a fake serving tenant; it is closed on test cleanup.
func New(t testing.TB, tenant Tenant) *Server {
	s := &Server{
		t:        t,
		tenant:   tenant,
		failures: make(map[string]Failure),
		counts:   make(map[string]int),
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.Close)
	return s
}

// Client returns a client whose GraphQL and REST APIs target the fake.
func (s *Server) Client() *observe.Client {
	metaAPI, err := meta.New(s.URL+"/v1/meta", s.Server.Client())
	if err != nil {
		s.t.Fatal(err)
	}
	return &observe.Client{
		Config: &observe.Config{CustomerID: "1", Domain: "example.invalid", ExportObjectBindings: true},
		Meta:   metaAPI,
		Rest:   rest.New(s.URL, s.Server.Client()),
	}
}

// Fail makes every later request for op fail with f.
func (s *Server) Fail(op string, f Failure) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures[op] = f
}

// Count returns how many requests for op were received.
func (s *Server) Count(op string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[op]
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v1/meta":
		s.serveGraphQL(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/v1/datasets":
		s.mu.Lock()
		s.queries = append(s.queries, r.URL.RawQuery)
		s.mu.Unlock()
		if s.fail(w, OpRestDatasets) {
			return
		}
		s.serveRestDatasets(w, r)
	default:
		s.t.Errorf("bindingtest: unexpected request %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", http.StatusNotFound)
	}
}

// fail counts op and applies its configured failure, reporting whether it did.
func (s *Server) fail(w http.ResponseWriter, op string) bool {
	s.mu.Lock()
	s.counts[op]++
	f := s.failures[op]
	s.mu.Unlock()
	switch f {
	case FailUnauthorized:
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case FailMalformed:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":`))
	case FailNetwork:
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			s.t.Errorf("bindingtest: hijack: %s", err)
			return true
		}
		_ = conn.Close()
	default:
		return false
	}
	return true
}

func (s *Server) serveGraphQL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.t.Errorf("bindingtest: decoding graphql request: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if s.fail(w, req.OperationName) {
		return
	}

	s.mu.Lock()
	tenant := s.tenant
	s.mu.Unlock()

	var data interface{}
	switch req.OperationName {
	case OpListWorkspaces:
		data = map[string]interface{}{
			"workspaces": []interface{}{
				map[string]interface{}{"id": tenant.Workspace.ID, "label": tenant.Workspace.Label},
			},
		}
	case OpListDatasetsIdNameOnly:
		matches := make([]interface{}, 0, len(tenant.Datasets))
		for _, ds := range tenant.Datasets {
			matches = append(matches, map[string]interface{}{
				"dataset": map[string]interface{}{"id": ds.ID, "name": ds.Label},
			})
		}
		data = map[string]interface{}{"datasets": matches}
	case OpListWorksheetsIdLabelOnly:
		matches := make([]interface{}, 0, len(tenant.Worksheets))
		for _, wk := range tenant.Worksheets {
			matches = append(matches, map[string]interface{}{
				"worksheet": map[string]interface{}{"id": wk.ID, "label": wk.Label},
			})
		}
		data = map[string]interface{}{"worksheetSearch": map[string]interface{}{"worksheets": matches}}
	case OpListUsers:
		users := make([]interface{}, 0, len(tenant.Users))
		for _, u := range tenant.Users {
			users = append(users, map[string]interface{}{
				"id": strconv.FormatInt(u.ID, 10), "email": u.Email, "label": u.Label,
			})
		}
		data = map[string]interface{}{"users": map[string]interface{}{"users": users}}
	case OpSearchMonitorV2Action:
		results := make([]interface{}, 0, len(tenant.MonitorV2Actions))
		for _, a := range tenant.MonitorV2Actions {
			results = append(results, map[string]interface{}{
				"id": a.ID, "name": a.Label, "workspaceId": tenant.Workspace.ID, "type": "Email",
			})
		}
		data = map[string]interface{}{"monitorV2Actions": map[string]interface{}{"results": results}}
	default:
		s.t.Errorf("bindingtest: unexpected graphql operation %q", req.OperationName)
		http.Error(w, "unexpected operation", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]interface{}{"data": data})
}

var idInFilter = regexp.MustCompile(`^id in \[([0-9, ]*)\]$`)

// serveRestDatasets implements the `id in [...]` filter with limit/offset paging.
func (s *Server) serveRestDatasets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	match := idInFilter.FindStringSubmatch(q.Get("filter"))
	if match == nil {
		s.t.Errorf("bindingtest: unsupported dataset filter %q", q.Get("filter"))
		http.Error(w, "unsupported filter", http.StatusBadRequest)
		return
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		http.Error(w, "bad limit", http.StatusBadRequest)
		return
	}
	offset, err := strconv.Atoi(q.Get("offset"))
	if err != nil || offset < 0 {
		http.Error(w, "bad offset", http.StatusBadRequest)
		return
	}
	wanted := make(map[string]bool)
	for _, id := range strings.Split(match[1], ",") {
		if id = strings.TrimSpace(id); id != "" {
			wanted[id] = true
		}
	}

	s.mu.Lock()
	var hits []interface{}
	for _, ds := range s.tenant.Datasets {
		if wanted[ds.ID] {
			hits = append(hits, map[string]interface{}{"id": ds.ID, "label": ds.Label})
		}
	}
	s.mu.Unlock()

	total := len(hits)
	page := []interface{}{}
	if offset < total {
		page = hits[offset:min(offset+limit, total)]
	}
	writeJSON(w, map[string]interface{}{
		"datasets": page,
		"meta":     map[string]interface{}{"totalCount": total},
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(fmt.Sprintf("bindingtest: encoding response: %s", err))
	}
}

var localRef = regexp.MustCompile(`\$\{local\.([A-Za-z0-9_-]+)\}`)

// UndefinedLocals returns the ${local.X} references in doc that bindingsJSON
// (a serialized binding.BindingsObject) does not define.
func UndefinedLocals(doc interface{}, bindingsJSON []byte) ([]string, error) {
	var b struct {
		Mappings map[string]struct {
			TfLocalBindingVar string `json:"tf_local_binding_var"`
		} `json:"mappings"`
		Workspace struct {
			TfLocalBindingVar string `json:"tf_local_binding_var"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(bindingsJSON, &b); err != nil {
		return nil, err
	}
	defined := map[string]bool{b.Workspace.TfLocalBindingVar: true}
	for _, t := range b.Mappings {
		defined[t.TfLocalBindingVar] = true
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var undefined []string
	for _, m := range localRef.FindAllStringSubmatch(string(raw), -1) {
		if !defined[m[1]] {
			undefined = append(undefined, m[1])
		}
	}
	return undefined, nil
}

// DatasetFilterIDs returns the ids named by each GET /v1/datasets filter.
func (s *Server) DatasetFilterIDs() [][]string {
	s.mu.Lock()
	queries := append([]string(nil), s.queries...)
	s.mu.Unlock()
	var out [][]string
	for _, raw := range queries {
		q, err := url.ParseQuery(raw)
		if err != nil {
			s.t.Errorf("bindingtest: parsing query %q: %s", raw, err)
			continue
		}
		match := idInFilter.FindStringSubmatch(q.Get("filter"))
		if match == nil {
			s.t.Errorf("bindingtest: unsupported dataset filter %q", q.Get("filter"))
			continue
		}
		var ids []string
		for _, id := range strings.Split(match[1], ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
		out = append(out, ids)
	}
	return out
}
