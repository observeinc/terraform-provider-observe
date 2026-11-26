package observe

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func sourceDatasetDataSourceDescription(key string) string {
	return descriptions.Get("source_dataset", "data_source_schema", key)
}

func dataSourceSourceDataset() *schema.Resource {
	computedString := func(key string) *schema.Schema {
		return &schema.Schema{Type: schema.TypeString, Computed: true, Description: sourceDatasetDataSourceDescription(key)}
	}
	computedInt := func(key string) *schema.Schema {
		return &schema.Schema{Type: schema.TypeInt, Computed: true, Description: sourceDatasetDataSourceDescription(key)}
	}
	return &schema.Resource{
		Description: descriptions.Get("source_dataset", "data_source_description"),
		ReadContext: dataSourceSourceDatasetRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"id", "label"},
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateID(),
				Description:      sourceDatasetDataSourceDescription("id"),
			},
			"label": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"id", "label"},
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      sourceDatasetDataSourceDescription("label"),
			},
			"description":                   computedString("description"),
			"icon_url":                      computedString("icon_url"),
			"object_tags":                   objectTagsSchemaFieldComputed(),
			"data_retention_days":           computedInt("data_retention_days"),
			"effective_data_retention_days": computedInt("effective_data_retention_days"),
			"schema_type":                   computedString("schema_type"),
			"storage":                       computedString("storage"),
			"storage_integration":           computedString("storage_integration"),
			"external_write": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: sourceDatasetDataSourceDescription("external_write"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
		},
	}
}

func dataSourceSourceDatasetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observeclient.Client).Rest
	var ds *rest.SourceDatasetResource
	if id := data.Get("id").(string); id != "" {
		var err error
		if ds, err = client.GetSourceDataset(ctx, id); err != nil {
			if rest.HasStatusCode(err, http.StatusNotFound) {
				return diag.Errorf("source dataset %s not found", id)
			}
			return sourceDatasetAPIError("read", id, err)
		}
	} else {
		label := data.Get("label").(string)
		var diags diag.Diagnostics
		if ds, diags = lookupSourceDatasetByLabel(ctx, client, label); diags.HasError() {
			return diags
		}
	}
	data.SetId(ds.Id)
	return sourceDatasetToResourceData(data, ds)
}

// lookupSourceDatasetByLabel returns the source dataset whose label is
// exactly label. Labels are unique per workspace.
func lookupSourceDatasetByLabel(ctx context.Context, client *rest.Client, label string) (*rest.SourceDatasetResource, diag.Diagnostics) {
	// A Go-quoted string is a valid CEL string literal.
	matches, err := client.ListSourceDatasets(ctx, "label == "+strconv.Quote(label))
	if err != nil {
		return nil, sourceDatasetAPIError("look up", strconv.Quote(label), err)
	}
	switch len(matches) {
	case 0:
		return nil, diag.Errorf("no source dataset has label %q", label)
	case 1:
		return &matches[0], nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.Id
		}
		return nil, diag.Errorf("%d source datasets have label %q (ids %s); set id instead", len(matches), label, strings.Join(ids, ", "))
	}
}
