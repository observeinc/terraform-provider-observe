package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourceFolder() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a folder. Folders are used to group other resources together. This resource is still in development and is not yet reflected in the UI.",
		CreateContext: resourceFolderCreate,
		UpdateContext: resourceFolderUpdate,
		ReadContext:   resourceFolderRead,
		DeleteContext: resourceFolderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func newFolderConfig(data *schema.ResourceData) (input *gql.FolderInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	input = &gql.FolderInput{
		Name: &name,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	return
}

func resourceFolderCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newFolderConfig(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateFolder(ctx, id.Id, config)
	if err != nil {
		return diag.Errorf("failed to create folder: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceFolderRead(ctx, data, meta)...)
}

func resourceFolderUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newFolderConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateFolder(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update folder: %s", err.Error())
	}

	return append(diags, resourceFolderRead(ctx, data, meta)...)
}

func resourceFolderRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	folder, err := client.GetFolder(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read folder: %s", err.Error())
	}

	return folderToResourceData(folder, data)
}

func folderToResourceData(folder *gql.Folder, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(folder.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", folder.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", folder.IconUrl); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", folder.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", folder.Oid().String()); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceFolderDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteFolder(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete folder: %s", err.Error())
	}
	return diags
}
