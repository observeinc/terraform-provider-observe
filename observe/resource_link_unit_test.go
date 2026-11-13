package observe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

// linkTestServer answers the link GraphQL operations with dfk and records the
// operation names it receives. Until the link is created, getDeferredForeignKey
// returns NOT_FOUND if missing is set.
func linkTestServer(t *testing.T, dfk string, missing ...bool) (*observeclient.Client, *[]string) {
	var ops []string
	notFound := len(missing) > 0 && missing[0]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		ops = append(ops, req.OperationName)
		w.Header().Set("Content-Type", "application/json")
		switch req.OperationName {
		case "getDeferredForeignKey":
			if notFound {
				_, _ = w.Write([]byte(`{"data":null,"errors":[{"message":"not found","extensions":{"code":"NOT_FOUND"}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"deferredForeignKey":` + dfk + `}}`))
		case "createDeferredForeignKey":
			notFound = false
			_, _ = w.Write([]byte(`{"data":{"deferredForeignKey":` + dfk + `}}`))
		case "updateDeferredForeignKey":
			_, _ = w.Write([]byte(`{"data":{"deferredForeignKey":` + dfk + `}}`))
		default:
			t.Errorf("unexpected operation %q", req.OperationName)
		}
	}))
	t.Cleanup(server.Close)
	m, err := gql.New(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return &observeclient.Client{Config: &observeclient.Config{}, Meta: m}, &ops
}

const linkTestDFK = `{
	"id": "41000301",
	"workspaceId": "41000001",
	"sourceDataset": {"datasetId": "41000101"},
	"targetDataset": {"datasetId": "41000102"},
	"srcFields": ["key"],
	"dstFields": ["key"],
	"label": "test-link",
	"resolution": {"sourceId": "41000101", "targetId": "41000102"},
	"status": {"errorText": ""},
	"managedById": null
}`

// linkUpdateData is a link whose config moved source and target to a new dataset
// version, like every link on a dataset does when the dataset is edited.
func linkUpdateData(t *testing.T, fields []interface{}, label string) *schema.ResourceData {
	raw := map[string]interface{}{
		"source": "o:::dataset:41000101/2026-01-02T03:04:05Z",
		"target": "o:::dataset:41000102/2026-01-02T03:04:05Z",
		"fields": fields,
	}
	if label != "" {
		raw["label"] = label
	}
	data := schema.TestResourceDataRaw(t, resourceLink().Schema, raw)
	data.SetId("41000301")
	return data
}

func TestResourceLinkUpdateSkipsUnchangedLink(t *testing.T) {
	for name, tc := range map[string]struct {
		fields []interface{}
		label  string
	}{
		"label unset":       {fields: []interface{}{"key"}},
		"label matches":     {fields: []interface{}{"key"}, label: "test-link"},
		"same-name mapping": {fields: []interface{}{"key:key"}},
	} {
		t.Run(name, func(t *testing.T) {
			client, ops := linkTestServer(t, linkTestDFK)
			if diags := resourceLinkUpdate(context.Background(), linkUpdateData(t, tc.fields, tc.label), client); diags.HasError() {
				t.Fatalf("update: %v", diags)
			}
			if slices.Contains(*ops, "updateDeferredForeignKey") {
				t.Fatalf("sent updateDeferredForeignKey for an unchanged link; ops = %v", *ops)
			}
		})
	}
}

func TestResourceLinkUpdateSendsRealChange(t *testing.T) {
	for name, tc := range map[string]struct {
		dfk    string
		fields []interface{}
		label  string
	}{
		"fields changed":       {dfk: linkTestDFK, fields: []interface{}{"key", "other"}},
		"target field renamed": {dfk: linkTestDFK, fields: []interface{}{"key:otherKey"}},
		"label changed":        {dfk: linkTestDFK, fields: []interface{}{"key"}, label: "renamed"},
		"source moved":         {dfk: strings.Replace(linkTestDFK, `"sourceDataset": {"datasetId": "41000101"}`, `"sourceDataset": {"datasetId": "41000199"}`, 1), fields: []interface{}{"key"}},
		"managed by someone":   {dfk: strings.Replace(linkTestDFK, `"managedById": null`, `"managedById": "41000777"`, 1), fields: []interface{}{"key"}},
	} {
		t.Run(name, func(t *testing.T) {
			client, ops := linkTestServer(t, tc.dfk)
			if diags := resourceLinkUpdate(context.Background(), linkUpdateData(t, tc.fields, tc.label), client); diags.HasError() {
				t.Fatalf("update: %v", diags)
			}
			if !slices.Contains(*ops, "updateDeferredForeignKey") {
				t.Fatalf("did not send updateDeferredForeignKey; ops = %v", *ops)
			}
		})
	}
}

func TestResourceLinkUpdateUnchangedButBrokenFails(t *testing.T) {
	dfk := strings.Replace(linkTestDFK, `"errorText": ""`, `"errorText": "field \"key\" not present in source dataset"`, 1)
	client, ops := linkTestServer(t, dfk)

	diags := resourceLinkUpdate(context.Background(), linkUpdateData(t, []interface{}{"key"}, ""), client)
	if !diags.HasError() {
		t.Fatal("expected an error for a link the dataset save broke")
	}
	if got, want := diags[0].Summary, `failed to update foreign key: field "key" not present in source dataset`; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if slices.Contains(*ops, "updateDeferredForeignKey") {
		t.Errorf("sent updateDeferredForeignKey; ops = %v", *ops)
	}
}

// linkApply runs a version-only update of the link through the SDK, as terraform
// apply does, and returns the state the SDK hands back to Terraform to save.
func linkApply(t *testing.T, client *observeclient.Client, oldSource, newSource string) (*terraform.InstanceState, diag.Diagnostics) {
	state := &terraform.InstanceState{
		ID: "41000301",
		Attributes: map[string]string{
			"id":       "41000301",
			"source":   oldSource,
			"target":   "o:::dataset:41000102",
			"fields.#": "1",
			"fields.0": "key",
			"label":    "test-link",
		},
	}
	diff := &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{
		"source": {Old: oldSource, New: newSource},
	}}
	return resourceLink().Apply(context.Background(), state, diff, client)
}

func TestResourceLinkUpdateBrokenKeepsPreviousState(t *testing.T) {
	const oldSource = "o:::dataset:41000101/2026-01-01T00:00:00Z"
	const newSource = "o:::dataset:41000101/2026-01-02T03:04:05Z"

	broken := strings.Replace(linkTestDFK, `"errorText": ""`, `"errorText": "field \"key\" not present in source dataset"`, 1)
	client, _ := linkTestServer(t, broken)
	got, diags := linkApply(t, client, oldSource, newSource)
	if !diags.HasError() {
		t.Fatal("expected an error for a link the dataset save broke")
	}
	// The old oid keeps the update in the next plan, so the next apply fails
	// again until the link is fixed.
	if src := got.Attributes["source"]; src != oldSource {
		t.Errorf("saved source = %q, want the previous %q", src, oldSource)
	}

	client, _ = linkTestServer(t, linkTestDFK)
	got, diags = linkApply(t, client, oldSource, newSource)
	if diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
	if src := got.Attributes["source"]; src != newSource {
		t.Errorf("saved source = %q, want the planned %q", src, newSource)
	}
}

func TestResourceLinkUpdateRecreatesMissingLink(t *testing.T) {
	client, ops := linkTestServer(t, linkTestDFK, true)
	data := linkUpdateData(t, []interface{}{"key"}, "")
	// Resolving the workspace from the oid keeps the create off the workspace API.
	if err := data.Set("workspace", "o:::workspace:41000001"); err != nil {
		t.Fatal(err)
	}

	if diags := resourceLinkUpdate(context.Background(), data, client); diags.HasError() {
		t.Fatalf("update: %v", diags)
	}
	if !slices.Contains(*ops, "createDeferredForeignKey") || slices.Contains(*ops, "updateDeferredForeignKey") {
		t.Fatalf("want a create and no update for a link deleted outside Terraform; ops = %v", *ops)
	}
}

func TestResourceLinkUpdateNotFoundOnUpdateFails(t *testing.T) {
	// The link existed when read, then was deleted before the update.
	var ops []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OperationName string `json:"operationName"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		ops = append(ops, req.OperationName)
		w.Header().Set("Content-Type", "application/json")
		if req.OperationName == "getDeferredForeignKey" {
			_, _ = w.Write([]byte(`{"data":{"deferredForeignKey":` + linkTestDFK + `}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":null,"errors":[{"message":"not found","extensions":{"code":"NOT_FOUND"}}]}`))
	}))
	t.Cleanup(server.Close)
	m, err := gql.New(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client := &observeclient.Client{Config: &observeclient.Config{}, Meta: m}

	data := linkUpdateData(t, []interface{}{"key", "other"}, "")
	// With the workspace known, a create would reach createDeferredForeignKey.
	if err := data.Set("workspace", "o:::workspace:41000001"); err != nil {
		t.Fatal(err)
	}

	diags := resourceLinkUpdate(context.Background(), data, client)
	if !diags.HasError() {
		t.Fatal("expected an error when the link disappears between the read and the update")
	}
	if slices.Contains(ops, "createDeferredForeignKey") {
		t.Errorf("created the link; ops = %v", ops)
	}
}

func TestLinkMatchesConfigManagedBy(t *testing.T) {
	managedBy := "41000777"
	other := "41000888"
	input := &gql.DeferredForeignKeyInput{
		SourceDataset: &gql.DeferredDatasetReferenceInput{DatasetId: stringPtr("1")},
		TargetDataset: &gql.DeferredDatasetReferenceInput{DatasetId: stringPtr("2")},
		SrcFields:     []string{"key"},
		DstFields:     []string{"key"},
	}
	link := func(managedById *string) *gql.DeferredForeignKey {
		return &gql.DeferredForeignKey{
			SourceDataset: &gql.DeferredForeignKeySourceDatasetDeferredDatasetReference{DatasetId: stringPtr("1")},
			TargetDataset: &gql.DeferredForeignKeyTargetDatasetDeferredDatasetReference{DatasetId: stringPtr("2")},
			SrcFields:     []string{"key"},
			DstFields:     []string{"key"},
			ManagedById:   managedById,
		}
	}
	for name, tc := range map[string]struct {
		server, client *string
		want           bool
	}{
		"neither":           {want: true},
		"same":              {server: &managedBy, client: &managedBy, want: true},
		"client sets one":   {client: &managedBy},
		"update clears one": {server: &managedBy},
		"different":         {server: &managedBy, client: &other},
	} {
		if got := linkMatchesConfig(link(tc.server), input, tc.client); got != tc.want {
			t.Errorf("%s: linkMatchesConfig = %v, want %v", name, got, tc.want)
		}
	}
}
