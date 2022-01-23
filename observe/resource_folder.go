package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceFolder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFolderCreate,
		UpdateContext: resourceFolderUpdate,
		ReadContext:   resourceFolderRead,
		DeleteContext: resourceFolderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
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

func newFolderConfig(data *schema.ResourceData) (config *observe.FolderConfig, diags diag.Diagnostics) {
	config = &observe.FolderConfig{
		Name: data.Get("name").(string),
	}

	if v, ok := data.GetOk("icon_url"); ok {
		s := v.(string)
		config.IconURL = &s
	}

	if v, ok := data.GetOk("description"); ok {
		s := v.(string)
		config.Description = &s
	}

	return
}

func resourceFolderCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newFolderConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateFolder(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create workspace: %s", err.Error())
	}

	data.SetId(result.ID)
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
		return diag.Errorf("failed to update workspace: %s", err.Error())
	}

	return append(diags, resourceFolderRead(ctx, data, meta)...)
}

func resourceFolderRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	folder, err := client.GetFolder(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read folder: %s", err.Error())
	}

	return folderToResourceData(folder, data)
}

func folderToResourceData(folder *observe.Folder, data *schema.ResourceData) (diags diag.Diagnostics) {
	workspaceOID := observe.OID{
		Type: observe.TypeWorkspace,
		ID:   folder.WorkspaceID,
	}

	if err := data.Set("workspace", workspaceOID.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", folder.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", folder.Config.IconURL); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", folder.Config.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", folder.OID().String()); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceFolderDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteFolder(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete action: %s", err.Error())
	}
	return diags
}
