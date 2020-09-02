package client

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	oidRegex      = regexp.MustCompile(`o:::(?P<type>[a-z]+):(?P<id>\d+)(/(?P<version>.*))?`)
	errInvalidOID = errors.New("invalid oid")
)

type Type string

const (
	TypeDataset       Type = "dataset"
	TypeWorkspace          = "workspace"
	TypeBookmarkGroup      = "bookmarkgroup"
	TypeBookmark           = "bookmark"
)

func (t Type) IsValid() bool {
	switch t {
	case TypeDataset, TypeWorkspace, TypeBookmarkGroup:
		return true
	default:
		return false
	}
}

type OID struct {
	Type    Type
	ID      string
	Version *string
}

func (o *OID) String() string {
	id := o.ID
	if o.Version != nil {
		id += "/" + *o.Version
	}
	return fmt.Sprintf("o:::%s:%s", o.Type, id)
}

func NewOID(s string) (*OID, error) {
	match := oidRegex.FindStringSubmatch(s)
	if len(match) == 0 {
		return nil, errInvalidOID
	}

	oid := new(OID)
	for i, name := range oidRegex.SubexpNames() {
		switch name {
		case "type":
			if t := Type(match[i]); t.IsValid() {
				oid.Type = t
			} else {
				return nil, fmt.Errorf("unknown type: %w", errInvalidOID)
			}
		case "id":
			oid.ID = match[i]
		case "version":
			if s := match[i]; s != "" {
				oid.Version = &s
			}
		}
	}

	if oid.ID == "" {
		return nil, errInvalidOID
	}
	return oid, nil
}
