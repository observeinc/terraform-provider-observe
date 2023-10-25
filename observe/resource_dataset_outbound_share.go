package observe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"

	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceDatasetOutboundShare() *schema.Resource {
	return &schema.Resource{
		Description:   "Shares a dataset with an external Snowflake account.",
		CreateContext: resourceDatasetOutboundShareCreate,
		ReadContext:   resourceDatasetOutboundShareRead,
		UpdateContext: resourceDatasetOutboundShareUpdate,
		DeleteContext: resourceDatasetOutboundShareDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"folder": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true, // Default folder when unset
				ValidateDiagFunc: validateOID(oid.TypeFolder),
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A descriptive name for the dataset sharing configuration. Displayed within Observe, not used in Snowflake.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the dataset sharing configuration.",
			},
			"dataset": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				DiffSuppressFunc: diffSuppressOIDVersion,
				Description:      "The OID of the dataset to be shared",
			},
			"outbound_share": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeSnowflakeOutboundShare),
				Description:      "The OID of the Observe Snowflake outbound share where the dataset will be shared",
			},
			"schema_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the schema within the shared database where the dataset view will be created.",
			},
			"view_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the view that will be created in the shared database, within the specified schema.",
			},
			"freshness_goal": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressTimeDuration,
			},
		},
	}
}

func newDatasetOutboundShare(d *schema.ResourceData) (*gql.DatasetOutboundShareInput, diag.Diagnostics) {
	freshnessGoal, err := time.ParseDuration(d.Get("freshness_goal").(string))
	if err != nil {
		return nil, diag.FromErr(err)
	}

	input := &gql.DatasetOutboundShareInput{
		Name:          d.Get("name").(string),
		SchemaName:    d.Get("schema_name").(string),
		ViewName:      d.Get("view_name").(string),
		FreshnessGoal: types.Int64Scalar(freshnessGoal),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	if v, ok := d.GetOk("folder"); ok {
		id, err := oid.NewOID(v.(string))
		if err != nil {
			return nil, diag.FromErr(err)
		}

		input.FolderId = id.Version
	}

	return input, nil
}

func resourceDatasetOutboundShareCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	workspaceId, err := oid.NewOID(d.Get("workspace").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	datasetId, err := oid.NewOID(d.Get("dataset").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	shareId, err := oid.NewOID(d.Get("outbound_share").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	input, diags := newDatasetOutboundShare(d)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateDatasetOutboundShare(ctx, workspaceId.Id, datasetId.Id, shareId.Id, input)
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to create dataset outbound share",
			Detail:   err.Error(),
		})
	}

	d.SetId(result.Id)

	if wd := waitDatasetOutboundShareLive(ctx, result, d.Timeout(schema.TimeoutCreate)-time.Second*10, client); wd.HasError() {
		return append(diags, wd...)
	}

	return append(diags, resourceDatasetOutboundShareRead(ctx, d, m)...)
}

func resourceDatasetOutboundShareUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	input, diags := newDatasetOutboundShare(d)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateDatasetOutboundShare(ctx, d.Id(), input)
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to update dataset outbound share",
			Detail:   err.Error(),
		})
	}

	if wd := waitDatasetOutboundShareLive(ctx, result, d.Timeout(schema.TimeoutUpdate)-time.Second*10, client); wd.HasError() {
		return append(diags, wd...)
	}

	return append(diags, resourceDatasetOutboundShareRead(ctx, d, m)...)
}

func resourceDatasetOutboundShareRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*observe.Client)

	datasetShare, err := client.GetDatasetOutboundShare(ctx, d.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to read dataset outbound share",
			Detail:   err.Error(),
		})
	}

	if err := d.Set("oid", datasetShare.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("name", datasetShare.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if datasetShare.Description != nil {
		if err := d.Set("description", *datasetShare.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := d.Set("workspace", oid.WorkspaceOid(datasetShare.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("folder", oid.FolderOid(datasetShare.FolderId, datasetShare.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("dataset", oid.DatasetOid(datasetShare.DatasetID).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("outbound_share", oid.SnowflakeOutboundShareOid(datasetShare.OutboundShareID).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("schema_name", datasetShare.SchemaName); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("view_name", datasetShare.ViewName); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("freshness_goal", types.DurationScalar(datasetShare.FreshnessGoal).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceDatasetOutboundShareDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*observe.Client)

	if err := client.DeleteDatasetOutboundShare(ctx, d.Id()); err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to delete dataset outbound share",
			Detail:   err.Error(),
		})
	}

	return diags
}

func waitDatasetOutboundShareLive(ctx context.Context, ds *gql.DatasetOutboundShare, timeout time.Duration, client *observe.Client) (diags diag.Diagnostics) {
	if ds.Status.State == gql.DatasetOutboundShareStateLive {
		return diags
	}

	c := &retry.StateChangeConf{
		Pending: []string{
			string(gql.DatasetOutboundShareStatePending),
		},
		Target: []string{
			string(gql.DatasetOutboundShareStateLive),
		},
		Refresh: func() (any, string, error) {
			resp, err := client.GetDatasetOutboundShare(ctx, ds.Id)
			if err != nil {
				return nil, "", err
			}

			switch resp.Status.State {
			case gql.DatasetOutboundShareStateError, gql.DatasetOutboundShareStateUnavailable:
				msg := fmt.Sprintf("dataset outbound share is in %q state", resp.Status.State)
				if resp.Status.Error != nil {
					msg += fmt.Sprintf(": %s", *resp.Status.Error)
				}

				return nil, string(resp.Status.State), errors.New(msg)
			}

			return resp, string(resp.Status.State), nil
		},
		Timeout:    timeout,
		Delay:      3 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	_, err := c.WaitForStateContext(ctx)
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error while waiting for dataset outbound share to be live",
			Detail:   err.Error(),
		})
	}

	return diags
}
