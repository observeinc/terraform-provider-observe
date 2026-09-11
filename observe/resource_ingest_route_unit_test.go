package observe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func TestParseIngestRouteImportID(t *testing.T) {
	typeName, routeID, err := parseIngestRouteImportID("otellogs/41030001")
	if err != nil {
		t.Fatalf("parse import ID: %v", err)
	}
	if typeName != "otellogs" || routeID != "41030001" {
		t.Fatalf("got %q, %q", typeName, routeID)
	}
	if _, _, err := parseIngestRouteImportID("41030001"); err == nil {
		t.Fatal("expected malformed import ID error")
	}
}

func TestIngestRouteOrderImportID(t *testing.T) {
	data := schema.TestResourceDataRaw(t, resourceIngestRouteOrder().Schema, map[string]interface{}{})
	data.SetId("otellogs")
	imported, err := resourceIngestRouteOrderImport(nil, data, nil)
	if err != nil {
		t.Fatalf("import route order: %v", err)
	}
	if len(imported) != 1 || imported[0].Get("type") != "otellogs" {
		t.Fatalf("unexpected imported state %#v", imported)
	}
}

func TestIngestRouteSchemaValidation(t *testing.T) {
	resource := resourceIngestRoute()
	if _, ok := resource.Schema["id"]; ok {
		t.Error("route ID must not use Terraform's reserved id attribute")
	}
	if !resource.Schema["route_id"].Computed {
		t.Error("route_id must be computed")
	}
	if !resource.Schema["type"].ForceNew {
		t.Error("type must force a new route")
	}
	if diags := resource.Schema["type"].ValidateDiagFunc("not-a-type", nil); !diags.HasError() {
		t.Error("invalid type was accepted")
	}
	if diags := resource.Schema["pipeline"].ValidateDiagFunc("   ", nil); !diags.HasError() {
		t.Error("blank pipeline was accepted")
	}
}

func TestMergeIngestRouteOrdering(t *testing.T) {
	routes := []rest.IngestRouteResource{
		{Id: "managed", Pipeline: "filter managed", ManagedBy: &rest.IngestRouteObjectRef{Id: "manager"}},
		{Id: "unmanaged-a", Pipeline: "filter a"},
		{Id: "configured-a", Pipeline: "filter configured"},
		{Id: "unmanaged-b", Pipeline: "filter b"},
		{Id: "default", Pipeline: ""},
	}
	got, err := mergeIngestRouteOrdering("otellogs", []string{"configured-a"}, routes)
	if err != nil {
		t.Fatalf("merge ordering: %v", err)
	}
	want := []string{"configured-a", "unmanaged-a", "unmanaged-b", "default"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMergeIngestRouteOrderingIncludesEachEligibleRouteOnce(t *testing.T) {
	routes := []rest.IngestRouteResource{
		{Id: "configured", Pipeline: "filter configured"},
		{Id: "unmanaged", Pipeline: "filter unmanaged"},
		{Id: "managed", Pipeline: "filter managed", ManagedBy: &rest.IngestRouteObjectRef{Id: "manager"}},
		{Id: "unmanaged", Pipeline: "filter duplicate"},
		{Id: "default", Pipeline: ""},
		{Id: "default", Pipeline: ""},
	}
	got, err := mergeIngestRouteOrdering("otellogs", []string{"configured"}, routes)
	if err != nil {
		t.Fatalf("merge ordering: %v", err)
	}
	want := []string{"configured", "unmanaged", "default"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ordering = %v, want %v", got, want)
	}
}

func TestMergeIngestRouteOrderingRejectsInvalidConfiguredRoutes(t *testing.T) {
	managedBy := &rest.IngestRouteObjectRef{Id: "manager"}
	cases := []struct {
		name   string
		ids    []string
		routes []rest.IngestRouteResource
	}{
		{"duplicate", []string{"route", "route"}, []rest.IngestRouteResource{{Id: "route", Pipeline: "filter true"}}},
		{"missing", []string{"missing"}, []rest.IngestRouteResource{{Id: "route", Pipeline: "filter true"}}},
		{"managed", []string{"managed"}, []rest.IngestRouteResource{{Id: "managed", Pipeline: "filter true", ManagedBy: managedBy}}},
		{"default", []string{"default"}, []rest.IngestRouteResource{{Id: "default", Pipeline: ""}}},
		{"wrong type", []string{"route"}, []rest.IngestRouteResource{{Id: "route", Type: "any", Pipeline: "filter true"}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := mergeIngestRouteOrdering("otellogs", testCase.ids, testCase.routes); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestResourceIngestRouteCreateEnablesConfiguredRoute(t *testing.T) {
	requests := make([]struct {
		method string
		body   map[string]interface{}
	}, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]interface{}
		if request.Method != http.MethodGet {
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode request body: %v", err)
				return
			}
		}
		requests = append(requests, struct {
			method string
			body   map[string]interface{}
		}{method: request.Method, body: body})
		writer.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodPost, http.MethodPatch, http.MethodGet:
			_, _ = writer.Write([]byte(`{"id":"41030001","type":"otellogs","pipeline":"filter true","destinationId":"41007777","enabled":true}`))
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{
		"type":           "otellogs",
		"pipeline":       "filter true",
		"destination_id": "41007777",
		"enabled":        true,
	})
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteCreate(context.Background(), data, client); diags.HasError() {
		t.Fatalf("create ingest route: %v", diags)
	}
	if data.Id() != "otellogs/41030001" {
		t.Errorf("state ID = %q, want %q", data.Id(), "otellogs/41030001")
	}
	if data.Get("route_id") != "41030001" {
		t.Errorf("route_id = %q, want bare route ID", data.Get("route_id"))
	}
	if len(requests) != 3 {
		t.Fatalf("requests = %d, want 3", len(requests))
	}
	if _, ok := requests[0].body["enabled"]; ok {
		t.Errorf("create request contains enabled: %#v", requests[0].body)
	}
	if requests[1].method != http.MethodPatch || requests[1].body["enabled"] != true {
		t.Errorf("enable request = %#v, want PATCH enabled=true", requests[1])
	}
}

func TestResourceIngestRouteCreateLeavesDisabledRouteUnpatched(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		if request.Method == http.MethodPatch {
			t.Errorf("disabled route must not be patched after creation")
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"41030001","type":"otellogs","pipeline":"filter true","destinationId":"41007777","enabled":false}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{
		"type":           "otellogs",
		"pipeline":       "filter true",
		"destination_id": "41007777",
		"enabled":        false,
	})
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteCreate(context.Background(), data, client); diags.HasError() {
		t.Fatalf("create disabled ingest route: %v", diags)
	}
	if requestCount != 2 {
		t.Errorf("requests = %d, want POST then GET", requestCount)
	}
}

func TestResourceIngestRouteCreateRetainsIDWhenEnableFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPatch {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(`{"message":"enable failed"}`))
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"41030001","type":"otellogs","pipeline":"filter true","destinationId":"41007777","enabled":false}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{
		"type":           "otellogs",
		"pipeline":       "filter true",
		"destination_id": "41007777",
		"enabled":        true,
	})
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteCreate(context.Background(), data, client); !diags.HasError() {
		t.Fatal("expected enable failure diagnostic")
	}
	if data.Id() != "otellogs/41030001" {
		t.Errorf("state ID = %q, want created route ID", data.Id())
	}
}

func TestResourceIngestRouteUpdateClearsSecondaryDestination(t *testing.T) {
	requests := make([]map[string]interface{}, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodPatch:
			var body map[string]interface{}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode update request: %v", err)
				return
			}
			requests = append(requests, body)
			_, _ = writer.Write([]byte(`{"id":"41030001","type":"otellogs","pipeline":"filter updated","destinationId":"41009999","enabled":false}`))
		case http.MethodGet:
			_, _ = writer.Write([]byte(`{"id":"41030001","type":"otellogs","pipeline":"filter updated","destinationId":"41009999","enabled":false}`))
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{
		"type":           "otellogs",
		"pipeline":       "filter updated",
		"destination_id": "41009999",
		"enabled":        false,
	})
	data.SetId("otellogs/41030001")
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteUpdate(context.Background(), data, client); diags.HasError() {
		t.Fatalf("update ingest route: %v", diags)
	}
	if len(requests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(requests))
	}
	request := requests[0]
	if value, ok := request["secondaryDestinationId"]; !ok || value != nil {
		t.Errorf("secondaryDestinationId = %#v, want explicit null", request["secondaryDestinationId"])
	}
	if _, ok := request["layout"]; ok {
		t.Errorf("update request must not contain layout: %#v", request)
	}
	if got, want := request["pipeline"], "filter updated"; got != want {
		t.Errorf("pipeline = %#v, want %q", got, want)
	}
	if got, want := request["destinationId"], "41009999"; got != want {
		t.Errorf("destinationId = %#v, want %q", got, want)
	}
	if got, want := request["enabled"], false; got != want {
		t.Errorf("enabled = %#v, want %t", got, want)
	}
}

func TestResourceIngestRouteOrderReadUsesConfiguredType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/ingest/routes/otellogs" {
			t.Errorf("route list path = %q, want configured type", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"configured","pipeline":"filter true"},{"id":"default","pipeline":""}]}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRouteOrder().Schema, map[string]interface{}{
		"type":      "otellogs",
		"route_ids": []interface{}{"configured"},
	})
	data.SetId("unexpected-state-id")
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteOrderRead(context.Background(), data, client); diags.HasError() {
		t.Fatalf("read route order: %v", diags)
	}
}

func TestResourceIngestRouteReadKeepsValidatedTypeWhenResponseOmitsType(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		if request.URL.Path != "/v1/ingest/routes/otellogs/41030001" {
			t.Errorf("route path = %q, want parsed type and ID", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"41030001","pipeline":"filter true","destinationId":"41007777","enabled":true}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{})
	data.SetId("otellogs/41030001")
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	for readAttempt := 0; readAttempt < 2; readAttempt++ {
		if diags := resourceIngestRouteRead(context.Background(), data, client); diags.HasError() {
			t.Fatalf("read ingest route: %v", diags)
		}
	}
	if data.Id() != "otellogs/41030001" {
		t.Errorf("state ID = %q, want parsed type and returned ID", data.Id())
	}
	if got, want := data.Get("type"), "otellogs"; got != want {
		t.Errorf("type = %q, want %q", got, want)
	}
	if requestCount != 2 {
		t.Errorf("read requests = %d, want 2", requestCount)
	}
}

func TestResourceIngestRouteOrderReadReflectsRemoteConfiguredOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"undeclared","pipeline":"filter true"},{"id":"configured-b","pipeline":"filter true"},{"id":"managed","pipeline":"filter true","managedBy":{"id":"manager"}},{"id":"configured-a","pipeline":"filter true"},{"id":"default","pipeline":""}]}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRouteOrder().Schema, map[string]interface{}{
		"type":      "otellogs",
		"route_ids": []interface{}{"configured-a", "configured-b"},
	})
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteOrderRead(context.Background(), data, client); diags.HasError() {
		t.Fatalf("read route order: %v", diags)
	}
	if got, want := makeStrSlice(data.Get("route_ids").([]interface{})), []string{"undeclared", "configured-b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("route_ids = %v, want remote priority prefix %v", got, want)
	}
}

func TestResourceIngestRouteOrderImportThenReadImportsEligibleRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"managed","pipeline":"filter managed","managedBy":{"id":"manager"}},{"id":"a","pipeline":"filter a"},{"id":"b","pipeline":"filter b"},{"id":"default","pipeline":""}]}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRouteOrder().Schema, map[string]interface{}{})
	data.SetId("otellogs")
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if _, err := resourceIngestRouteOrderImport(context.Background(), data, client); err != nil {
		t.Fatalf("import route order: %v", err)
	}
	if diags := resourceIngestRouteOrderRead(context.Background(), data, client); diags.HasError() {
		t.Fatalf("read imported route order: %v", diags)
	}
	if got, want := makeStrSlice(data.Get("route_ids").([]interface{})), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("route_ids = %v, want imported eligible routes %v", got, want)
	}
}

func TestRemoteRoutePriorityPrefix(t *testing.T) {
	testCases := []struct {
		name       string
		configured []string
		routes     []rest.IngestRouteResource
		want       []string
	}{
		{
			name:       "managed routes ignored",
			configured: []string{"a", "b"},
			routes: []rest.IngestRouteResource{
				{Id: "managed", Pipeline: "filter managed", ManagedBy: &rest.IngestRouteObjectRef{Id: "manager"}},
				{Id: "a", Pipeline: "filter a"},
				{Id: "b", Pipeline: "filter b"},
				{Id: "default", Pipeline: ""},
			},
			want: []string{"a", "b"},
		},
		{
			name:       "undeclared route ahead detected",
			configured: []string{"a", "b"},
			routes: []rest.IngestRouteResource{
				{Id: "c", Pipeline: "filter c"},
				{Id: "a", Pipeline: "filter a"},
				{Id: "b", Pipeline: "filter b"},
			},
			want: []string{"c", "a"},
		},
		{
			name:       "configured reorder detected",
			configured: []string{"a", "b"},
			routes: []rest.IngestRouteResource{
				{Id: "b", Pipeline: "filter b"},
				{Id: "a", Pipeline: "filter a"},
				{Id: "c", Pipeline: "filter c"},
			},
			want: []string{"b", "a"},
		},
		{
			name:       "unchanged prefix stable",
			configured: []string{"a", "b"},
			routes: []rest.IngestRouteResource{
				{Id: "a", Pipeline: "filter a"},
				{Id: "b", Pipeline: "filter b"},
				{Id: "c", Pipeline: "filter c"},
			},
			want: []string{"a", "b"},
		},
		{
			name:       "too short remote",
			configured: []string{"a", "b"},
			routes:     []rest.IngestRouteResource{{Id: "a", Pipeline: "filter a"}},
			want:       []string{"a"},
		},
		{
			name:       "empty configured imports all eligible routes",
			configured: nil,
			routes: []rest.IngestRouteResource{
				{Id: "managed", Pipeline: "filter managed", ManagedBy: &rest.IngestRouteObjectRef{Id: "manager"}},
				{Id: "a", Pipeline: "filter a"},
				{Id: "b", Pipeline: "filter b"},
				{Id: "default", Pipeline: ""},
			},
			want: []string{"a", "b"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := remoteRoutePriorityPrefix(testCase.configured, testCase.routes); !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("remoteRoutePriorityPrefix() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestResourceIngestRouteOrderReadDropsMissingManagedAndDefaultConfiguredRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"managed","pipeline":"filter true","managedBy":{"id":"manager"}},{"id":"default","pipeline":""},{"id":"configured","pipeline":"filter true"}]}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRouteOrder().Schema, map[string]interface{}{
		"type":      "otellogs",
		"route_ids": []interface{}{"missing", "managed", "default", "configured"},
	})
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteOrderRead(context.Background(), data, client); diags.HasError() {
		t.Fatalf("read route order: %v", diags)
	}
	if got, want := makeStrSlice(data.Get("route_ids").([]interface{})), []string{"configured"}; !reflect.DeepEqual(got, want) {
		t.Errorf("route_ids = %v, want %v", got, want)
	}
}

func TestResourceIngestRouteReadClearsManagedAndDefaultRoutes(t *testing.T) {
	for _, testCase := range []struct {
		name string
		body string
	}{
		{name: "managed", body: `{"id":"41030001","type":"otellogs","pipeline":"filter true","managedBy":{"id":"manager"}}`},
		{name: "default", body: `{"id":"41030001","type":"otellogs","pipeline":""}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(testCase.body))
			}))
			defer server.Close()

			data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{})
			data.SetId("otellogs/41030001")
			client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
			if diags := resourceIngestRouteRead(context.Background(), data, client); diags.HasError() {
				t.Fatalf("read ingest route: %v", diags)
			}
			if data.Id() != "" {
				t.Errorf("state ID = %q, want cleared", data.Id())
			}
		})
	}
}

func TestResourceIngestRouteImportRejectsManagedAndDefaultRoutes(t *testing.T) {
	for _, testCase := range []struct {
		name string
		body string
	}{
		{name: "managed", body: `{"id":"41030001","type":"otellogs","pipeline":"filter true","managedBy":{"id":"manager"}}`},
		{name: "default", body: `{"id":"41030001","type":"otellogs","pipeline":""}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(testCase.body))
			}))
			defer server.Close()

			data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{})
			data.SetId("otellogs/41030001")
			client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
			if _, err := resourceIngestRouteImport(context.Background(), data, client); err == nil {
				t.Fatal("expected import rejection")
			}
		})
	}
}

func TestResourceIngestRouteImportSetsCompositeStateID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"41030001","type":"otellogs","pipeline":"filter true"}`))
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRoute().Schema, map[string]interface{}{})
	data.SetId("otellogs/41030001")
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	imported, err := resourceIngestRouteImport(context.Background(), data, client)
	if err != nil {
		t.Fatalf("import ingest route: %v", err)
	}
	if len(imported) != 1 || imported[0].Id() != "otellogs/41030001" {
		t.Fatalf("unexpected imported state %#v", imported)
	}
}

func TestResourceIngestRouteOrderReconcileOmitsManagedRoutes(t *testing.T) {
	var ordering []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"managed","pipeline":"filter managed","managedBy":{"id":"manager"}},{"id":"other","pipeline":"filter other"},{"id":"configured","pipeline":"filter configured"},{"id":"default","pipeline":""}]}`))
		case http.MethodPatch:
			var body rest.IngestRoutePriorityUpdateRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode ordering request: %v", err)
				return
			}
			ordering = body.Ordering
			_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"configured","pipeline":"filter configured"},{"id":"other","pipeline":"filter other"},{"id":"default","pipeline":""}]}`))
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRouteOrder().Schema, map[string]interface{}{
		"type":      "otellogs",
		"route_ids": []interface{}{"configured"},
	})
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteOrderReconcile(context.Background(), data, client); diags.HasError() {
		t.Fatalf("reconcile route order: %v", diags)
	}
	want := []string{"configured", "other", "default"}
	if !reflect.DeepEqual(ordering, want) {
		t.Errorf("ordering = %v, want %v", ordering, want)
	}
}

func TestResourceIngestRouteOrderReconcileRetriesBadOrderingAfterRelist(t *testing.T) {
	listRequests := 0
	patchRequests := 0
	var secondOrdering []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			listRequests++
			if listRequests == 1 {
				_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"configured","pipeline":"filter configured"},{"id":"other","pipeline":"filter other"},{"id":"default","pipeline":""}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"ingestRoutes":[{"id":"configured","pipeline":"filter configured"},{"id":"concurrent","pipeline":"filter concurrent"},{"id":"other","pipeline":"filter other"},{"id":"default","pipeline":""}]}`))
		case http.MethodPatch:
			patchRequests++
			var body rest.IngestRoutePriorityUpdateRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode ordering request: %v", err)
				return
			}
			if patchRequests == 1 {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte(`{"message":"bad ordering"}`))
				return
			}
			secondOrdering = body.Ordering
			_, _ = writer.Write([]byte(`{"ingestRoutes":[]}`))
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()

	data := schema.TestResourceDataRaw(t, resourceIngestRouteOrder().Schema, map[string]interface{}{
		"type":      "otellogs",
		"route_ids": []interface{}{"configured"},
	})
	client := &observeclient.Client{Rest: rest.New(server.URL, server.Client())}
	if diags := resourceIngestRouteOrderReconcile(context.Background(), data, client); diags.HasError() {
		t.Fatalf("reconcile route order: %v", diags)
	}
	if patchRequests != 2 {
		t.Errorf("PATCH requests = %d, want 2", patchRequests)
	}
	if got, want := secondOrdering, []string{"configured", "concurrent", "other", "default"}; !reflect.DeepEqual(got, want) {
		t.Errorf("retry ordering = %v, want %v", got, want)
	}
}
