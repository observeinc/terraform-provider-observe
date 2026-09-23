// Package binding provides utilities for generating bindings for cross-tenant exports.
// Essentially we want to replace all raw object ids in the resource definition with references
// instead, created by looking up the objects by name using data sources.
// Overall, the process works like so:
//	1. The Observe backend adds a data source with the `export_object_bindings` flag set.
//  2. When the data source is read, a Generator is created, and then:
//      a. Collect walks all fields that could contain ids and records dataset candidates.
//      b. Resolve looks up the labels of all candidates in one pass. All I/O happens here or in
//         NewGenerator.
//      c. Generate/TryBind replace each known id with a local variable reference, and add a
//         "binding" entry to the Generator's internal state. This binding includes all the
//         information necessary for later generating a data source that fetches the resource
//         by name, and the local variable definition for the above reference.
//  3. Finally, we aggregate all the bindings from above and insert it somewhere into the data
//     source state (typically an internal "_bindings" field).
//  4. The Observe backend then extracts those bindings from the data source state, and uses
//     them to generate data sources + locals in addition to the main resource export
//     (in which the ids have already been replaced with local variable references).

package binding

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

var (
	replaceInvalid  = regexp.MustCompile(`([^0-9a-zA-Z-_]+)`)
	hasLeadingDigit = regexp.MustCompile(`^[0-9]`)
)

type ResourceCacheEntry struct {
	TfName    string
	LookupKey string
}

type ResourceCache struct {
	idToLabel       map[Ref]ResourceCacheEntry
	datasetLabels   map[string]string   // resolved dataset id -> label
	absentDatasets  map[string]struct{} // looked up but not found
	workspaceOid    *oid.OID
	workspaceEntry  *ResourceCacheEntry
	forResourceKind Kind
	forResourceName string
}

// NewResourceCache loads all resources of the given kinds except datasets, which
// Generator.Resolve looks up by id.
func NewResourceCache(ctx context.Context, kinds KindSet, client *observe.Client, forResourceKind Kind, forResourceName string) (ResourceCache, error) {
	var cache = ResourceCache{
		idToLabel:       make(map[Ref]ResourceCacheEntry),
		datasetLabels:   make(map[string]string),
		absentDatasets:  make(map[string]struct{}),
		forResourceKind: forResourceKind,
		forResourceName: sanitizeIdentifier(forResourceName),
	}
	// special case: one workspace per customer, always needed for lookup
	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return cache, err
	}
	cache.addEntry(KindWorkspace, workspaces[0].Label, workspaces[0].Label, workspaces[0].Oid().String(), false, nil, make(map[string]struct{}))
	cache.workspaceOid = workspaces[0].Oid()
	cache.workspaceEntry = cache.LookupId(KindWorkspace, cache.workspaceOid.String())

	for resourceKind := range kinds {
		// colisions are really bad, so make a best effort to prevent them
		existingResourceNames := make(map[string]struct{})
		disambiguator := 1
		switch resourceKind {
		case KindWorksheet:
			worksheets, err := client.ListWorksheetIdLabelOnly(ctx, cache.workspaceOid.Id)
			if err != nil {
				return cache, err
			}
			for _, wk := range worksheets {
				cache.addEntry(KindWorksheet, wk.Label, wk.Label, wk.Id, true, &disambiguator, existingResourceNames)
			}
		case KindUser:
			users, err := client.ListUsers(ctx)
			if err != nil {
				return cache, err
			}
			for _, user := range users {
				// lookupKey is the email because observe_user is looked up by email, not display name.
				// Use FormatInt (not user.Id.String()) — String() JSON-quotes the number ("2347"),
				// but OID parsing captures the bare digits (2347), so the cache keys must match.
				cache.addEntry(KindUser, user.Email, user.Label, strconv.FormatInt(int64(user.Id), 10), true, &disambiguator, existingResourceNames)
			}
		case KindMonitorV2Action:
			actions, err := client.SearchMonitorV2Action(ctx, &cache.workspaceOid.Id, nil)
			if err != nil {
				return cache, err
			}
			for _, action := range actions {
				cache.addEntry(KindMonitorV2Action, action.Name, action.Name, action.Id, true, &disambiguator, existingResourceNames)
			}
		}
	}
	return cache, nil
}

func (c *ResourceCache) addEntry(kind Kind, lookupKey string, displayName string, id string, addPrefix bool, disambiguator *int, existingNames map[string]struct{}) {
	resourceName := sanitizeIdentifier(displayName)
	for base := resourceName; ; *disambiguator++ {
		if _, found := existingNames[resourceName]; !found {
			break
		}
		resourceName = fmt.Sprintf("%s_%d", base, *disambiguator)
	}
	var empty struct{}
	existingNames[resourceName] = empty

	var tfName string
	if addPrefix {
		tfName = fmt.Sprintf("%s_%s__%s_%s", c.forResourceKind, c.forResourceName, kind, resourceName)
	} else {
		tfName = fmt.Sprintf("%s_%s", kind, resourceName)
	}
	c.idToLabel[Ref{Kind: kind, Key: id}] = ResourceCacheEntry{
		TfName:    tfName,
		LookupKey: lookupKey,
	}
}

func (c *ResourceCache) LookupId(kind Kind, id string) *ResourceCacheEntry {
	maybeEnt, ok := c.idToLabel[Ref{Kind: kind, Key: id}]
	if !ok {
		return nil
	}
	return &maybeEnt
}

// DatasetLabeler resolves dataset ids to labels, omitting absent or hidden ids.
type DatasetLabeler interface {
	LookupDatasetLabels(ctx context.Context, ids []string) (map[string]string, error)
}

type Generator struct {
	resourceType    Kind
	resourceName    string
	enabledBindings KindSet
	bindings        Mapping
	cache           ResourceCache
	labeler         DatasetLabeler
	candidates      map[string]struct{} // collected dataset ids awaiting Resolve
	err             error               // first binding error, reported by GetBindings
}

// NewGenerator creates a new binding generator for the given resource type and name.
// Callers Collect all ids, call Resolve once, then Generate/TryBind; none of the
// latter perform I/O.
func NewGenerator(ctx context.Context, resourceType Kind, resourceName string,
	client *observe.Client, enabledBindings KindSet) (Generator, error) {
	rc, err := NewResourceCache(ctx, enabledBindings, client, resourceType, resourceName)
	if err != nil {
		return Generator{}, err
	}
	bindings := NewMapping()
	return Generator{
		resourceType:    resourceType,
		resourceName:    resourceName,
		enabledBindings: enabledBindings,
		bindings:        bindings,
		cache:           rc,
		labeler:         client,
		candidates:      make(map[string]struct{}),
	}, nil
}

// Collect records the dataset ids that Generate would try to bind in data.
func (g *Generator) Collect(data interface{}) {
	g.walk(data, false)
}

// CollectId records id as a candidate for a later TryBindId.
func (g *Generator) CollectId(kind Kind, id string) {
	if _, enabled := g.enabledBindings[kind]; !enabled || kind != KindDataset || !isDatasetId(id) {
		return
	}
	g.candidates[id] = struct{}{}
}

// CollectOid records oidObj as a candidate for a later TryBindOid.
func (g *Generator) CollectOid(oidObj oid.OID) {
	if kind, ok := resolveOidToKind(oidObj); ok {
		g.CollectId(kind, oidObj.Id)
	}
}

// Resolve looks up all collected dataset ids not yet resolved, in id order, and
// names them. Fails on lookup errors and on distinct datasets sharing a label.
func (g *Generator) Resolve(ctx context.Context) error {
	var ids []string
	for id := range g.candidates {
		_, known := g.cache.datasetLabels[id]
		_, absent := g.cache.absentDatasets[id]
		if !known && !absent {
			ids = append(ids, id)
		}
	}
	g.candidates = make(map[string]struct{})
	if len(ids) == 0 {
		return nil
	}
	labels, err := g.labeler.LookupDatasetLabels(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to look up datasets: %w", err)
	}
	for _, id := range ids {
		if label, ok := labels[id]; ok {
			g.cache.datasetLabels[id] = label
		} else {
			g.cache.absentDatasets[id] = struct{}{}
		}
	}
	return g.nameDatasets()
}

// nameDatasets rebuilds dataset cache entries in numeric id order, so names do
// not depend on collection or response order. Ids are canonical positive
// decimals (isDatasetId), so ordering by length then text is numeric order.
func (g *Generator) nameDatasets() error {
	ids := make([]string, 0, len(g.cache.datasetLabels))
	for id := range g.cache.datasetLabels {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if len(ids[i]) != len(ids[j]) {
			return len(ids[i]) < len(ids[j])
		}
		return ids[i] < ids[j]
	})
	idByLabel := make(map[string]string, len(ids))
	for _, id := range ids {
		label := g.cache.datasetLabels[id]
		if other, dup := idByLabel[label]; dup {
			return fmt.Errorf("datasets %s and %s share the label %q and cannot be told apart by an export binding", other, id, label)
		}
		idByLabel[label] = id
	}
	disambiguator := 1
	existingNames := make(map[string]struct{})
	for _, id := range ids {
		label := g.cache.datasetLabels[id]
		g.cache.addEntry(KindDataset, label, label, id, true, &disambiguator, existingNames)
	}
	return nil
}

// isDatasetId reports whether id is a canonical positive decimal id.
func isDatasetId(id string) bool {
	n, err := strconv.ParseInt(id, 10, 64)
	return err == nil && n > 0 && strconv.FormatInt(n, 10) == id
}

func (g *Generator) fail(err error) {
	if g.err == nil {
		g.err = err
	}
}

// lookup by kind and id, if valid then return a local variable reference,
// otherwise return the id (no-op)
func (g *Generator) TryBindId(kind Kind, id string) (maybeRef string, didBind bool) {
	return g.tryBind(kind, id, false)
}

// infer the kind from the oid and lookup, if valid then return a local variable reference,
// otherwise return the oid as a string (no-op)
func (g *Generator) TryBindOid(oidObj oid.OID) (maybeRef string, didBind bool) {
	kind, ok := resolveOidToKind(oidObj)
	if !ok {
		return oidObj.String(), false
	}
	maybeRef, didBind = g.tryBind(kind, oidObj.Id, true)
	if !didBind {
		return oidObj.String(), false
	}
	return maybeRef, true
}

func (g *Generator) tryBind(kind Kind, id string, isOid bool) (maybeRef string, didBind bool) {

	var e *ResourceCacheEntry
	if kind == KindWorkspace && id == g.cache.workspaceOid.Id {
		// workspaces are special since there should only be one primary one
		e = g.cache.workspaceEntry
	} else {
		// if not enabled, skip
		if _, found := g.enabledBindings[kind]; !found {
			return id, false
		}
		// lookup
		e = g.cache.LookupId(kind, id)
		if e == nil {
			if kind == KindDataset && isDatasetId(id) {
				if _, absent := g.cache.absentDatasets[id]; !absent {
					g.fail(fmt.Errorf("internal error: dataset %s was not collected before Resolve", id))
				}
			}
			return id, false
		}
	}
	// process into local var ref
	insertPrefix := kind == KindWorkspace
	terraformLocal := g.fmtTfLocalVar(kind, e, insertPrefix)
	ref := Ref{Kind: kind, Key: e.LookupKey}
	if prev, ok := g.bindings[ref]; ok && kind == KindDataset && prev.IsOid != isOid {
		// a binding holds one IsOid, so one local cannot serve both forms
		g.fail(fmt.Errorf("dataset %q is referenced both by id and by oid, which export bindings do not support", e.LookupKey))
	}
	g.bindings[ref] = Target{
		TfName:            e.TfName,
		TfLocalBindingVar: terraformLocal,
		IsOid:             isOid,
	}
	return g.fmtTfLocalVarRef(terraformLocal), true
}

// Generate walks the provided data structure and for all ids encountered,
// generates a binding for it, and replaces the id with a local variable reference
func (g *Generator) Generate(data interface{}) {
	g.walk(data, true)
}

// walk visits every id candidate in data, binding it if bind is set and
// collecting it otherwise.
func (g *Generator) walk(data interface{}, bind bool) {
	mapOverJsonStringKeys(data, func(key string, value string) string {
		if valueOid, err := oid.NewOID(value); err == nil {
			if !bind {
				g.CollectOid(*valueOid)
				return value
			}
			ref, _ := g.TryBindOid(*valueOid)
			return ref
		}
		for _, kind := range guessKindFromKey(key) {
			if !bind {
				g.CollectId(kind, value)
			} else if maybeRef, didBind := g.TryBindId(kind, value); didBind {
				return maybeRef
			}
		}
		return value
	})
}

// GenerateJson does the same as Generate, but accepts a raw json string. It
// returns the Generator's first error, if any.
func (g *Generator) GenerateJson(jsonStr []byte) ([]byte, error) {
	return transformJson(jsonStr, func(dataPtr *interface{}) error {
		g.Generate(*dataPtr)
		return g.err
	})
}

// GetBindings returns the bindings generated so far, or the first error from
// generating them.
func (g *Generator) GetBindings() (BindingsObject, error) {
	if g.err != nil {
		return BindingsObject{}, g.err
	}
	enabledList := make([]Kind, 0)
	for binding := range g.enabledBindings {
		enabledList = append(enabledList, binding)
	}
	// sort for stability of comparison later on
	sort.Slice(enabledList, func(i int, j int) bool {
		return string(enabledList[i]) < string(enabledList[j])
	})

	workspaceTarget := g.cache.workspaceEntry
	if workspaceTarget == nil {
		return BindingsObject{}, fmt.Errorf("Internal error: workspace was not resolved correctly.")
	}

	return BindingsObject{
		Mappings: g.bindings,
		Kinds:    enabledList,
		Workspace: Target{
			TfLocalBindingVar: g.fmtTfLocalVar(KindWorkspace, workspaceTarget, true),
			TfName:            workspaceTarget.TfName,
			IsOid:             true,
		},
		WorkspaceName: g.cache.workspaceEntry.LookupKey,
	}, nil
}

func (g *Generator) GetBindingsJson() ([]byte, error) {
	bindings, err := g.GetBindings()
	if err != nil {
		return nil, err
	}
	return json.Marshal(bindings)
}

// InsertBindingsObject inserts the bindings object into the provided map
func (g *Generator) InsertBindingsObject(data map[string]interface{}) error {
	bindingsObject, err := g.GetBindings()
	if err != nil {
		return err
	}
	data[bindingsKey] = bindingsObject
	return nil
}

// InsertBindingsObjectJson inserts the bindings object into the provided json data, root must be a map
func (g *Generator) InsertBindingsObjectJson(jsonData []byte) ([]byte, error) {
	return transformJson(jsonData, func(dataPtr *interface{}) error {
		return g.InsertBindingsObject((*dataPtr).(map[string]interface{}))
	})
}

func (g *Generator) fmtTfLocalVar(kind Kind, e *ResourceCacheEntry, insertPrefix bool) string {
	if insertPrefix {
		return sanitizeIdentifier(fmt.Sprintf("binding__%s_%s__%s", g.resourceType, g.resourceName, e.TfName))
	}
	return sanitizeIdentifier(fmt.Sprintf("binding__%s", e.TfName))
}

func (g *Generator) fmtTfLocalVarRef(tfLocalVar string) string {
	return fmt.Sprintf("${local.%s}", tfLocalVar)
}

func resolveOidToKind(oidObj oid.OID) (Kind, bool) {
	switch oidObj.Type {
	case oid.TypeDataset:
		return KindDataset, true
	case oid.TypeWorksheet:
		return KindWorksheet, true
	case oid.TypeWorkspace:
		return KindWorkspace, true
	case oid.TypeUser:
		return KindUser, true
	case oid.TypeMonitorV2Action:
		return KindMonitorV2Action, true
	default:
		return "", false
	}
}

func guessKindFromKey(key string) []Kind {
	switch key {
	case "id":
		return []Kind{KindDataset, KindWorksheet}
	case "datasetId":
		fallthrough
	case "keyForDatasetId":
		fallthrough
	case "sourceDatasetId":
		fallthrough
	case "targetDataset":
		fallthrough
	case "dataset":
		return []Kind{KindDataset}
	case "workspaceId":
		return []Kind{KindWorkspace}
	case "userId":
		return []Kind{KindUser}
	default:
		return []Kind{}
	}
}

func mapOverJsonStringKeys(data interface{}, f func(key string, value string) string) {
	var stack []interface{}
	stack = append(stack, data)
	for len(stack) > 0 {
		var cur interface{}
		cur, stack = stack[len(stack)-1], stack[:len(stack)-1]
		switch jsonNode := cur.(type) {
		case map[string]interface{}:
			for k, v := range jsonNode {
				switch kvValue := v.(type) {
				case string:
					jsonNode[k] = f(k, kvValue)
				// if value looks like a composite type, push onto stack for further
				// processing
				case map[string]interface{}:
					stack = append(stack, kvValue)
				case []interface{}:
					stack = append(stack, kvValue)
				}
			}
		case []interface{}:
			for i, v := range jsonNode {
				switch val := v.(type) {
				case string:
					jsonNode[i] = f("", val)
				case map[string]interface{}:
					stack = append(stack, val)
				case []interface{}:
					stack = append(stack, val)
				}
			}
		}
	}
}

func transformJson(data []byte, f func(data *interface{}) error) ([]byte, error) {
	var deserialized interface{}
	err := json.Unmarshal(data, &deserialized)
	if err != nil {
		return nil, fmt.Errorf("Failed to deserialize json: %w", err)
	}
	err = f(&deserialized)
	if err != nil {
		return nil, fmt.Errorf("Failed to transform json data: %w", err)

	}
	serialized, err := json.Marshal(deserialized)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize json data: %w", err)
	}
	return serialized, nil
}

func sanitizeIdentifier(name string) string {
	sanitized := strings.ToLower(replaceInvalid.ReplaceAllString(name, "_"))

	if hasLeadingDigit.MatchString(sanitized) {
		sanitized = "_" + sanitized
	}

	return sanitized
}
