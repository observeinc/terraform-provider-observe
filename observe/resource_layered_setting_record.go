package observe

import (
	"context"
	"fmt"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
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
				ValidateDiagFunc: validateOID(oid.TypeWorkspace, oid.TypeFolder, oid.TypeApp, oid.TypeMonitor, oid.TypeWorksheet, oid.TypeDashboard, oid.TypeDataset, oid.TypeCustomer, oid.TypeUser),
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
