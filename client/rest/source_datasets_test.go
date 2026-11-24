package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const sourceDatasetJSON = `{
  "id": "41000001",
  "label": "tf-pkg/Logs",
  "description": "d",
  "objectTags": {"team": ["a", "b"]},
  "dataRetentionDays": 30,
  "effectiveDataRetentionDays": 30,
  "dataSource": {"kind": "Ingest", "ingest": {"storage": "Iceberg", "schemaType": "Any",
    "iceberg": {"storageIntegration": {"id": "41009999"}, "externalWrite": true}}}
}`

func decodeBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestSourceDatasetClient(t *testing.T) {
	var gotForce, gotFilter string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/source-datasets":
			body := decodeBody(t, r)
			ingest := body["dataSource"].(map[string]interface{})["ingest"].(map[string]interface{})
			iceberg := ingest["iceberg"].(map[string]interface{})
			if iceberg["storageIntegrationId"] != "41009999" || iceberg["externalWrite"] != true {
				t.Errorf("unexpected iceberg body %#v", iceberg)
			}
			if ingest["storage"] != "Iceberg" || ingest["schemaType"] != "Any" {
				t.Errorf("unexpected ingest body %#v", ingest)
			}
			if body["dataRetentionDays"] != float64(0) {
				t.Errorf("dataRetentionDays = %#v, want 0", body["dataRetentionDays"])
			}
			if _, ok := body["description"]; ok {
				t.Errorf("unset description must be omitted")
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(sourceDatasetJSON))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/source-datasets/41000001":
			_, _ = w.Write([]byte(sourceDatasetJSON))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/source-datasets":
			gotFilter = r.URL.Query().Get("filter")
			_, _ = w.Write([]byte(`{"sourceDatasets": [` + sourceDatasetJSON + `]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/source-datasets/41000001":
			body := decodeBody(t, r)
			want := map[string]interface{}{"description": nil, "objectTags": map[string]interface{}{}}
			if !reflect.DeepEqual(body, want) {
				t.Errorf("patch body = %#v, want %#v", body, want)
			}
			_, _ = w.Write([]byte(sourceDatasetJSON))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/source-datasets/41000001":
			gotForce = r.URL.Query().Get("force")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	client := New(server.URL, server.Client())
	zero := int64(0)
	ds, err := client.CreateSourceDataset(ctx, &SourceDatasetCreateRequest{
		Label:             "tf-pkg/Logs",
		DataRetentionDays: &zero,
		DataSource: SourceDatasetDataSourceInput{Kind: "Ingest", Ingest: SourceDatasetIngestInput{
			Storage: "Iceberg", SchemaType: "Any",
			Iceberg: &SourceDatasetIcebergInput{StorageIntegrationId: "41009999", ExternalWrite: true},
		}},
	})
	if err != nil {
		t.Fatalf("CreateSourceDataset: %v", err)
	}
	if ds.Id != "41000001" || ds.DataSource.Ingest.Iceberg.StorageIntegration.Id != "41009999" || *ds.DataRetentionDays != 30 {
		t.Errorf("unexpected decode %#v", ds)
	}
	if _, err := client.GetSourceDataset(ctx, "41000001"); err != nil {
		t.Fatalf("GetSourceDataset: %v", err)
	}
	empty := map[string][]string{}
	if _, err := client.UpdateSourceDataset(ctx, "41000001", &SourceDatasetUpdateRequest{
		DescriptionSpecified: true,
		ObjectTags:           &empty,
	}); err != nil {
		t.Fatalf("UpdateSourceDataset: %v", err)
	}
	list, err := client.ListSourceDatasets(ctx, `label.matches("^tf-")`)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListSourceDatasets: %v %d", err, len(list))
	}
	if gotFilter != `label.matches("^tf-")` {
		t.Errorf("filter = %q", gotFilter)
	}
	if err := client.DeleteSourceDataset(ctx, "41000001", true); err != nil {
		t.Fatalf("DeleteSourceDataset: %v", err)
	}
	if gotForce != "true" {
		t.Errorf("force = %q, want true", gotForce)
	}
	if err := client.DeleteSourceDataset(ctx, "41000001", false); err != nil {
		t.Fatalf("DeleteSourceDataset: %v", err)
	}
	if gotForce != "" {
		t.Errorf("force = %q, want unset", gotForce)
	}
}

func TestSourceDatasetUpdateRequestMarshal(t *testing.T) {
	days := int64(7)
	cases := []struct {
		name string
		req  SourceDatasetUpdateRequest
		want string
	}{
		{"empty", SourceDatasetUpdateRequest{}, `{}`},
		{"description only", SourceDatasetUpdateRequest{Description: stringPointer("x"), DescriptionSpecified: true}, `{"description":"x"}`},
		{"clear icon", SourceDatasetUpdateRequest{IconSpecified: true}, `{"icon":null}`},
		{"retention", SourceDatasetUpdateRequest{DataRetentionDays: &days, DataRetentionDaysSpecified: true}, `{"dataRetentionDays":7}`},
		{"nil tags map sent as empty", SourceDatasetUpdateRequest{ObjectTags: new(map[string][]string)}, `{"objectTags":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestSourceDatasetErrorType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"type":"dependents_exist","message":"has dependents"}`))
	}))
	defer server.Close()
	err := New(server.URL, server.Client()).DeleteSourceDataset(context.Background(), "1", false)
	if !HasStatusCode(err, http.StatusConflict) || !HasErrorType(err, "dependents_exist") {
		t.Fatalf("unexpected error %v", err)
	}
}

func stringPointer(s string) *string { return &s }

func TestSourceDatasetClientHonorsContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a canceled context must not reach the server: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	client := New(server.URL, server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := map[string]func() error{
		"create": func() error {
			_, err := client.CreateSourceDataset(ctx, &SourceDatasetCreateRequest{Label: "tf-pkg/a"})
			return err
		},
		"get": func() error { _, err := client.GetSourceDataset(ctx, "1"); return err },
		"update": func() error {
			_, err := client.UpdateSourceDataset(ctx, "1", &SourceDatasetUpdateRequest{})
			return err
		},
		"delete": func() error { return client.DeleteSourceDataset(ctx, "1", true) },
		"list":   func() error { _, err := client.ListSourceDatasets(ctx, ""); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, context.Canceled) {
			t.Errorf("%s: expected context.Canceled, got %v", name, err)
		}
	}
}
