package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// observe_source_dataset used to register a raw Snowflake table as a dataset
// via the saveSourceDataset mutation. It was only ever meant for internal use
// and is no longer supported. The resource is kept as a no-op so that existing
// configurations and state still plan and apply: it never calls the Observe
// API, and it will be removed in a future release.
const sourceDatasetDeprecationMessage = "`observe_source_dataset` is no longer supported and will be removed in a future release. " +
	"It no longer calls the Observe API: existing resources are kept in state unchanged, updates are not applied, " +
	"and destroying one only removes it from state without deleting the dataset. " +
	"Remove it from your configuration, or run `terraform state rm` on it."

var sourceDatasetFieldResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"type": {
			Type:     schema.TypeString,
			Required: true,
		},
		"sql_type": {
			Type:     schema.TypeString,
			Required: true,
		},
		"is_enum": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		"is_searchable": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		"is_hidden": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		"is_const": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		"is_metric": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
	},
}

func resourceSourceDataset() *schema.Resource {
	return &schema.Resource{
		Description:        "Deprecated and non-functional. " + sourceDatasetDeprecationMessage,
		DeprecationMessage: sourceDatasetDeprecationMessage,
		CreateContext:      resourceSourceDatasetCreate,
		ReadContext:        resourceSourceDatasetRead,
		UpdateContext:      resourceSourceDatasetUpdate,
		DeleteContext:      resourceSourceDatasetDelete,
		// The schema is unchanged from the functional resource so that existing
		// configurations and state remain valid.
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				DiffSuppressFunc: diffSuppressWorkspace,
				Deprecated:       "workspace is no longer required and will be ignored. It may be removed in a future version.",
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"schema": {
				Type:     schema.TypeString,
				Required: true,
			},
			"table_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"source_update_table_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"valid_from_field": {
				Type:     schema.TypeString,
				Required: true,
			},
			"batch_seq_field": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"is_insert_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"field": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     sourceDatasetFieldResource,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"freshness": {
				Deprecated:       "Freshness is not meaningful for source datasets.",
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				DiffSuppressFunc: diffSuppressDuration,
			},
		},
	}
}

// Creating a new resource is refused rather than faked: a fake resource would
// have no dataset behind it and an empty oid for anything that references it.
func resourceSourceDatasetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  "observe_source_dataset can no longer be created",
		Detail:   sourceDatasetDeprecationMessage,
	}}
}

// Read leaves the existing state untouched so that references such as
// observe_source_dataset.x.oid keep resolving.
func resourceSourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceSourceDatasetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "observe_source_dataset changes were not applied",
		Detail:   fmt.Sprintf("Dataset %s was not modified. %s", data.Get("oid"), sourceDatasetDeprecationMessage),
	}}
}

func resourceSourceDatasetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "observe_source_dataset removed from state only",
		Detail:   fmt.Sprintf("Dataset %s was not deleted. Delete it in Observe if it is no longer needed.", data.Get("oid")),
	}}
}
