package oid

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

var (
	oidRegex      = regexp.MustCompile(`o:::(?P<type>[a-z]+):(?P<id>\d+)(/(?P<version>.*))?`)
	ornRegex      = regexp.MustCompile(`o::(?P<customer>\d+):(?P<type>[a-z]+):(?P<id>\d+)?`)
	errInvalidOID = errors.New("invalid oid")
)

type Type string

const (
	TypeApp                     Type = "app"
	TypeAppDataSource           Type = "appdatasource"
	TypeBoard                   Type = "board"
	TypeBookmark                Type = "bookmark"
	TypeBookmarkGroup           Type = "bookmarkgroup"
	TypeChannel                 Type = "channel"
	TypeChannelAction           Type = "channelaction"
	TypeCustomer                Type = "customer"
	TypeDashboard               Type = "dashboard"
	TypeDataset                 Type = "dataset"
	TypeDatastream              Type = "datastream"
	TypeDatastreamToken         Type = "datastreamtoken"
	TypeFiledrop                Type = "filedrop"
	TypeFolder                  Type = "folder"
	TypeLayeredSettingRecord    Type = "layeredsettingrecord"
	TypeLink                    Type = "link"
	TypeMonitor                 Type = "monitor"
	TypeMonitorV2               Type = "monitorV2"
	TypeMonitorAction           Type = "monitoraction"
	TypeMonitorActionAttachment Type = "monitoractionattachment"
	TypePoller                  Type = "poller"
	TypePreferredPath           Type = "preferredpath"
	TypeUser                    Type = "user"
	TypeWorksheet               Type = "worksheet"
	TypeWorkspace               Type = "workspace"
	TypeRbacGroup               Type = "rbacgroup"
	TypeRbacGroupmember         Type = "rbacgroupmember"
	TypeRbacStatement           Type = "rbacstatement"
	TypeSnowflakeOutboundShare  Type = "snowflakeoutboundshare"
	TypeDatasetOutboundShare    Type = "datasetoutboundshare"
)

func (t Type) IsValid() bool {
	switch t {
	case TypeApp:
	case TypeAppDataSource:
	case TypeBoard:
	case TypeBookmark:
	case TypeBookmarkGroup:
	case TypeChannel:
	case TypeChannelAction:
	case TypeCustomer:
	case TypeDashboard:
	case TypeDataset:
	case TypeDatastream:
	case TypeDatastreamToken:
	case TypeFolder:
	case TypeLayeredSettingRecord:
	case TypeLink:
	case TypeMonitor:
	case TypeMonitorAction:
	case TypeMonitorActionAttachment:
	case TypeMonitorV2:
	case TypePoller:
	case TypePreferredPath:
	case TypeUser:
	case TypeWorksheet:
	case TypeWorkspace:
	case TypeRbacGroup:
	case TypeRbacGroupmember:
	case TypeRbacStatement:
	case TypeSnowflakeOutboundShare:
	case TypeDatasetOutboundShare:
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
	id := strings.Trim(o.Id, "\"")
	if o.Version != nil {
		id += "/" + *o.Version
	}
	return fmt.Sprintf("o:::%s:%s", o.Type, id)
}

func NewOID(s string) (*OID, error) {
	orn, oidStr := extractORN(s)
	match := oidRegex.FindStringSubmatch(oidStr)
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
			oid.Id = oneOf(orn, match[i])
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

func oneOf(vals ...string) string {
	for _, s := range vals {
		if s != "" {
			return s
		}
	}
	return ""
}

// extractORN returns an ORN and an OID
//   - If input has an ORN o:::rbacgroup:o::123458:rbacgroup:8000002523
//     Output would be: "o::123458:rbacgroup:8000002523", "o:::rbacgroup:8000002523"
//   - If input is a regular OID o:::user:12345678
//     Output would be: "", "o:::user:12345678"
func extractORN(s string) (orn, oid string) {
	match := ornRegex.FindStringSubmatch(s)
	if len(match) == 0 {
		return "", s
	}
	orn = match[0]
	id := match[len(match)-1]
	oid = strings.Replace(s, orn, id, -1)
	return orn, oid
}

func AppOid(id string) OID {
	return OID{Id: id, Type: TypeApp}
}

func AppDataSourceOid(id string) OID {
	return OID{Id: id, Type: TypeAppDataSource}
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

func CustomerOid(id string) OID {
	return OID{Id: id, Type: TypeCustomer}
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

func LayeredSettingRecordOid(id string) OID {
	return OID{Id: id, Type: TypeLayeredSettingRecord}
}

func LinkOid(id string) OID {
	return OID{Id: id, Type: TypeLink}
}

func MonitorOid(id string) OID {
	return OID{Id: id, Type: TypeMonitor}
}

func MonitorActionOid(id string) OID {
	return OID{Id: id, Type: TypeMonitorAction}
}

func MonitorV2Oid(id string) OID {
	return OID{Id: id, Type: TypeMonitorV2}
}

func PollerOid(id string) OID {
	return OID{Id: id, Type: TypePoller}
}

func PreferredPathOid(id string) OID {
	return OID{Id: id, Type: TypePreferredPath}
}

func UserOid(uid types.UserIdScalar) OID {
	return OID{Id: uid.String(), Type: TypeUser}
}

func OidToUserId(oid OID) *types.UserIdScalar {
	if oid.Type != TypeUser {
		panic(fmt.Sprintf("How did a %q OID get used as a UserId?", oid.Type))
	}
	uid, err := strconv.ParseInt(oid.Id, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("UserId should be integer: %q is not integer: %s", oid.Id, err.Error()))
	}
	ret := types.UserIdScalar(uid)
	return &ret
}

func WorksheetOid(id string) OID {
	return OID{Id: id, Type: TypeWorksheet}
}

func WorkspaceOid(id string) OID {
	return OID{Id: id, Type: TypeWorkspace}
}

func RbacGroupOid(id string) OID {
	// note: id is an ORN of the form `o::<customerid>:rbacgroup:<coid>`
	// the generated OID currently adds a prefix `o:::rbacgroup:` to the above string
	return OID{Id: id, Type: TypeRbacGroup}
}

func RbacGroupmemberOid(id string) OID {
	// note: id is an ORN of the form `o::<customerid>:rbacgroupmember:<coid>`
	// the generated OID currently adds a prefix `o:::rbacgroupmember:` to the above string
	return OID{Id: id, Type: TypeRbacGroupmember}
}

func RbacStatementOid(id string) OID {
	// note: id is an ORN of the form `o::<customerid>:rbacstatement:<coid>`
	// the generated OID currently adds a prefix `o:::rbacstatement:` to the above string
	return OID{Id: id, Type: TypeRbacStatement}
}

func SnowflakeOutboundShareOid(id string) OID {
	return OID{Id: id, Type: TypeSnowflakeOutboundShare}
}
