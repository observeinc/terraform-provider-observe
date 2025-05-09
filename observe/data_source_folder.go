package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceFolder() *schema.Resource {
	return &schema.Resource{
		Description: "Fetches data for an existing Observe folder.",

		ReadContext: dataSourceFolderRead,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"name": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
				Description:  descriptions.Get("folder", "schema", "name"),
			},
			"id": {
				Type:             schema.TypeString,
				ExactlyOneOf:     []string{"name", "id"},
				Optional:         true,
				ValidateDiagFunc: validateID(),
				Description:      descriptions.Get("folder", "schema", "id"),
			},
			// computed values
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("folder", "schema", "description"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
		},
	}
}

func dataSourceFolderRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client     = meta.(*observe.Client)
		name       = data.Get("name").(string)
		explicitId = data.Get("id").(string)
	)

	implicitId, _ := oid.NewOID(data.Get("workspace").(string))

	var f *gql.Folder
	var err error

	if explicitId != "" {
		f, err = client.GetFolder(ctx, explicitId)
	} else if name != "" {
		defer func() {
			// right now SDK does not report where this error happened,
			// so we need to provide a little extra context
			for i := range diags {
				diags[i].Detail = fmt.Sprintf("failed to read folder %q", name)
			}
		}()

		f, err = client.LookupFolder(ctx, implicitId.Id, name)
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	data.SetId(f.Id)
	return folderToResourceData(f, data)
}
