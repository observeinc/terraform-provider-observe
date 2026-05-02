# DatasetRef nested-shape migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update `terraform-provider-observe`'s REST client to transparently accept both the current flat `DatasetRef {id, label?}` JSON shape and the upcoming nested `{id, record?: DatasetBrief}` shape, so the provider release can ship before the API server rolls over for the single affected customer.

**Architecture:** The `DatasetRef` Go struct is reshaped in-memory to the nested form (`Id string; Record *DatasetBrief`). A custom `UnmarshalJSON` sniffs which wire shape arrived and normalizes both into the nested representation, so the single consumer (`GetInboundShareTable`) is shape-agnostic. The provider's Terraform attribute schema and the GraphQL-enrichment fallback in `observe/resource_inbound_share_table.go` are untouched.

**Tech Stack:** Go 1.24, `encoding/json`, `net/http/httptest`, standard `testing` package. No new dependencies.

**Companion spec:** `docs/superpowers/specs/2026-05-02-datasetref-migration-design.md`. The follow-up cleanup PR (delete the flat branch from `UnmarshalJSON` after rollover completes) is scoped there, not in this plan.

---

## File Structure

### Files this plan modifies
- `client/rest/inbound_share_types.go` — add `encoding/json` import, add `DatasetBrief`, reshape `DatasetRef`, add `UnmarshalJSON` on `*DatasetRef`.
- `client/rest/inbound_share_types_test.go` — add eight `TestDatasetRef_UnmarshalJSON_*` cases (one a table test with four subtests) plus a `TestTrackTableResponseMarshaling_NestedShape` sibling.
- `client/rest/inbound_share_tables.go` — update `GetInboundShareTable` (lines 59–74) to read `SourceDataset.Record.Label`.
- `client/rest/inbound_share_tables_test.go` — update the existing `.Label` assertion in `TestUpdateInboundShareTable`, add `TestGetInboundShareTable_NestedShape`, add `TestGetInboundShareTable_IdOnlyResponse`.

### Files this plan does NOT touch
- `observe/resource_inbound_share_table.go` — the `enrichResultFromDataset` GraphQL fallback stays as-is (spec §2).
- Any Terraform attribute schema or state-migration hook — no user-visible schema change.
- `CHANGELOG.md` — the file is frozen; release notes go in the GitHub Release, handled out-of-band by the release operator.

### Import change in `inbound_share_types.go`
The existing single-line import expands to a multi-line block that adds `encoding/json` alongside the existing `client/oid` import. The exact text is given in Task 2 Step 1.

---

## Task 1: Add a failing decoder test for the nested shape

**Purpose:** Start with a red test that exercises the new nested-shape decoding contract. This forces the type reshape in Task 2 and proves the happy-path decoder branch before we touch any consumer.

**Files:**
- Modify: `client/rest/inbound_share_types_test.go` — append a new test function at end of file.

- [ ] **Step 1: Append the failing test**

Append this to the end of `client/rest/inbound_share_types_test.go`:

```go
func TestDatasetRef_UnmarshalJSON_NestedShape(t *testing.T) {
	jsonData := `{
		"id": "41067890",
		"record": {
			"label": "Customer Events",
			"description": "Snowflake share for customer events",
			"icon": "icons/event"
		}
	}`

	var ref DatasetRef
	if err := json.Unmarshal([]byte(jsonData), &ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ref.Id != "41067890" {
		t.Errorf("expected Id 41067890, got %q", ref.Id)
	}
	if ref.Record == nil {
		t.Fatal("expected Record to be non-nil")
	}
	if ref.Record.Label != "Customer Events" {
		t.Errorf("expected Record.Label %q, got %q", "Customer Events", ref.Record.Label)
	}
	if ref.Record.Description != "Snowflake share for customer events" {
		t.Errorf("expected Record.Description %q, got %q", "Snowflake share for customer events", ref.Record.Description)
	}
	if ref.Record.Icon != "icons/event" {
		t.Errorf("expected Record.Icon %q, got %q", "icons/event", ref.Record.Icon)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./client/rest/... -run TestDatasetRef_UnmarshalJSON_NestedShape`
Expected output (build failure — not test failure):
```
./inbound_share_types_test.go: undefined: DatasetBrief  (or similar)
FAIL    github.com/observeinc/terraform-provider-observe/client/rest [build failed]
```
The test cannot compile because `DatasetRef.Record` and `DatasetBrief` do not exist yet. This is the intended red state.

- [ ] **Step 3: Do NOT commit yet**

The test file is in an uncompilable state. The type reshape in Task 2 lands as a single atomic change with this test; we commit together there.

---

## Task 2: Reshape types, implement `UnmarshalJSON`, and update both in-tree consumers so the project compiles again

**Purpose:** Atomic change that adds the new types, implements the decoder, and updates the only two files that read the old flat `.Label` field. After this task, `go build ./...` and `go test ./client/rest/...` pass, with the new decoder test from Task 1 passing on the nested branch.

**Files:**
- Modify: `client/rest/inbound_share_types.go` — import, new type, reshape, new method.
- Modify: `client/rest/inbound_share_tables.go:59-74` — read `Record.Label` in `GetInboundShareTable`.
- Modify: `client/rest/inbound_share_tables_test.go:187-189` — update `.Label` assertion in `TestUpdateInboundShareTable`.

- [ ] **Step 1: Update the import block in `inbound_share_types.go`**

Replace the existing single-line import (line 3) with the multi-line form:

```go
import (
	"encoding/json"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)
```

- [ ] **Step 2: Add `DatasetBrief` and reshape `DatasetRef`**

Replace lines 21–25 (the old `DatasetRef` definition) of `client/rest/inbound_share_types.go`:

```go
// DatasetRef represents a reference to a dataset
type DatasetRef struct {
	Id    string `json:"id"`
	Label string `json:"label,omitempty"`
}
```

...with:

```go
// DatasetRef represents a reference to a dataset. The Record field is
// populated when the server-side expand projection is honoured; consumers
// that need the human-readable metadata read Record.Label.
//
// NOTE: the custom UnmarshalJSON below accepts both the legacy flat
// {id, label} wire shape and the new nested {id, record} shape. After the
// server-side migration completes, the follow-up cleanup PR (see
// docs/superpowers/specs/2026-05-02-datasetref-migration-design.md §7)
// deletes UnmarshalJSON; the default struct unmarshaller is sufficient for
// nested-only wire traffic.
type DatasetRef struct {
	Id     string        `json:"id"`
	Record *DatasetBrief `json:"record,omitempty"`
}

// DatasetBrief carries the embedded dataset metadata that accompanies a
// dataset reference when the server-side expand projection is honoured.
type DatasetBrief struct {
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// UnmarshalJSON accepts BOTH the legacy flat shape ({"id":..., "label":...})
// and the new nested shape ({"id":..., "record":{"label":...}}). The legacy
// shape is normalised into the nested representation so consumers only deal
// with one in-memory form. When both the top-level label and the nested
// record are present (possible during a transitional rollout), the nested
// form wins. A top-level JSON null decodes to DatasetRef{Id:"", Record:nil}
// via the default code path — no special case is needed.
//
// An empty Id ("") is not rejected at decode time: it is up to the caller
// (currently GetInboundShareTable in inbound_share_tables.go, which checks)
// to validate the invariant. Future consumers that read DatasetRef.Id
// directly inherit that responsibility.
func (d *DatasetRef) UnmarshalJSON(data []byte) error {
	var raw struct {
		Id     string        `json:"id"`
		Label  string        `json:"label"`
		Record *DatasetBrief `json:"record"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.Id = raw.Id
	switch {
	case raw.Record != nil:
		d.Record = raw.Record
	case raw.Label != "":
		d.Record = &DatasetBrief{Label: raw.Label}
	default:
		d.Record = nil
	}
	return nil
}
```

- [ ] **Step 3: Update the consumer in `inbound_share_tables.go`**

In `client/rest/inbound_share_tables.go`, replace lines 59–74 (the `if table.SourceDataset != nil { ... } else { ... }` block inside `GetInboundShareTable`):

```go
	// If there's a source dataset, populate it
	if table.SourceDataset != nil {
		// Validate that we have required fields
		if table.SourceDataset.Id == "" {
			return nil, fmt.Errorf("API response missing sourceDataset.id for table %s", tableId)
		}
		if table.SourceDataset.Label == "" {
			return nil, fmt.Errorf("API response missing sourceDataset.label for table %s", tableId)
		}

		result.Dataset = InboundShareDataset{
			Id:    table.SourceDataset.Id,
			Label: table.SourceDataset.Label,
		}
	} else {
		return nil, fmt.Errorf("API response missing sourceDataset for table %s", tableId)
	}
```

...with:

```go
	// If there's a source dataset, populate it. DatasetRef.UnmarshalJSON
	// normalises both the legacy flat shape and the nested shape into
	// SourceDataset.Record, so the label is always read from Record.Label.
	// The Record-nil and empty-Label branches are reported separately so
	// operators can distinguish a missing record from a record whose label
	// happened to be empty.
	if table.SourceDataset == nil {
		return nil, fmt.Errorf("API response missing sourceDataset for table %s", tableId)
	}
	if table.SourceDataset.Id == "" {
		return nil, fmt.Errorf("API response missing sourceDataset.id for table %s", tableId)
	}
	if table.SourceDataset.Record == nil {
		return nil, fmt.Errorf("API response missing sourceDataset.record for table %s", tableId)
	}
	if table.SourceDataset.Record.Label == "" {
		return nil, fmt.Errorf("API response missing sourceDataset.record.label for table %s", tableId)
	}

	result.Dataset = InboundShareDataset{
		Id:    table.SourceDataset.Id,
		Label: table.SourceDataset.Record.Label,
	}
```

- [ ] **Step 4: Update the test assertion in `inbound_share_tables_test.go`**

In `client/rest/inbound_share_tables_test.go`, replace lines 187–189 (inside `TestUpdateInboundShareTable`):

```go
	if table.SourceDataset.Label != "Updated Label" {
		t.Errorf("Expected dataset label 'Updated Label', got %s", table.SourceDataset.Label)
	}
```

...with:

```go
	if table.SourceDataset.Record == nil {
		t.Fatalf("Expected SourceDataset.Record to be non-nil")
	}
	if table.SourceDataset.Record.Label != "Updated Label" {
		t.Errorf("Expected dataset label 'Updated Label', got %s", table.SourceDataset.Record.Label)
	}
```

The server fixture in this test still returns the flat shape (`"sourceDataset": {"id":"...", "label":"Updated Label"}`). That's deliberate — it exercises the flat → nested normalization end-to-end.

- [ ] **Step 5: Confirm the full tree builds**

Run: `go build ./...`
Expected: exits 0 with no output.

- [ ] **Step 6: Confirm `client/rest` tests pass**

Run: `go test ./client/rest/... -count=1`
Expected: all tests pass, including the new `TestDatasetRef_UnmarshalJSON_NestedShape` and the existing `TestTrackTable`, `TestGetInboundShareTable`, `TestUpdateInboundShareTable` (which exercise the flat wire shape and prove the normalization path).

- [ ] **Step 7: Run `go vet`**

Run: `go vet ./...`
Expected: exits 0 with no output.

- [ ] **Step 8: Commit**

```bash
git add client/rest/inbound_share_types.go client/rest/inbound_share_types_test.go \
        client/rest/inbound_share_tables.go client/rest/inbound_share_tables_test.go
git commit -m "$(cat <<'EOF'
feat(rest): reshape DatasetRef to nested form with dual-shape decoder

The sharein API's DatasetRef JSON shape is migrating from flat
{id, label?} to nested {id, record?: DatasetBrief}. Add a custom
UnmarshalJSON that accepts both shapes and normalises to the nested
form, so the provider release can ship before the server rollover
reaches affected tenants.

- client/rest/inbound_share_types.go: add DatasetBrief, reshape
  DatasetRef to {Id, Record *DatasetBrief}, add UnmarshalJSON.
- client/rest/inbound_share_tables.go: GetInboundShareTable now reads
  SourceDataset.Record.Label (populated from either wire shape by the
  decoder).
- Tests updated accordingly; the decoder branch coverage and the
  end-to-end nested-wire-shape fixtures land in follow-up commits.
EOF
)"
```

---

## Task 3: Add the remaining decoder unit tests (flat, id-only, both, empty, malformed, wrong-type, adversarial)

**Purpose:** Lock in coverage for every branch of `UnmarshalJSON` as regression guards, including adversarial cases: non-object `record`, explicit JSON nulls, duplicate keys. Each test is deterministic and self-contained.

**Files:**
- Modify: `client/rest/inbound_share_types_test.go` — append seven new test functions (one of which is a table test with four subtests).

- [ ] **Step 1: Add the flat-shape test**

Append to `client/rest/inbound_share_types_test.go`:

```go
func TestDatasetRef_UnmarshalJSON_FlatShape(t *testing.T) {
	jsonData := `{"id":"41067890","label":"Customer Events"}`

	var ref DatasetRef
	if err := json.Unmarshal([]byte(jsonData), &ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ref.Id != "41067890" {
		t.Errorf("expected Id 41067890, got %q", ref.Id)
	}
	if ref.Record == nil {
		t.Fatal("expected Record to be synthesised for flat shape")
	}
	if ref.Record.Label != "Customer Events" {
		t.Errorf("expected Record.Label %q, got %q", "Customer Events", ref.Record.Label)
	}
	if ref.Record.Description != "" {
		t.Errorf("expected Record.Description empty for flat shape, got %q", ref.Record.Description)
	}
	if ref.Record.Icon != "" {
		t.Errorf("expected Record.Icon empty for flat shape, got %q", ref.Record.Icon)
	}
}
```

- [ ] **Step 2: Add the id-only test**

Append:

```go
func TestDatasetRef_UnmarshalJSON_IdOnly(t *testing.T) {
	jsonData := `{"id":"41067890"}`

	var ref DatasetRef
	if err := json.Unmarshal([]byte(jsonData), &ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ref.Id != "41067890" {
		t.Errorf("expected Id 41067890, got %q", ref.Id)
	}
	if ref.Record != nil {
		t.Errorf("expected Record to be nil for id-only response, got %+v", ref.Record)
	}
}
```

- [ ] **Step 3: Add the both-present test (nested wins)**

Append:

```go
func TestDatasetRef_UnmarshalJSON_BothPresent(t *testing.T) {
	// During a transitional rollout the server could emit both the top-level
	// label and the nested record. The nested form must win.
	jsonData := `{
		"id": "41067890",
		"label": "old flat label",
		"record": {"label": "new nested label"}
	}`

	var ref DatasetRef
	if err := json.Unmarshal([]byte(jsonData), &ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ref.Record == nil {
		t.Fatal("expected Record to be populated from nested shape")
	}
	if ref.Record.Label != "new nested label" {
		t.Errorf("expected nested label to win, got %q", ref.Record.Label)
	}
}
```

- [ ] **Step 4: Add the empty-label test**

Append:

```go
func TestDatasetRef_UnmarshalJSON_EmptyLabel(t *testing.T) {
	// An explicit empty flat label is treated as id-only (no synthesised
	// blank brief). NOTE: this is deliberate and goes away with the spec §7
	// cleanup PR — future refactors should not "fix" it to synthesise an
	// empty brief.
	jsonData := `{"id":"41067890","label":""}`

	var ref DatasetRef
	if err := json.Unmarshal([]byte(jsonData), &ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ref.Record != nil {
		t.Errorf("expected Record to be nil for empty flat label, got %+v", ref.Record)
	}
}
```

- [ ] **Step 5: Add the malformed-input test**

Append:

```go
func TestDatasetRef_UnmarshalJSON_MalformedJSON(t *testing.T) {
	var ref DatasetRef
	err := json.Unmarshal([]byte(`{`), &ref)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
```

- [ ] **Step 6: Add the type-mismatch test**

Append. This pins the contract that the normalization switch only runs on well-typed input — a future refactor that replaces the `raw` struct with manual token-level decoding would silently regress without this — and that on error the receiver is left zero (the decode bails out before any assignment to `d.*`).

```go
func TestDatasetRef_UnmarshalJSON_RecordWrongType(t *testing.T) {
	// record arriving as a JSON string (rather than an object) must surface
	// an error, not silently fall through to the flat or id-only branch.
	jsonData := `{"id":"41067890","record":"not an object"}`

	var ref DatasetRef
	err := json.Unmarshal([]byte(jsonData), &ref)
	if err == nil {
		t.Fatal("expected error for record with non-object type, got nil")
	}
	// Invariant: on decode error the receiver is left zero. If a future
	// refactor starts assigning d.Id or d.Record before the shape check,
	// this will catch it.
	if ref.Id != "" {
		t.Errorf("expected ref.Id to be empty on decode error, got %q", ref.Id)
	}
	if ref.Record != nil {
		t.Errorf("expected ref.Record to be nil on decode error, got %+v", ref.Record)
	}
}
```

- [ ] **Step 7: Add the adversarial-inputs table test**

Append. Covers explicit JSON nulls, non-object `record`, and duplicate-key last-wins semantics — real-world adversarial shapes that upstream APIs or fuzzers can emit. All four cases are cheap and guard against silent regressions in the decoder's error-vs-tolerate split.

```go
func TestDatasetRef_UnmarshalJSON_AdversarialInputs(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, ref DatasetRef)
	}{
		{
			name:  "RecordExplicitNull",
			input: `{"id":"x","record":null}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "x" {
					t.Errorf("expected Id %q, got %q", "x", ref.Id)
				}
				if ref.Record != nil {
					t.Errorf("expected Record nil for explicit null, got %+v", ref.Record)
				}
			},
		},
		{
			name:  "LabelExplicitNull",
			input: `{"id":"x","label":null}`,
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "x" {
					t.Errorf("expected Id %q, got %q", "x", ref.Id)
				}
				if ref.Record != nil {
					t.Errorf("expected Record nil for null label, got %+v", ref.Record)
				}
			},
		},
		{
			name:    "RecordArrayType",
			input:   `{"id":"x","record":[]}`,
			wantErr: true,
		},
		{
			name:  "DuplicateIdKeys",
			input: `{"id":"a","id":"b"}`,
			// encoding/json takes last-wins on duplicate keys; pin that so a
			// future refactor that switches to a strict streaming decoder
			// surfaces the change instead of silently flipping semantics.
			check: func(t *testing.T, ref DatasetRef) {
				if ref.Id != "b" {
					t.Errorf("expected last-wins Id %q, got %q", "b", ref.Id)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ref DatasetRef
			err := json.Unmarshal([]byte(tc.input), &ref)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, ref)
			}
		})
	}
}
```

- [ ] **Step 8: Run the new tests**

Run: `go test ./client/rest/... -count=1 -run 'TestDatasetRef_UnmarshalJSON_'`
Expected: 8 top-level tests (Task 1's nested-shape test plus the seven added in this task, one of which includes four subtests).

- [ ] **Step 9: Run the full `client/rest` suite**

Run: `go test ./client/rest/... -count=1`
Expected: all tests pass.

- [ ] **Step 10: Commit**

```bash
git add client/rest/inbound_share_types_test.go
git commit -m "$(cat <<'EOF'
test(rest): cover DatasetRef.UnmarshalJSON branches

Add regression coverage for the flat, id-only, both-present,
empty-label, malformed-JSON, and record-wrong-type branches of the
custom decoder introduced in the previous commit. An adversarial
inputs table test pins behaviour for explicit nulls, non-object
record types, and duplicate-key last-wins semantics.
EOF
)"
```

---

## Task 4: Add end-to-end consumer fixtures for the nested shape and for id-only responses

**Purpose:** Prove the consumer (`GetInboundShareTable`) and the `TrackTableResponse` marshaller work against the new wire shape and that id-only server responses are rejected with a clear error. These tests exercise the code paths future customers will hit after the server rolls over. The new tests share an httptest-server helper to keep fixture-boilerplate drift under control.

**Files:**
- Modify: `client/rest/inbound_share_types_test.go` — add `TestTrackTableResponseMarshaling_NestedShape`.
- Modify: `client/rest/inbound_share_tables_test.go` — add a private `newGetInboundShareTableServer` helper plus `TestGetInboundShareTable_NestedShape` and `TestGetInboundShareTable_IdOnlyResponse`.

- [ ] **Step 1: Add `TestTrackTableResponseMarshaling_NestedShape`**

Append to `client/rest/inbound_share_types_test.go`:

```go
func TestTrackTableResponseMarshaling_NestedShape(t *testing.T) {
	// Sibling of TestTrackTableResponseMarshaling (which exercises the flat
	// wire shape). This fixture uses the post-migration nested form.
	responseJSON := `{
		"table": {
			"id": "41056789",
			"share": {
				"id": "41012345",
				"shareName": "ACME_CUSTOMER_DATA"
			},
			"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
			"tableName": "CUSTOMER_EVENTS",
			"schemaName": "PUBLIC",
			"tableType": "TABLE",
			"status": "Active",
			"sourceDataset": {
				"id": "41067890",
				"record": {
					"label": "Customer Events",
					"description": "Snowflake share for customer events",
					"icon": "icons/event"
				}
			},
			"createdBy": {"id": "123"},
			"createdAt": "2026-03-26T10:00:00Z",
			"updatedBy": {"id": "123"},
			"updatedAt": "2026-03-26T10:00:00Z"
		},
		"dataset": {
			"id": "41067890",
			"label": "Customer Events",
			"kind": "Event",
			"source": "sharein_table_41056789",
			"createdBy": {"id": "123"},
			"createdAt": "2026-03-26T10:00:00Z",
			"updatedBy": {"id": "123"},
			"updatedAt": "2026-03-26T10:00:00Z"
		}
	}`

	var response TrackTableResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Table.Id != "41056789" {
		t.Errorf("Expected Table.Id 41056789, got %s", response.Table.Id)
	}
	if response.Table.SourceDataset == nil {
		t.Fatal("Expected Table.SourceDataset to be non-nil")
	}
	if response.Table.SourceDataset.Record == nil {
		t.Fatal("Expected Table.SourceDataset.Record to be populated by nested shape")
	}
	if response.Table.SourceDataset.Record.Label != "Customer Events" {
		t.Errorf("Expected nested label 'Customer Events', got %s", response.Table.SourceDataset.Record.Label)
	}
	if response.Table.SourceDataset.Record.Description != "Snowflake share for customer events" {
		t.Errorf("Expected nested description, got %q", response.Table.SourceDataset.Record.Description)
	}
	if response.Table.SourceDataset.Record.Icon != "icons/event" {
		t.Errorf("Expected nested icon 'icons/event', got %q", response.Table.SourceDataset.Record.Icon)
	}
	if response.Dataset.Id != "41067890" {
		t.Errorf("Expected Dataset.Id 41067890, got %s", response.Dataset.Id)
	}
}
```

- [ ] **Step 2: Add the shared httptest helper**

Append to `client/rest/inbound_share_tables_test.go`. The helper captures the path/method assertions and response-body writing that `TestGetInboundShareTable_NestedShape` and `TestGetInboundShareTable_IdOnlyResponse` both need, so the two new tests only differ in their JSON payloads.

```go
// newGetInboundShareTableServer returns an httptest server that asserts the request
// matches the shareId/tableId GET contract and replies 200 OK with
// responseJSON. Helper shared by the nested-shape and id-only read-path
// tests. Always responds 200 — not suitable for testing non-success status
// codes; extend when that need arises.
func newGetInboundShareTableServer(t *testing.T, shareId, tableId, responseJSON string) *httptest.Server {
	t.Helper()
	wantPath := "/v1/shares/inbound/" + shareId + "/tables/" + tableId
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("Expected path %s, got %s", wantPath, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
}
```

- [ ] **Step 3: Add `TestGetInboundShareTable_NestedShape`**

Append to `client/rest/inbound_share_tables_test.go`. Uses the helper from Step 2; the only thing this test contributes beyond the existing `TestGetInboundShareTable` (which uses the flat wire shape) is the nested JSON fixture.

```go
func TestGetInboundShareTable_NestedShape(t *testing.T) {
	// Sibling of TestGetInboundShareTable (flat wire shape). Same endpoint,
	// same client call, nested JSON over the wire — result must be identical.
	server := newGetInboundShareTableServer(t, "41012345", "41056789", `{
		"id": "41056789",
		"share": {
			"id": "41012345",
			"shareName": "ACME_CUSTOMER_DATA"
		},
		"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
		"tableName": "CUSTOMER_EVENTS",
		"schemaName": "PUBLIC",
		"tableType": "TABLE",
		"status": "Active",
		"sourceDataset": {
			"id": "41067890",
			"record": {
				"label": "Customer Events",
				"description": "Snowflake share for customer events",
				"icon": "icons/event"
			}
		},
		"createdBy": {"id": "123"},
		"createdAt": "2026-03-26T10:00:00Z",
		"updatedBy": {"id": "123"},
		"updatedAt": "2026-03-26T10:00:00Z"
	}`)
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	result, err := client.GetInboundShareTable(ctx, "41012345", "41056789")
	if err != nil {
		t.Fatalf("GetInboundShareTable failed: %v", err)
	}
	if result.Table.Id != "41056789" {
		t.Errorf("Expected table ID 41056789, got %s", result.Table.Id)
	}
	if result.Dataset.Id != "41067890" {
		t.Errorf("Expected dataset ID 41067890, got %s", result.Dataset.Id)
	}
	if result.Dataset.Label != "Customer Events" {
		t.Errorf("Expected dataset label 'Customer Events', got %s", result.Dataset.Label)
	}
}
```

- [ ] **Step 4: Add `TestGetInboundShareTable_IdOnlyResponse`**

Append to `client/rest/inbound_share_tables_test.go`. An id-only response (no flat label, no nested record) trips the `Record == nil` branch in `GetInboundShareTable`, so the expected error substring is `sourceDataset.record`.

```go
func TestGetInboundShareTable_IdOnlyResponse(t *testing.T) {
	// If the sharein endpoint were ever to return an id-only sourceDataset
	// (no flat label, no nested record), GetInboundShareTable must refuse
	// the response with a clear error. This documents the read-path
	// contract: id-only is not sufficient for a Terraform read.
	server := newGetInboundShareTableServer(t, "41012345", "41056789", `{
		"id": "41056789",
		"share": {"id": "41012345"},
		"fullTablePath": "PUBLIC/CUSTOMER_EVENTS",
		"tableName": "CUSTOMER_EVENTS",
		"schemaName": "PUBLIC",
		"tableType": "TABLE",
		"status": "Active",
		"sourceDataset": {"id": "41067890"},
		"createdBy": {"id": "123"},
		"createdAt": "2026-03-26T10:00:00Z",
		"updatedBy": {"id": "123"},
		"updatedAt": "2026-03-26T10:00:00Z"
	}`)
	defer server.Close()

	client := New(server.URL, server.Client())
	ctx := context.Background()

	_, err := client.GetInboundShareTable(ctx, "41012345", "41056789")
	if err == nil {
		t.Fatal("expected error for id-only sourceDataset response, got nil")
	}
	// Assert against the full Record-nil prefix (not just "sourceDataset.record")
	// so a future reorder that trips the Record.Label branch on a pathological
	// payload would fail this test instead of silently passing the wrong
	// branch.
	if !strings.Contains(err.Error(), "missing sourceDataset.record for table") {
		t.Errorf("expected error to mention 'missing sourceDataset.record for table', got %v", err)
	}
}
```

Because this test calls `strings.Contains`, `inbound_share_tables_test.go` needs `"strings"` in its import block. Update the imports at the top of the file:

```go
import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)
```

- [ ] **Step 5: Run the new tests**

Run: `go test ./client/rest/... -count=1 -run 'TestTrackTableResponseMarshaling_NestedShape|TestGetInboundShareTable_NestedShape|TestGetInboundShareTable_IdOnlyResponse'`
Expected: 3 passing tests.

- [ ] **Step 6: Run the full `client/rest` suite**

Run: `go test ./client/rest/... -count=1`
Expected: all tests pass (no regressions).

- [ ] **Step 7: Commit**

```bash
git add client/rest/inbound_share_types_test.go client/rest/inbound_share_tables_test.go
git commit -m "$(cat <<'EOF'
test(rest): exercise sharein endpoints with nested DatasetRef fixtures

Add end-to-end fixture tests that route the new nested wire shape
through TrackTableResponse unmarshalling and through the
GetInboundShareTable client call. A shared httptest helper keeps
the GET-path boilerplate in one place. Also add an id-only
response test to lock in the refuse-without-label contract on
the read path.
EOF
)"
```

---

## Task 5: Final verification across the whole module

**Purpose:** Before declaring the plan complete, prove that the whole module still builds, vets, and tests clean — not just the package we touched. Catches any incidental breakage in packages that transitively import `client/rest`.

**Files:** None modified.

- [ ] **Step 1: Clean build**

Run: `go build ./...`
Expected: exits 0 with no output.

- [ ] **Step 2: Vet the whole module**

Run: `go vet ./...`
Expected: exits 0 with no output.

- [ ] **Step 3: Run unit tests across the module**

Run: `go test ./... -count=1 -short`
Expected: `client/rest/...` tests pass. Acceptance tests under `observe/` require a live Observe stack and may fail fast with connection errors — that is acceptable here. What we care about is the `client/rest` suite plus compile-time health elsewhere, which Step 1 already covers authoritatively. There is intentionally no grep for old `.Label` accesses: `go build ./...` at Step 1 is the real check and catches what grep would miss (intermediate-variable access, consumers in other packages, reflective use).

- [ ] **Step 4: Confirm `gofmt` cleanliness**

Run: `gofmt -l client/rest/`
Expected: exits 0 with no output.

- [ ] **Step 5: Review the commit log**

Run: `git log --oneline master..HEAD`
Expected: three commits from this plan (Task 2, Task 3, Task 4) plus the `chore: ignore .worktrees/ directory` bootstrap and the spec commit. Inspect messages to confirm they read sensibly as release history.

- [ ] **Step 6: Tell the reviewer the plan is complete**

No auto-push. The release operator decides when to push to the remote and open the PR. Report to the user:

> Implementation complete on branch `feature/nested-datasetref` in worktree `.worktrees/nested-datasetref`. Three implementation commits plus the spec. `go build`, `go vet`, `go test ./client/rest/...` all clean. Ready for review or push.
