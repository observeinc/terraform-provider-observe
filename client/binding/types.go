package binding

import (
	"fmt"
	"regexp"
)

type Ref struct {
	kind Kind
	key  string
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

const (
	KindDataset   Kind = "dataset"
	KindWorksheet Kind = "worksheet"
	KindWorkspace Kind = "workspace"
	KindUser      Kind = "user"
)

const (
	bindingsKey = "bindings"
)

var bindingRefParseRegex = regexp.MustCompile(`(.*):(.*)`)

var allKinds = NewKindSet(
	KindDataset,
	KindWorksheet,
	KindWorkspace,
	KindUser,
)

func (r *Ref) String() string {
	return fmt.Sprintf("%s:%s", r.kind, r.key)
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
	return Ref{kind: maybeKind, key: matches[2]}, true
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
