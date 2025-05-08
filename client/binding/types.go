package binding

import (
	"fmt"
	"regexp"
)

type Ref struct {
	Kind Kind
	Key  string
}

type Target struct {
	TfLocalBindingVar string `json:"tf_local_binding_var"`
	TfName            string `json:"tf_name"`
}

type Mapping map[Ref]Target

type Kind string

type KindSet map[Kind]struct{}

type BindingsObject struct {
	Mappings      Mapping `json:"mappings"`
	Kinds         []Kind  `json:"kinds"`
	Workspace     Target  `json:"workspace"`
	WorkspaceName string  `json:"workspace_name"`
}

var (
	// must match the data source names, see DataSourcesMap in observe/provider.go
	KindDataset   = addKind("dataset")
	KindWorksheet = addKind("worksheet")
	KindWorkspace = addKind("workspace")
	KindUser      = addKind("user")
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
