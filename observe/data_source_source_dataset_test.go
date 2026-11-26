package observe

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const testIcebergSourceDatasetResponse = `{"id":"41000002","label":"tf-pkg/Logs \"x\"","description":null,"objectTags":{"team":["a","b"]},` +
	`"dataRetentionDays":null,"effectiveDataRetentionDays":30,"dataSource":{"kind":"Ingest","ingest":{"storage":"Iceberg","schemaType":"OtelLogs",` +
	`"iceberg":{"storageIntegration":{"id":"41009999"},"externalWrite":true}}}}`

func readSourceDatasetDataSource(t *testing.T, config map[string]interface{}, handler http.HandlerFunc) (*schema.ResourceData, error) {
	t.Helper()
	d := schema.TestResourceDataRaw(t, dataSourceSourceDataset().Schema, config)
	diags := dataSourceSourceDatasetRead(context.Background(), d, newSourceDatasetTestClient(t, handler))
	if diags.HasError() {
		return d, fmt.Errorf("%s", diags[0].Summary)
	}
	return d, nil
}

func TestDataSourceSourceDatasetSchema(t *testing.T) {
	if err := dataSourceSourceDataset().InternalValidate(nil, false); err != nil {
		t.Fatalf("invalid data source schema: %s", err)
	}
}

func TestDataSourceSourceDatasetByID(t *testing.T) {
	d, err := readSourceDatasetDataSource(t, map[string]interface{}{"id": "41000002"}, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/source-datasets/41000002" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_, _ = w.Write([]byte(testIcebergSourceDatasetResponse))
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"label":                         `tf-pkg/Logs "x"`,
		"storage":                       "Iceberg",
		"schema_type":                   "OtelLogs",
		"storage_integration":           testStorageIntegrationOID,
		"external_write":                true,
		"data_retention_days":           sourceDatasetRetentionInherit,
		"effective_data_retention_days": 30,
		"oid":                           "o:::dataset:41000002",
	}
	for k, v := range want {
		if got := d.Get(k); got != v {
			t.Errorf("%s = %#v, want %#v", k, got, v)
		}
	}
	if got := d.Get("object_tags.team"); got != "a,b" {
		t.Errorf("object_tags.team = %#v, want %q", got, "a,b")
	}
	if d.Id() != "41000002" {
		t.Errorf("id = %q", d.Id())
	}
}

func TestDataSourceSourceDatasetByLabel(t *testing.T) {
	list := func(items ...string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Query().Get("filter"), `label == "tf-pkg/Logs \"x\""`; r.URL.Path != "/v1/source-datasets" || got != want {
				t.Errorf("unexpected request %s filter=%s, want filter=%s", r.URL.Path, got, want)
			}
			_, _ = fmt.Fprintf(w, `{"sourceDatasets":[%s],"meta":{"totalCount":%d}}`, strings.Join(items, ","), len(items))
		}
	}
	config := map[string]interface{}{"label": `tf-pkg/Logs "x"`}

	d, err := readSourceDatasetDataSource(t, config, list(testIcebergSourceDatasetResponse))
	if err != nil {
		t.Fatal(err)
	}
	if d.Id() != "41000002" || d.Get("storage") != "Iceberg" {
		t.Fatalf("unexpected result id=%q storage=%v", d.Id(), d.Get("storage"))
	}

	if _, err := readSourceDatasetDataSource(t, config, list()); err == nil || !strings.Contains(err.Error(), "no source dataset has label") {
		t.Fatalf("expected not found error, got %v", err)
	}

	other := strings.Replace(testIcebergSourceDatasetResponse, `"41000002"`, `"41000003"`, 1)
	if _, err := readSourceDatasetDataSource(t, config, list(testIcebergSourceDatasetResponse, other)); err == nil || !strings.Contains(err.Error(), "(ids 41000002, 41000003); set id instead") {
		t.Fatalf("expected ambiguous label error, got %v", err)
	}
}

func TestDataSourceSourceDatasetNotFound(t *testing.T) {
	_, err := readSourceDatasetDataSource(t, map[string]interface{}{"id": "41000009"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"not_found","message":"source dataset not found"}`))
	})
	if err == nil || err.Error() != "source dataset 41000009 not found" {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestAccObserveSourceDatasetDataSource(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "observe_source_dataset" "src" {
  label               = "%[1]s/Logs"
  description         = "looked up by a data source"
  data_retention_days = 7
  deletion_protection = false
}

data "observe_source_dataset" "by_id" {
  id = observe_source_dataset.src.id
}

data "observe_source_dataset" "by_label" {
  label = observe_source_dataset.src.label
}`, prefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.observe_source_dataset.by_id", "oid", "observe_source_dataset.src", "oid"),
					resource.TestCheckResourceAttrPair("data.observe_source_dataset.by_label", "id", "observe_source_dataset.src", "id"),
					resource.TestCheckResourceAttr("data.observe_source_dataset.by_id", "label", prefix+"/Logs"),
					resource.TestCheckResourceAttr("data.observe_source_dataset.by_id", "description", "looked up by a data source"),
					resource.TestCheckResourceAttr("data.observe_source_dataset.by_id", "storage", "Native"),
					resource.TestCheckResourceAttr("data.observe_source_dataset.by_id", "schema_type", "Any"),
					resource.TestCheckResourceAttr("data.observe_source_dataset.by_id", "external_write", "false"),
					resource.TestCheckResourceAttr("data.observe_source_dataset.by_label", "data_retention_days", "7"),
					resource.TestCheckResourceAttrSet("data.observe_source_dataset.by_label", "effective_data_retention_days"),
				),
			},
		},
	})
}
