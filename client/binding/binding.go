package binding

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

var (
	replaceInvalid  = regexp.MustCompile(`([^0-9a-zA-Z-_]+)`)
	hasLeadingDigit = regexp.MustCompile(`^[0-9]`)
)

type ResourceCacheEntry struct {
	TfName string
	Label  string
}

type ResourceCache struct {
	idToLabel       map[Ref]ResourceCacheEntry
	workspaceOid    *oid.OID
	workspaceEntry  *ResourceCacheEntry
	forResourceKind string
	forResourceName string
}

// NewResourceCache loads all resources of the given kinds and creates a cache of id -> label mappings
func NewResourceCache(ctx context.Context, kinds KindSet, client *observe.Client, forResourceKind string, forResourceName string) (ResourceCache, error) {
	var cache = ResourceCache{
		idToLabel:       make(map[Ref]ResourceCacheEntry),
		forResourceKind: forResourceKind,
		forResourceName: sanitizeIdentifier(forResourceName),
	}
	// special case: one workspace per customer, always needed for lookup
	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return cache, err
	}
	cache.addEntry(KindWorkspace, workspaces[0].Label, workspaces[0].Oid().String(), false, nil, make(map[string]struct{}))
	cache.workspaceOid = workspaces[0].Oid()
	cache.workspaceEntry = cache.LookupId(KindWorkspace, cache.workspaceOid.String())

	for resourceKind := range kinds {
		// colisions are really bad, so make a best effort to prevent them
		existingResourceNames := make(map[string]struct{})
		disambiguator := 1
		switch resourceKind {
		case KindDataset:
			datasets, err := client.ListDatasetsIdNameOnly(ctx)
			if err != nil {
				return cache, err
			}
			for _, ds := range datasets {
				cache.addEntry(KindDataset, ds.Name, ds.Id, true, &disambiguator, existingResourceNames)
			}
		case KindWorksheet:
			worksheets, err := client.ListWorksheetIdLabelOnly(ctx, cache.workspaceOid.Id)
			if err != nil {
				return cache, err
			}
			for _, wk := range worksheets {
				cache.addEntry(KindWorksheet, wk.Label, wk.Id, true, &disambiguator, existingResourceNames)
			}
		case KindUser:
			users, err := client.ListUsers(ctx)
			if err != nil {
				return cache, err
			}
			for _, user := range users {
				cache.addEntry(KindUser, user.Label, user.Id.String(), true, &disambiguator, existingResourceNames)
			}
		}
	}
	return cache, nil
}

func (c *ResourceCache) addEntry(kind Kind, label string, id string, addPrefix bool, disambiguator *int, existingNames map[string]struct{}) {
	resourceName := sanitizeIdentifier(label)
	if _, found := existingNames[resourceName]; found {
		resourceName = fmt.Sprintf("%s_%d", resourceName, *disambiguator)
		*disambiguator++
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
		TfName: tfName,
		Label:  label,
	}
}

func (c *ResourceCache) LookupId(kind Kind, id string) *ResourceCacheEntry {
	maybeEnt, ok := c.idToLabel[Ref{Kind: kind, Key: id}]
	if !ok {
		return nil
	}
	return &maybeEnt
}

type Generator struct {
	resourceType    string
	resourceName    string
	enabledBindings KindSet
	bindings        Mapping
	cache           ResourceCache
}

// NewGenerator creates a new binding generator for the given resource type and name,
// which keeps track of all the bindings generated for raw ids found through later
// calls to Generate and TryBind.
func NewGenerator(ctx context.Context, resourceType string, resourceName string,
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
	}, nil
}

// lookup by kind and id, if valid then return a local variable reference,
// otherwise return the id (no-op)
func (g *Generator) TryBind(kind Kind, id string) string {
	var e *ResourceCacheEntry
	if kind == KindWorkspace && id == g.cache.workspaceOid.Id {
		// workspaces are special since there should only be one primary one
		e = g.cache.workspaceEntry
	} else {
		// lookup
		e = g.cache.LookupId(kind, id)
		if e == nil {
			return id
		}
	}
	// process into local var ref
	insertPrefix := kind == KindWorkspace
	terraformLocal := g.fmtTfLocalVar(kind, e, insertPrefix)
	g.bindings[Ref{Kind: kind, Key: e.Label}] = Target{
		TfName:            e.TfName,
		TfLocalBindingVar: terraformLocal,
	}
	return g.fmtTfLocalVarRef(terraformLocal)
}

// Generate walks the provided data structure and for all ids encountered,
// generates a binding for it, and replaces the id with a local variable reference
func (g *Generator) Generate(data interface{}) {
	mapOverJsonStringKeys(data, func(key string, value string, jsonMapNode map[string]interface{}) {
		var id string
		var kinds []Kind
		if valueOid, err := oid.NewOID(value); err == nil {
			id = valueOid.Id
			kinds = resolveOidToKinds(valueOid)
		} else {
			id = value
			kinds = resolveKeyToKinds(key)
		}

		for _, kind := range kinds {
			// if not enabled, skip
			if _, found := g.enabledBindings[kind]; !found {
				continue
			}
			// try bind the name
			maybeRef := g.TryBind(kind, id)
			// if lookup succeeded, then the returned value should be a lv ref and not the
			// input id
			if maybeRef != value {
				jsonMapNode[key] = maybeRef
				break
			}
		}
	})
}

// GenerateJson does the same as Generate, but accepts a raw json string
func (g *Generator) GenerateJson(jsonStr []byte) ([]byte, error) {
	return transformJson(jsonStr, func(dataPtr *interface{}) error {
		g.Generate(*dataPtr)
		return nil
	})
}

// GetBindings returns the bindings generated so far
func (g *Generator) GetBindings() (BindingsObject, error) {
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
		},
		WorkspaceName: g.cache.workspaceEntry.Label,
	}, nil
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

func resolveOidToKinds(oidObj *oid.OID) []Kind {
	kind := Kind(oidObj.Type)
	if kind.Valid() {
		return []Kind{kind}
	}
	return []Kind{}
}

func resolveKeyToKinds(key string) []Kind {
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

func mapOverJsonStringKeys(data interface{}, f func(key string, value string, jsonMapNode map[string]interface{})) {
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
					f(k, kvValue, jsonNode)
				// if value looks like a composite type, push onto stack for further
				// processing
				case map[string]interface{}:
					stack = append(stack, kvValue)
				case []interface{}:
					stack = append(stack, kvValue)
				}
			}
		case []interface{}:
			for _, object := range jsonNode {
				stack = append(stack, object)
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
