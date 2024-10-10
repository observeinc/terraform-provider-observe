package binding

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
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
	idToLabel      map[Ref]ResourceCacheEntry
	workspaceOid   *oid.OID
	workspaceEntry *ResourceCacheEntry
}

func NewResourceCache(ctx context.Context, kinds KindSet, client *observe.Client) (ResourceCache, error) {
	var cache = ResourceCache{idToLabel: make(map[Ref]ResourceCacheEntry)}
	// special case: one workspace per customer, always needed for lookup
	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return cache, err
	}
	cache.addEntry(KindWorkspace, workspaces[0].Label, workspaces[0].Oid().String(), nil, make(map[string]struct{}))
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
				cache.addEntry(KindDataset, ds.Name, ds.Id, &disambiguator, existingResourceNames)
			}
		case KindWorksheet:
			worksheets, err := client.ListWorksheetIdLabelOnly(ctx, cache.workspaceOid.Id)
			if err != nil {
				return cache, err
			}
			for _, wk := range worksheets {
				cache.addEntry(KindWorksheet, wk.Label, wk.Id, &disambiguator, existingResourceNames)
			}
		case KindUser:
			users, err := client.ListUsers(ctx)
			if err != nil {
				return cache, err
			}
			for _, user := range users {
				cache.addEntry(KindUser, user.Label, user.Id.String(), &disambiguator, existingResourceNames)
			}
		}
	}
	return cache, nil
}

func (c *ResourceCache) addEntry(kind Kind, label string, id string, disambiguator *int, existingNames map[string]struct{}) {
	resourceName := sanitizeIdentifier(label)
	if _, found := existingNames[resourceName]; found {
		resourceName = fmt.Sprintf("%s_%d", resourceName, *disambiguator)
		*disambiguator++
	}
	var empty struct{}
	existingNames[resourceName] = empty
	c.idToLabel[Ref{kind: kind, key: id}] = ResourceCacheEntry{
		TfName: resourceName,
		Label:  label,
	}
}

func (c *ResourceCache) LookupId(kind Kind, id string) *ResourceCacheEntry {
	maybeEnt, ok := c.idToLabel[Ref{kind: kind, key: id}]
	if !ok {
		return nil
	}
	return &maybeEnt
}

type Generator struct {
	enabled         bool
	resourceType    string
	resourceName    string
	enabledBindings KindSet
	bindings        Mapping
	cache           ResourceCache
}

func NewGenerator(ctx context.Context, enabled bool, resourceType string, resourceName string,
	client *observe.Client, enabledBindings KindSet) (Generator, error) {
	enabled = enabled && client.Config.ExportObjectBindings
	if !enabled {
		return Generator{enabled: false}, nil
	}
	rc, err := NewResourceCache(ctx, enabledBindings, client)
	if err != nil {
		return Generator{}, err
	}
	bindings := NewMapping()
	return Generator{
		enabled:         true,
		resourceType:    resourceType,
		resourceName:    resourceName,
		enabledBindings: enabledBindings,
		bindings:        bindings,
		cache:           rc,
	}, nil
}

// lookup by kind and id, if valid and enabled then return a loval variable reference,
// otherwise return the id (no-op)
func (g *Generator) TryBind(kind Kind, id string) string {
	if !g.enabled {
		return id
	}
	var e *ResourceCacheEntry
	if kind == KindWorkspace && id == g.cache.workspaceOid.String() {
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
	terraformLocal := g.fmtTfLocalVar(kind, e.TfName)
	g.bindings[Ref{kind: kind, key: e.Label}] = Target{
		TfName:            e.TfName,
		TfLocalBindingVar: terraformLocal,
	}
	return g.fmtTfLocalVarRef(terraformLocal)
}

func (g *Generator) Generate(data interface{}) {
	mapOverJsonStringKeys(data, func(key string, value string, jsonMapNode map[string]interface{}) {
		kinds := resolveKeyToKinds(key)
		for _, kind := range kinds {
			// if not enabled, skip
			if _, found := g.enabledBindings[kind]; !found {
				continue
			}
			// try bind the name
			maybeRef := g.TryBind(kind, value)
			// if lookup succeeded, then the returned value should be a lv ref and not the
			// input id
			if maybeRef != value {
				jsonMapNode[key] = maybeRef
				break
			}
		}
	})
}

func (g *Generator) GenerateJson(jsonStr []byte) ([]byte, error) {
	if !g.enabled {
		return jsonStr, nil
	}
	serialized, err := transformJson(jsonStr, func(dataPtr *interface{}) error {
		g.Generate(*dataPtr)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return serialized, nil
}

func (g *Generator) InsertBindingsObject(data map[string]interface{}) error {
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
		return fmt.Errorf("Internal error: workspace was not resolved correctly.")
	}
	bindingsObject := BindingsObject{
		Mappings: g.bindings,
		Kinds:    enabledList,
		Workspace: Target{
			TfLocalBindingVar: g.fmtTfLocalVar(KindWorkspace, workspaceTarget.TfName),
			TfName:            workspaceTarget.TfName,
		},
		WorkspaceName: g.cache.workspaceEntry.Label,
	}
	data[bindingsKey] = bindingsObject
	return nil
}

func (g *Generator) InsertBindingsObjectJson(jsonData *types.JsonObject) (*types.JsonObject, error) {
	if !g.enabled {
		return jsonData, nil
	}
	serialized, err := transformJson([]byte(jsonData.String()), func(dataPtr *interface{}) error {
		return g.InsertBindingsObject((*dataPtr).(map[string]interface{}))
	})
	if err != nil {
		return nil, err
	}
	return types.JsonObject(serialized).Ptr(), nil
}

func (g *Generator) HasBindings() bool {
	return g.enabled && len(g.bindings) > 0
}

func (g *Generator) fmtTfLocalVar(kind Kind, targetTfName string) string {
	return sanitizeIdentifier(fmt.Sprintf("binding__%s_%s__%s_%s", g.resourceType, g.resourceName, kind, targetTfName))
}

func (g *Generator) fmtTfLocalVarRef(tfLocalVar string) string {
	return fmt.Sprintf("${local.%s}", tfLocalVar)
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
	path := strings.Split(name, "/")

	shortForm := strings.ToLower(path[len(path)-1])
	sanitized := replaceInvalid.ReplaceAllString(shortForm, "_")

	if hasLeadingDigit.MatchString(sanitized) {
		sanitized = "_" + sanitized
	}

	return sanitized
}
