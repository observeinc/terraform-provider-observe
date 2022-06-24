package oid

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
	TypeApp             Type = "app"
	TypeBoard           Type = "board"
	TypeBookmark        Type = "bookmark"
	TypeBookmarkGroup   Type = "bookmarkgroup"
	TypeChannel         Type = "channel"
	TypeChannelAction   Type = "channelaction"
	TypeDashboard       Type = "dashboard"
	TypeDataset         Type = "dataset"
	TypeDatastream      Type = "datastream"
	TypeDatastreamToken Type = "datastreamtoken"
	TypeFolder          Type = "folder"
	TypeLink            Type = "link"
	TypeMonitor         Type = "monitor"
	TypePoller          Type = "poller"
	TypePreferredPath   Type = "preferredpath"
	TypeWorksheet       Type = "worksheet"
	TypeWorkspace       Type = "workspace"
)

func (t Type) IsValid() bool {
	switch t {
	case TypeApp:
	case TypeBoard:
	case TypeBookmark:
	case TypeBookmarkGroup:
	case TypeChannel:
	case TypeChannelAction:
	case TypeDashboard:
	case TypeDataset:
	case TypeDatastream:
	case TypeDatastreamToken:
	case TypeFolder:
	case TypeLink:
	case TypeMonitor:
	case TypePoller:
	case TypePreferredPath:
	case TypeWorksheet:
	case TypeWorkspace:
	default:
		return false
	}
	return true
}

type OID struct {
	Type    Type
	Id      string
	Version *string
}

func (o OID) String() string {
	id := o.Id
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
			oid.Id = match[i]
		case "version":
			if s := match[i]; s != "" {
				oid.Version = &s
			}
		}
	}

	if oid.Id == "" {
		return nil, errInvalidOID
	}
	return oid, nil
}

func AppOid(id string) OID {
	return OID{Id: id, Type: TypeApp}
}

func BoardOid(id string) OID {
	return OID{Id: id, Type: TypeBoard}
}

func BookmarkOid(id string) OID {
	return OID{Id: id, Type: TypeBookmark}
}

func BookmarkGroupOid(id string) OID {
	return OID{Id: id, Type: TypeBookmarkGroup}
}

func ChannelOid(id string) OID {
	return OID{Id: id, Type: TypeChannel}
}

func ChannelActionOid(id string) OID {
	return OID{Id: id, Type: TypeChannelAction}
}

func DashboardOid(id string) OID {
	return OID{Id: id, Type: TypeDashboard}
}

func DatasetOid(id string) OID {
	return OID{Id: id, Type: TypeDataset}
}

func DatastreamOid(id string) OID {
	return OID{Id: id, Type: TypeDatastream}
}

func DatastreamTokenOid(id string) OID {
	return OID{Id: id, Type: TypeDatastreamToken}
}

func FolderOid(id string, wsid string) OID {
	return OID{Id: wsid, Type: TypeFolder, Version: &id}
}

func LinkOid(id string) OID {
	return OID{Id: id, Type: TypeLink}
}

func MonitorOid(id string) OID {
	return OID{Id: id, Type: TypeMonitor}
}

func PollerOid(id string) OID {
	return OID{Id: id, Type: TypePoller}
}

func PreferredPathOid(id string) OID {
	return OID{Id: id, Type: TypePreferredPath}
}

func WorksheetOid(id string) OID {
	return OID{Id: id, Type: TypeWorksheet}
}

func WorkspaceOid(id string) OID {
	return OID{Id: id, Type: TypeWorkspace}
}
