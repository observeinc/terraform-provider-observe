package observe

import (
	"strings"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"

	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func targetDecode(data *schema.ResourceData, target *gql.LayeredSettingRecordTargetInput) diag.Diagnostics {
	targetOidStr, hasOid := data.GetOk("target")
	var targetOid *oid.OID
	if hasOid {
		targetOid, _ = oid.NewOID(targetOidStr.(string))
	}

	switch targetOid.Type {
	case oid.TypeCustomer:
		target.CustomerId = &targetOid.Id
	case oid.TypeWorkspace:
		target.WorkspaceId = &targetOid.Id
	case oid.TypeFolder:
		target.FolderId = &targetOid.Id
	case oid.TypeApp:
		target.AppId = &targetOid.Id
	case oid.TypeWorksheet:
		target.WorksheetId = &targetOid.Id
	case oid.TypeDashboard:
		target.DashboardId = &targetOid.Id
	case oid.TypeMonitor:
		target.MonitorId = &targetOid.Id
	case oid.TypeDataset:
		target.DatasetId = &targetOid.Id
	case oid.TypeUser:
		target.UserId = oid.OidToUserId(*targetOid)
	default:
		return diag.Errorf("The type %q is not valid for target_oid %s", targetOid.Type, targetOid)
	}
	return nil
}

func targetEncode(data *schema.ResourceData, target *gql.LayeredSettingRecordTarget) (diags diag.Diagnostics) {
	var value string
	switch {
	case target.CustomerId != nil:
		value = oid.CustomerOid(*target.CustomerId).String()
	case target.WorkspaceId != nil:
		value = oid.WorkspaceOid(*target.WorkspaceId).String()
	case target.FolderId != nil:
		value = oid.FolderOid(*target.FolderId, "").String()
	case target.AppId != nil:
		value = oid.AppOid(*target.AppId).String()
	case target.WorksheetId != nil:
		value = oid.WorksheetOid(*target.WorksheetId).String()
	case target.DashboardId != nil:
		value = oid.DashboardOid(*target.DashboardId).String()
	case target.MonitorId != nil:
		value = oid.MonitorOid(*target.MonitorId).String()
	case target.DatasetId != nil:
		value = oid.DatasetOid(*target.DatasetId).String()
	case target.UserId != nil:
		value = oid.UserOid(*target.UserId).String()
	default:
		diags = append(diags, diag.Errorf("Unknown target for observe_layered_setting_record: %#v", *target)...)
	}
	if value != "" {
		if err := data.Set("target", value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	} else {
		diags = append(diags, diag.Errorf("the target specified is empty")...)
	}
	return diags
}

func primitiveValueDecode(data *schema.ResourceData, ret *gql.PrimitiveValueInput) diag.Diagnostics {
	valueBool, hasBool := data.GetOk("value_bool")
	//	NOTE: I rely on the fact that sizeof(int) == sizeof(int64) on modern systems
	valueInt, hasInt := data.GetOk("value_int64")
	valueFloat, hasFloat := data.GetOk("value_float64")
	valueString, hasString := data.GetOk("value_string")
	valueDuration, hasDuration := data.GetOk("value_duration")
	valueTimestamp, hasTimestamp := data.GetOk("value_timestamp")
	nvalue := 0
	var kinds []string
	if hasBool && valueBool != nil {
		b := valueBool.(bool)
		ret.Bool = &b
		nvalue++
		kinds = append(kinds, "value_bool")
	}
	if hasInt && valueInt != nil {
		i64 := types.Int64Scalar(valueInt.(int))
		ret.Int64 = &i64
		nvalue++
		kinds = append(kinds, "value_int64")
	}
	if hasFloat && valueFloat != nil {
		vlt := valueFloat.(float64)
		ret.Float64 = &vlt
		nvalue++
		kinds = append(kinds, "value_float64")
	}
	if hasString && valueString != nil {
		vstr := valueString.(string)
		ret.String = &vstr
		nvalue++
		kinds = append(kinds, "value_string")
	}
	if hasDuration && valueDuration != nil {
		dur, _ := time.ParseDuration(valueDuration.(string))
		i64 := types.Int64Scalar(dur.Nanoseconds())
		ret.Duration = &i64
		nvalue++
		kinds = append(kinds, "value_duration")
	}
	if hasTimestamp && valueTimestamp != nil {
		tsp, _ := time.Parse(time.RFC3339, valueTimestamp.(string))
		tss := types.TimeScalar(tsp)
		ret.Timestamp = &tss
		nvalue++
		kinds = append(kinds, "value_timestamp")
	}
	if nvalue == 0 {
		return diag.Errorf("A value must be specified (value_string, value_bool, etc)")
	}
	if nvalue > 1 {
		return diag.Errorf("Only one value may be specified (value_string, value_bool, etc); there are %d: %s", len(kinds), strings.Join(kinds, ","))
	}
	return nil
}

func primitiveValueEncode(data *schema.ResourceData, p *gql.PrimitiveValue) (diags diag.Diagnostics) {
	var kinds []string
	if p.Bool != nil {
		if err := data.Set("value_bool", *p.Bool); err != nil {
			return diag.FromErr(err)
		}
		kinds = append(kinds, "value_bool")
	}
	if p.Float64 != nil {
		if err := data.Set("value_float64", *p.Float64); err != nil {
			return diag.FromErr(err)
		}
		kinds = append(kinds, "value_float64")
	}
	if p.Int64 != nil {
		if err := data.Set("value_int64", int(*p.Int64)); err != nil {
			return diag.FromErr(err)
		}
		kinds = append(kinds, "value_int64")
	}
	if p.String != nil {
		if err := data.Set("value_string", *p.String); err != nil {
			return diag.FromErr(err)
		}
		kinds = append(kinds, "value_string")
	}
	if p.Timestamp != nil {
		if err := data.Set("value_timestamp", time.Time(*p.Timestamp).Format(time.RFC3339)); err != nil {
			return diag.FromErr(err)
		}
		kinds = append(kinds, "value_timestamp")
	}
	if p.Duration != nil {
		dur := time.Duration(int64(*p.Duration))
		if err := data.Set("value_duration", dur.String()); err != nil {
			return diag.FromErr(err)
		}
		kinds = append(kinds, "value_duration")
	}
	if len(kinds) == 0 {
		return diag.Errorf("There is no recognized value for config override value: %#v", p)
	}
	if len(kinds) > 1 {
		return diag.Errorf("A value can only have one kind; got %d (%s)", len(kinds), strings.Join(kinds, ", "))
	}
	return nil
}
