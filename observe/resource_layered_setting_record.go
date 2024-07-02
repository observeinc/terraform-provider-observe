package observe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLayeredSettingRecord() *schema.Resource {
	return &schema.Resource{
		Description:   "Managed a layered setting record. Layered settings can be used to set configurable parameters at different levels, including specific objects, workspaces, or customers.",
		CreateContext: resourceLayeredSettingRecordCreate,
		ReadContext:   resourceLayeredSettingRecordRead,
		UpdateContext: resourceLayeredSettingRecordUpdate,
		DeleteContext: resourceLayeredSettingRecordDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"setting": {
				Type:     schema.TypeString,
				Required: true,
				//	TODO: we could generate a list of all valid settings, but
				//	keeping that up to date is a never-ending tail-chasing job
				//	until we get build integration with monorepo.
			},
			"value_int64": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"value_float64": {
				Type:     schema.TypeFloat,
				Optional: true,
			},
			"value_bool": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"value_string": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"value_duration": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressDuration,
			},
			"value_timestamp": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimestamp,
			},
			"target": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(maps.Keys(layeredSettingTargetFuncs)...),
				DiffSuppressFunc: diffSuppressOIDVersion,
			},
		},
	}
}

func newLayeredSettingRecordConfig(data *schema.ResourceData) (input *gql.LayeredSettingRecordInput, diags diag.Diagnostics) {
	workspaceOid, _ := oid.NewOID(data.Get("workspace").(string))
	name := data.Get("name").(string)
	setting := data.Get("setting").(string)

	ret := gql.LayeredSettingRecordInput{
		Name:        name,
		WorkspaceId: workspaceOid.Id,
	}
	ret.SettingAndTargetScope.Setting = setting
	if diags = targetDecode(data, &ret.SettingAndTargetScope.Target); diags != nil {
		return nil, diags
	}
	if diags = primitiveValueDecode(data, &ret.Value); diags != nil {
		return nil, diags
	}

	return &ret, nil
}

func layeredSettingRecordToResourceData(c *gql.LayeredSettingRecord, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(c.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("name", c.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("setting", c.SettingAndTargetScope.Setting); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if dd := targetEncode(data, &c.SettingAndTargetScope.Target); len(dd) > 0 {
		diags = append(diags, dd...)
	}
	if dd := primitiveValueEncode(data, &c.Value); len(dd) > 0 {
		diags = append(diags, dd...)
	}
	data.SetId(c.Id)

	return diags
}

func resourceLayeredSettingRecordCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	setting, diags := newLayeredSettingRecordConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateLayeredSettingRecord(ctx, setting)
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create layeredsetting",
			Detail:   err.Error(),
		})
	}

	data.SetId(result.Id)
	return append(diags, resourceLayeredSettingRecordRead(ctx, data, meta)...)
}

func resourceLayeredSettingRecordRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetLayeredSettingRecord(ctx, data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve layeredsetting [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}
	return layeredSettingRecordToResourceData(result, data)
}

func resourceLayeredSettingRecordUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	setting, diags := newLayeredSettingRecordConfig(data)
	if diags.HasError() {
		return diags
	}
	dataid := data.Id()
	setting.Id = &dataid

	result, err := client.UpdateLayeredSettingRecord(ctx, setting)
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update layeredsetting [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return layeredSettingRecordToResourceData(result, data)
}

func resourceLayeredSettingRecordDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteLayeredSettingRecord(ctx, data.Id()); err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to delete layeredsetting [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}
	return diags
}

// layeredSettingTargetFunc is a function that sets the corresponding ID field
// in the target object
type layeredSettingTargetFunc func(*gql.LayeredSettingRecordTargetInput, *oid.OID)

// layeredSettingTargetFuncs is a map of OID types to target functions that set
// the proper ID field based on the OID type.
var layeredSettingTargetFuncs = map[oid.Type]layeredSettingTargetFunc{
	oid.TypeCustomer: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.CustomerId = &o.Id
	},
	oid.TypeWorkspace: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.WorkspaceId = &o.Id
	},
	oid.TypeFolder: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.FolderId = &o.Id
	},
	oid.TypeApp: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.AppId = &o.Id
	},
	oid.TypeWorksheet: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.WorksheetId = &o.Id
	},
	oid.TypeDashboard: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.DashboardId = &o.Id
	},
	oid.TypeMonitor: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.MonitorId = &o.Id
	},
	oid.TypeDataset: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.DatasetId = &o.Id
	},
	oid.TypeDatastream: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.DatastreamId = &o.Id
	},
	oid.TypeUser: func(t *gql.LayeredSettingRecordTargetInput, o *oid.OID) {
		t.UserId = oid.OidToUserId(*o)
	},
}

func targetDecode(data *schema.ResourceData, target *gql.LayeredSettingRecordTargetInput) diag.Diagnostics {
	targetOidStr, hasOid := data.GetOk("target")
	var targetOid *oid.OID
	if hasOid {
		targetOid, _ = oid.NewOID(targetOidStr.(string))
	}

	setTarget, ok := layeredSettingTargetFuncs[targetOid.Type]
	if !ok {
		return diag.FromErr(fmt.Errorf("invalid target type: %s", targetOid.Type))
	}

	setTarget(target, targetOid)

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
	case target.DatastreamId != nil:
		value = oid.DatastreamOid(*target.DatastreamId).String()
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
	// value_bool must be read with GetOkExists because it is both optional and has no default
	// GetOk returns !ok for the zero value of the type
	// GetOkExists, which is deprecated, is the only way to handle this schema as declared
	// Fixing this requires refactoring the schema (breaking). A more appropriate way to handle this API would be to have `type` and `value` fields, both strings.
	// The `value` would be a JSON-encoded string of the appropriate type.
	valueBool, hasBool := data.GetOkExists("value_bool")

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
		return diag.Errorf("Only one value may be specified (value_string, value_bool, etc); there are %d: %s.", len(kinds), strings.Join(kinds, ","))
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
