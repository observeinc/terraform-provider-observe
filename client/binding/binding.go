package binding

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

var (
	replaceInvalid  = regexp.MustCompile(`([^0-9a-zA-Z-_]+)`)
	hasLeadingDigit = regexp.MustCompile(`^[0-9]`)
)

type ResourceCache struct {
	idToLabel Mapping
}

type NewResourceCacheOptArgs struct {
	workspaceId string
}

func NewResourceCache(ctx context.Context, kinds KindSet, client *observe.Client, optArgs NewResourceCacheOptArgs) (ResourceCache, error) {
	var cache = ResourceCache{idToLabel: make(Mapping)}
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
			worksheets, err := client.ListWorksheetIdLabelOnly(ctx, optArgs.workspaceId)
			if err != nil {
				return cache, err
			}
			for _, wk := range worksheets {
				cache.addEntry(KindWorksheet, wk.Label, wk.Id, &disambiguator, existingResourceNames)
			}
		case KindWorkspace:
			// special case: one workspace per customer, will always be referred to as
			// "default"
			workspaces, err := client.ListWorkspaces(ctx)
			if err != nil {
				return cache, err
			}
			cache.addEntry(KindWorkspace, "Default", workspaces[0].Oid().String(), &disambiguator, existingResourceNames)
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
	c.idToLabel[Ref{kind: kind, key: id}] = Target{
		TfName: resourceName,
		Value:  label,
	}
}

func (c *ResourceCache) LookupId(kind Kind, id string) *Target {
	maybeLabel, ok := c.idToLabel[Ref{kind: kind, key: id}]
	if !ok {
		return nil
	}
	return &maybeLabel
}

type Generator struct {
	enabled         bool
	resourceType    string
	resourceName    string
	enabledBindings KindSet
	client          *observe.Client
	bindings        Mapping
	cache           ResourceCache
}

func NewGenerator(ctx context.Context, enabled bool, resourceType string, resourceName string,
	client *observe.Client, enabledBindings KindSet) (Generator, error) {
	enabled = enabled && client.Config.ExportObjectBindings
	if !enabled {
		return Generator{enabled: false}, nil
	}
	rc, err := NewResourceCache(ctx, enabledBindings, client, NewResourceCacheOptArgs{})
	if err != nil {
		return Generator{}, err
	}
	bindings := NewMapping()
	return Generator{
		enabled:         true,
		resourceType:    resourceType,
		resourceName:    resourceName,
		enabledBindings: enabledBindings,
		client:          client,
		bindings:        bindings,
		cache:           rc,
	}, nil
}

func (g *Generator) TryBind(kind Kind, id string) string {
	if ! g.enabled {
		return id
	}
	if t := g.cache.LookupId(kind, id); t != nil {
		tfLocal := sanitizeIdentifier(fmt.Sprintf("binding__%s_%s__%s_%s", g.resourceType, g.resourceName, kind, t.TfName))
		g.bindings[Ref{kind: kind, key: t.Value}] = Target{
			Value: t.Value,
			TfName: t.TfName,
			TfLocalBindingVar: tfLocal,
		}
		return fmt.Sprintf("${local.%s}", tfLocal)
	}
	return id
}

func (g *Generator) Generate(ctx context.Context, data interface{}) {
	mapOverJsonStringKeys(data, func(key string, value string, jsonMapNode map[string]interface{}) {
		kinds := resolveKeyToKinds(key)
		for _, kind := range kinds {
			// if not enabled, skip
			if _, found := g.enabledBindings[kind]; !found {
				continue
			}
			// Lookup resource in cache
			maybeLabel := g.cache.LookupId(kind, value)
			// update with binding
			if maybeLabel != nil {
				b := Ref{kind: kind, key: maybeLabel.Value}
				terraformLocal := sanitizeIdentifier(fmt.Sprintf("binding__%s_%s__%s_%s", g.resourceType, g.resourceName, kind, maybeLabel.TfName))
				g.bindings[b] = Target{
					Value:             value,
					TfName:            maybeLabel.TfName,
					TfLocalBindingVar: terraformLocal,
				}
				jsonMapNode[key] = fmt.Sprintf("${local.%s}", terraformLocal)
				break
			}
		}
	})
}

func (g *Generator) GenerateJson(ctx context.Context, jsonStr []byte) ([]byte, error) {
	if !g.enabled {
		return jsonStr, nil
	}
	serialized, err := transformJson(jsonStr, func(dataPtr *interface{}) error {
		g.Generate(ctx, *dataPtr)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return serialized, nil
}

func (g *Generator) InsertBindingsObject(data map[string]interface{}) {
	enabledList := make([]Kind, 0)
	for binding := range g.enabledBindings {
		enabledList = append(enabledList, binding)
	}
	bindingsObject := BindingsObject{
		Mappings: g.bindings,
		Kinds:    enabledList,
	}
	data[bindingsKey] = bindingsObject
}

func (g *Generator) InsertBindingsObjectJson(jsonData *types.JsonObject) (*types.JsonObject, error) {
	if !g.enabled {
		return jsonData, nil
	}
	serialized, err := transformJson([]byte(jsonData.String()), func(dataPtr *interface{}) error {
		g.InsertBindingsObject((*dataPtr).(map[string]interface{}))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return types.JsonObject(serialized).Ptr(), nil
}

func resolveKeyToKinds(key string) []Kind {
	switch key {
	case "id":
		return []Kind{KindDataset, KindWorksheet}
	case "datasetId":
		fallthrough
	case "targetDataset":
		fallthrough
	case "dataset":
		return []Kind{KindDataset}
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
