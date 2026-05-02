# DatasetRef REST shape migration — Terraform provider design

**Date:** 2026-05-02
**Status:** Draft — awaiting review
**Scope:** `terraform-provider-observe`
**Companion upstream plan:** `nested_datasetref_0fc6e3a2.plan.md` (Observe backend repo)

## 1. Context

The Observe backend is reshaping its REST `DatasetRef` type from a flat `{id, label?}` to a nested `{id, record?: DatasetBrief}`, where `DatasetBrief` carries `label`, `description`, and `icon`. The new shape mirrors the merged `ObjectRef` / `ObjectBrief` template from Gerrit 87946. Only `record` changes across the migration; `id` stays required.

In `terraform-provider-observe`, the REST `DatasetRef` has exactly one consumer: `InboundShareTable.SourceDataset` (`client/rest/inbound_share_types.go:84`), populated by the sharein endpoint. The read consumer (`client/rest/inbound_share_tables.go:59-74`) extracts `.Id` and `.Label` to populate `InboundShareDataset` in `TrackTableResponse`, which is then used to populate `dataset_label` in Terraform state for the `observe_inbound_share_table` resource.

The provider already makes a second GraphQL `GetDataset` call on Create and Read and overwrites `dataset_label` via `enrichResultFromDataset` (`observe/resource_inbound_share_table.go:219-223`), so the REST label is effectively a pre-fill before the authoritative GraphQL value arrives. However, the current REST reader hard-fails if `SourceDataset.Label == ""`, and that is what actually breaks at rollover time.

One customer uses this resource. The rollout model is: release the provider first, ping the responsible customer contact to upgrade, then let the API rollover proceed in their tenant. To make that sequencing safe, the new provider release must accept BOTH the legacy flat shape and the new nested shape.

## 2. Scope and constraints

### In scope
- `client/rest/inbound_share_types.go` — reshape `DatasetRef` to `{Id, Record *DatasetBrief}`, add `DatasetBrief{Label, Description, Icon}`, add a custom `UnmarshalJSON` on `DatasetRef` that transparently accepts both wire shapes and normalizes to the nested representation in memory.
- `client/rest/inbound_share_tables.go` — `GetInboundShareTable` reads the label from `SourceDataset.Record.Label` (post-normalization).
- Unit tests in `client/rest/` covering both wire shapes, id-only responses, malformed input, and the updated `GetInboundShareTable` consumer.
- CHANGELOG entry and the outgoing customer-ping template.

### Explicitly out of scope
- The Terraform attribute schema for `observe_inbound_share_table`. `dataset_label` remains `Required`; no attributes are renamed, added, or removed; no state migration is required.
- `observe/resource_inbound_share_table.go`. The GraphQL fallback via `enrichResultFromDataset` is unchanged.
- Removal of flat-shape compatibility. Deferred to a follow-up PR (§7).

### Success criteria
| Provider | API server | Outcome |
|---|---|---|
| old | old | works (baseline, no change) |
| new | old | works — decoder takes the flat branch, label comes from the synthesised `Record` |
| new | new | works — decoder takes the nested branch, label comes from `Record.Label` directly |
| old | new | **broken — the failure the ping is meant to prevent** |

## 3. Type changes — `client/rest/inbound_share_types.go`

### New `DatasetBrief`

```go
// DatasetBrief carries the denormalised metadata that accompanies a dataset
// reference when the server-side `expand` projection is honoured.
type DatasetBrief struct {
    Label       string `json:"label,omitempty"`
    Description string `json:"description,omitempty"`
    Icon        string `json:"icon,omitempty"`
}
```

Only `Label` is consumed today. `Description` and `Icon` are included to match the API contract and to avoid a second struct change when a consumer surfaces for them.

### Reshaped `DatasetRef`

```go
type DatasetRef struct {
    Id     string        `json:"id"`
    Record *DatasetBrief `json:"record,omitempty"`
}
```

The flat `Label` field is removed from the Go type. In-memory, callers only see the nested shape.

### `UnmarshalJSON` — sniff and normalize

```go
// UnmarshalJSON accepts BOTH the legacy flat shape ({"id":..., "label":...})
// and the new nested shape ({"id":..., "record":{"label":...}}). The legacy
// shape is normalised into the nested representation so consumers only deal
// with one in-memory form.
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
        // New nested shape. Use it verbatim. If the server transiently sends
        // both, the nested form wins (forward-compatible).
        d.Record = raw.Record
    case raw.Label != "":
        // Legacy flat shape. Synthesise a brief containing just the label.
        d.Record = &DatasetBrief{Label: raw.Label}
    default:
        // Id-only response (e.g., list/track/update paths post-migration).
        // Record stays nil; the caller decides whether that is acceptable.
        d.Record = nil
    }
    return nil
}
```

**Normalization rules:**
- Nested present → use verbatim.
- Nested absent, flat `label` present → synthesise `Record{Label: label}`.
- Both absent → `Record` stays `nil` (valid for id-only refs).
- Both present → nested wins.

No `MarshalJSON` is defined. The provider never writes `DatasetRef` in a request body.

## 4. Consumer change — `client/rest/inbound_share_tables.go`

`GetInboundShareTable` is the only reader. Diff:

```go
// Before (lines 59-74):
if table.SourceDataset != nil {
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

// After:
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

Both wire shapes converge on `Record.Label` thanks to §3's decoder, so the consumer is shape-agnostic. The error messages are structural — `sourceDataset`, `sourceDataset.id`, `sourceDataset.record`, `sourceDataset.record.label` — so operators can distinguish a missing record from a record that lost its label without needing to know which wire shape arrived on the wire.

## 5. Rollout plan

### Sequencing
1. Release provider `vN+1` (this change) to the Terraform registry. Still compatible with the current flat API. Customer upgrade is a no-op until their API server rolls.
2. Send the ping (template below) to the responsible customer contact.
3. Back-end `DatasetRef` nested-shape rollout proceeds on its own schedule. When it reaches the customer's tenant, the upgraded provider transparently takes the nested branch.
4. Post-rollover, after confirming the customer is on `vN+1` or newer and their last `terraform apply` against the new API succeeded, land the follow-up cleanup PR (§7).

### Version bump
Patch. The customer-visible Terraform schema is unchanged, no state migration, no behavioural change on success paths.

### CHANGELOG entry (drafted)
> - `observe_inbound_share_table`: updated the REST client to accept the upcoming nested `DatasetRef` shape from the Observe sharein API. Existing flat responses continue to work; no Terraform config changes required. Customers using this resource should upgrade to this provider version before the API rollover reaches their tenant.

The repo's `CHANGELOG.md` is frozen (see its header) and release notes live in the GitHub Release attached to each tag — the drafted entry above belongs in the GitHub Release body, not in `CHANGELOG.md`. The follow-up cleanup PR (§7) follows the same convention.

### Customer ping template (drafted)
> Subject: Heads-up — Terraform provider upgrade before Observe API rollover
>
> Hi \<name\>,
>
> We're rolling out a schema change to the Observe sharein API that affects the `observe_inbound_share_table` resource in your Terraform. We've released `terraform-provider-observe` \<vN+1\> which transparently handles both the current and new response shapes.
>
> Please upgrade to \<vN+1\> (or newer) at your convenience. Upgrading before the API rollover reaches your tenant is safest — the new provider works against both, so there's no downside to doing it now. After your tenant's rollover, the older provider version will fail reads on this resource.
>
> No `.tf` changes are required.

### Rollback
Pin to the previous provider version via `version = "…"`. The new provider only adds a decoder branch; no state has been touched, so rollback is clean.

## 6. Testing strategy

### New unit tests — `client/rest/inbound_share_types_test.go`
1. `TestDatasetRef_UnmarshalJSON_FlatShape` — `{"id":"41067890","label":"Customer Events"}` → `Record.Label == "Customer Events"`, `Description == ""`, `Icon == ""`.
2. `TestDatasetRef_UnmarshalJSON_NestedShape` — full nested input → all three brief fields populated.
3. `TestDatasetRef_UnmarshalJSON_IdOnly` — `{"id":"41067890"}` → `Id` set, `Record == nil`, no error.
4. `TestDatasetRef_UnmarshalJSON_BothPresent` — both flat and nested keys → nested wins.
5. `TestDatasetRef_UnmarshalJSON_EmptyLabel` — flat label is empty string → `Record == nil` (no synthesised blank brief).
6. `TestDatasetRef_UnmarshalJSON_MalformedJSON` — bare `{` → error.

### Updates to existing tests
- `TestTrackTableResponseMarshaling` (`inbound_share_types_test.go:104`) keeps its current flat-shape fixture as one case; add a sibling `TestTrackTableResponseMarshaling_NestedShape` using the nested shape.
- The `inbound_share_tables_test.go` test asserting `table.SourceDataset.Label == "Updated Label"` (lines 187-188) becomes `table.SourceDataset.Record.Label`. Add a paired case using the flat wire shape so `GetInboundShareTable` is exercised end-to-end against both shapes.

### New test — `TestGetInboundShareTable_IdOnlyResponse`
Fixture where `sourceDataset` is `{"id":"…"}` with no label or record. Assert the `"missing sourceDataset.record"` error path. Documents the read-path contract: id-only is insufficient for a Terraform read.

### Not doing
- Acceptance tests against a live stack. The sharein backend is mid-migration; unit coverage with fixtures for both wire shapes gives equivalent confidence without the operational burden.
- A new mock server. Existing REST tests use inline `httptest.NewServer` stubs; the new tests follow the same pattern.

### Pre-release manual checklist
- `make test` clean.
- `go vet ./...` and `go build ./...` clean.
- `terraform plan` / `terraform apply` round-trip on a dev stack still on the old API using this branch's provider build — no diff.
- If a dev stack on the new API is available, repeat the round-trip there.

## 7. Follow-up cleanup PR

Prepared and ready to merge once the new server shape has fully rolled out to the affected customer's tenant AND the customer is confirmed on `vN+1` or newer.

### Changes in the cleanup PR
1. **Delete `DatasetRef.UnmarshalJSON` entirely.** After cleanup the nested shape is the only wire form and the default `encoding/json` struct unmarshal is sufficient. Deleting the method is cleaner than trimming branches inside it.

   ```go
   // After cleanup: no custom UnmarshalJSON.
   type DatasetRef struct {
       Id     string        `json:"id"`
       Record *DatasetBrief `json:"record,omitempty"`
   }
   ```

2. **Drop the flat-shape unit tests.** `TestDatasetRef_UnmarshalJSON_FlatShape`, `TestDatasetRef_UnmarshalJSON_BothPresent`, `TestDatasetRef_UnmarshalJSON_EmptyLabel`, and the "flat-shape" half of the `GetInboundShareTable` paired test all go away. Nested-shape and id-only tests remain.

3. **CHANGELOG entry:**

   > - `observe_inbound_share_table`: removed compatibility code for the pre-migration flat `DatasetRef` response shape. No user-visible change; this completes the server-side migration rolled out in vN+1.

### Merge trigger
Operator confirms (a) the customer is on `vN+1` or newer and (b) their tenant's API server has rolled over. Until both are true, the cleanup PR stays as a draft / unmerged branch.

### Why a separate PR instead of a feature flag
The two states (dual-shape vs. nested-only) are disjoint and the transition is short-lived. A flag would add a config knob with no lifetime justification. A draft PR is the right primitive: it carries diff, review, and CI history without being live.
