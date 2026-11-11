package observe

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func decodeReferenceTableResponse(t *testing.T, body string) *rest.ReferenceTable {
	t.Helper()
	var refTable rest.ReferenceTable
	if err := json.Unmarshal([]byte(body), &refTable); err != nil {
		t.Fatalf("decode reference table: %v", err)
	}
	return &refTable
}

func referenceTableDatasetWithColumns(columns ...string) *gql.Dataset {
	fields := make([]gql.DatasetTypedefDefObjectTypedefFieldsObjectFieldDef, len(columns))
	for i, column := range columns {
		fields[i].Name = column
		fields[i].Type.Rep = "string"
	}
	return &gql.Dataset{Typedef: gql.DatasetTypedef{Def: &gql.DatasetTypedefDefObjectTypedef{Fields: fields}}}
}

func TestReferenceTableToResourceDataReadsPrimaryKeyAndLabelField(t *testing.T) {
	refTable := decodeReferenceTableResponse(t, `{
		"id": "41000001",
		"label": "hosts",
		"checksum": "9b2cf535f27731c974343645a3985328",
		"datasetId": "41000002",
		"primaryKey": ["host", "region"],
		"labelField": "owner"
	}`)
	dataset := referenceTableDatasetWithColumns("host", "region", "owner")

	for name, resourceSchema := range map[string]map[string]*schema.Schema{
		"resource":    resourceReferenceTable().Schema,
		"data source": dataSourceReferenceTable().Schema,
	} {
		t.Run(name, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, resourceSchema, map[string]interface{}{})
			if diags := referenceTableToResourceData(refTable, dataset, data); diags.HasError() {
				t.Fatalf("map reference table: %v", diags)
			}
			if got, want := data.Get("primary_key").([]interface{}), []interface{}{"host", "region"}; !equalInterfaceSlices(got, want) {
				t.Errorf("primary_key = %v, want %v", got, want)
			}
			if got, want := data.Get("label_field"), "owner"; got != want {
				t.Errorf("label_field = %q, want %q", got, want)
			}
			if got, want := data.Get("checksum"), "9b2cf535f27731c974343645a3985328"; got != want {
				t.Errorf("checksum = %q, want %q", got, want)
			}
		})
	}
}

func TestReferenceTableToResourceDataWithoutPrimaryKeyOrLabelField(t *testing.T) {
	refTable := decodeReferenceTableResponse(t, `{
		"id": "41000001",
		"label": "hosts",
		"checksum": "9b2cf535f27731c974343645a3985328",
		"datasetId": "41000002",
		"primaryKey": null
	}`)

	data := schema.TestResourceDataRaw(t, resourceReferenceTable().Schema, map[string]interface{}{})
	if diags := referenceTableToResourceData(refTable, referenceTableDatasetWithColumns("host"), data); diags.HasError() {
		t.Fatalf("map reference table: %v", diags)
	}
	if got := data.Get("primary_key.#"); got != 0 {
		t.Errorf("primary_key.# = %v, want 0", got)
	}
	if got := data.Get("label_field"); got != "" {
		t.Errorf("label_field = %q, want empty", got)
	}
}

func equalInterfaceSlices(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
