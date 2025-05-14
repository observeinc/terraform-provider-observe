package binding

import (
	"fmt"
	"regexp"
)

type Ref struct {
	Kind Kind
	Key  string // id when used in ResourceCache, label when used in Mapping
}

type Target struct {
	// Name of the local variable that will be generated for this target, used in place of the raw ids
	TfLocalBindingVar string `json:"tf_local_binding_var"`
	// Name of the terraform data source that will be generated for this target
	TfName string `json:"tf_name"`
	// When generating locals, used to determine whether we should do ${data_source}.id or ${data_source}.oid
	IsOid bool `json:"is_oid"`
}

// A binding, i.e. mapping of resource kind + label -> terraform local variable name (which the ids have been replaced with)
type Mapping map[Ref]Target

type Kind string

type KindSet map[Kind]struct{}

// BindingsObject contains all the information necessary to generate terraform
// data sources and local variable definitions to support the local variable
// references replacing the raw ids in the resource data.
type BindingsObject struct {
	Mappings      Mapping `json:"mappings"`
	Kinds         []Kind  `json:"kinds"`
	Workspace     Target  `json:"workspace"`
	WorkspaceName string  `json:"workspace_name"`
}

var (
	// must match the data source names, see DataSourcesMap in observe/provider.go
	KindDataset         = addKind("dataset")
	KindWorksheet       = addKind("worksheet")
	KindWorkspace       = addKind("workspace")
	KindUser            = addKind("user")
	KindDashboard       = addKind("dashboard")
	KindMonitorV2       = addKind("monitor_v2")
	KindMonitorV2Action = addKind("monitor_v2_action")
	KindMonitor         = addKind("monitor")
)

const (
	bindingsKey = "bindings"
)

var bindingRefParseRegex = regexp.MustCompile(`(.*):(.*)`)

var allKinds = NewKindSet()

func addKind(k Kind) Kind {
	allKinds[k] = struct{}{}
	return k
}

func (r *Ref) String() string {
	return fmt.Sprintf("%s:%s", r.Kind, r.Key)
}

func (r Ref) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

func (r *Ref) UnmarshalText(text []byte) error {
	ref, ok := NewRefFromString(string(text))
	if !ok {
		return fmt.Errorf("failed to unmarshal reference type")
	}
	*r = ref
	return nil
}

func NewRefFromString(s string) (Ref, bool) {
	matches := bindingRefParseRegex.FindStringSubmatch(s)
	if len(matches) == 0 {
		return Ref{}, false
	}
	maybeKind := Kind(matches[1])
	if _, ok := allKinds[maybeKind]; !ok {
		return Ref{}, false
	}
	return Ref{Kind: maybeKind, Key: matches[2]}, true
}

func NewMapping() Mapping {
	return make(Mapping)
}

func NewKindSet(kinds ...Kind) KindSet {
	set := make(KindSet)
	var empty struct{}
	for _, kind := range kinds {
		set[kind] = empty
	}
	return set
}
