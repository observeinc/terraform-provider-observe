package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIngestRouteClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ingest/routes/otellogs/41030001" && r.URL.Path != "/v1/ingest/routes/otellogs" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost:
			assertIngestRouteBody(t, r, true)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(ingestRouteJSON))
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/ingest/routes/otellogs/41030001":
			assertIngestRouteBody(t, r, false)
			_, _ = w.Write([]byte(ingestRouteJSON))
		case r.Method == http.MethodPatch:
			var body IngestRoutePriorityUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode ordering: %v", err)
				return
			}
			if len(body.Ordering) != 2 || body.Ordering[0] != "41030001" {
				t.Errorf("unexpected ordering %#v", body.Ordering)
			}
			_, _ = w.Write([]byte(`{"ingestRoutes": [` + ingestRouteJSON + `]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/ingest/routes/otellogs/41030001":
			_, _ = w.Write([]byte(ingestRouteJSON))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"ingestRoutes": [` + ingestRouteJSON + `]}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	req := &IngestRouteCreateRequest{
		Pipeline:               `filter FIELDS.service == "api"`,
		DestinationId:          "41007777",
		SecondaryDestinationId: routeStringPointer("41008888"),
	}
	route, err := client.CreateIngestRoute(context.Background(), "otellogs", req)
	if err != nil || route.Id != "41030001" {
		t.Fatalf("CreateIngestRoute = %#v, %v", route, err)
	}
	if route.Layout == nil || route.Layout["x"] != "server-only" {
		t.Errorf("layout was not decoded: %#v", route.Layout)
	}
	if _, err := client.GetIngestRoute(context.Background(), "otellogs", route.Id); err != nil {
		t.Fatalf("GetIngestRoute: %v", err)
	}
	if _, err := client.ListIngestRoutes(context.Background(), "otellogs"); err != nil {
		t.Fatalf("ListIngestRoutes: %v", err)
	}
	if _, err := client.UpdateIngestRoute(context.Background(), "otellogs", route.Id, &IngestRouteUpdateRequest{
		SecondaryDestinationId:          nil,
		SecondaryDestinationIdSpecified: true,
		Enabled:                         routeBoolPointer(false),
	}); err != nil {
		t.Fatalf("UpdateIngestRoute: %v", err)
	}
	if _, err := client.UpdateIngestRoutePriorities(context.Background(), "otellogs", []string{"41030001", "41030002"}); err != nil {
		t.Fatalf("UpdateIngestRoutePriorities: %v", err)
	}
	if err := client.DeleteIngestRoute(context.Background(), "otellogs", route.Id); err != nil {
		t.Fatalf("DeleteIngestRoute: %v", err)
	}
}

func assertIngestRouteBody(t *testing.T, r *http.Request, create bool) {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read request body: %v", err)
		return
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Errorf("decode request: %v", err)
		return
	}
	if _, ok := body["layout"]; ok {
		t.Errorf("layout must not be sent by Terraform client: %s", raw)
	}
	if create {
		if _, ok := body["enabled"]; ok {
			t.Errorf("create must not serialize enabled: %s", raw)
		}
	}
	if !create {
		if value, ok := body["secondaryDestinationId"]; !ok || value != nil {
			t.Errorf("update must serialize secondaryDestinationId=null: %s", raw)
		}
	}
}

func TestUpdateIngestRoutePrioritiesPreservesBadRequestStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad ordering"}`))
	}))
	defer server.Close()

	_, err := New(server.URL, server.Client()).UpdateIngestRoutePriorities(context.Background(), "otellogs", []string{"41030001"})
	if !HasStatusCode(err, http.StatusBadRequest) {
		t.Fatalf("priority update error = %v, want HTTP 400", err)
	}
}

const ingestRouteJSON = `{"id":"41030001","type":"otellogs","pipeline":"filter true","layout":{"x":"server-only"},"destinationId":"41007777","secondaryDestinationId":"41008888","enabled":true,"managedBy":null}`

func routeStringPointer(value string) *string { return &value }

func routeBoolPointer(value bool) *bool { return &value }
