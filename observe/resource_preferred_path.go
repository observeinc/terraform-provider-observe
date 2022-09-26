package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourcePreferredPath() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePreferredPathCreate,
		ReadContext:   resourcePreferredPathRead,
		UpdateContext: resourcePreferredPathUpdate,
		DeleteContext: resourcePreferredPathDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				ExactlyOneOf:     []string{"folder", "workspace"},
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"folder": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ExactlyOneOf:     []string{"folder", "workspace"},
				ValidateDiagFunc: validateOID(oid.TypeFolder),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"source": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"step": {
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"link": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateOID(oid.TypeLink),
						},
						"reverse": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
		},
	}
}

func newPreferredPathConfig(data *schema.ResourceData) (input *gql.PreferredPathInput, wsid string, diags diag.Diagnostics) {
	var (
		name        = data.Get("name").(string)
		source, _   = oid.NewOID(data.Get("source").(string))
		description = data.Get("description").(string)
		steps       = data.Get("step").([]interface{})
	)

	input = &gql.PreferredPathInput{
		Name:          &name,
		SourceDataset: &source.Id,
		Description:   &description,
	}

	if folder, ok := data.GetOk("folder"); ok {
		id, _ := oid.NewOID(folder.(string))
		// Folder ID is stored in the Version field, workspace ID is stored in the Id field
		input.FolderId = id.Version
		wsid = id.Id
	} else {
		workspace := data.Get("workspace").(string)
		id, _ := oid.NewOID(workspace)
		wsid = id.Id
	}

	for _, el := range steps {
		step := el.(map[string]interface{})

		var link *oid.OID
		if v := step["link"]; v != nil {
			link, _ = oid.NewOID(v.(string))
		}

		reverse := step["reverse"].(bool)
		input.Path = append(input.Path, gql.PreferredPathStepInput{
			LinkId:  &link.Id,
			Reverse: &reverse,
		})
	}

	return
}

func resourcePreferredPathCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, wsid, diags := newPreferredPathConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreatePreferredPath(ctx, wsid, config)
	if err != nil {
		return diag.Errorf("failed to create preferred path: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourcePreferredPathRead(ctx, data, meta)...)
}

func resourcePreferredPathUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, _, diags := newPreferredPathConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdatePreferredPath(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update preferred path: %s", err.Error())
	}

	return append(diags, resourcePreferredPathRead(ctx, data, meta)...)
}

func resourcePreferredPathRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	path, err := client.GetPreferredPath(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read preferred path: %s", err.Error())
	}

	if err := data.Set("folder", oid.FolderOid(path.FolderId, path.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", path.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", path.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourcePreferredPathDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeletePreferredPath(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete preferred path: %s", err.Error())
	}
	return diags
}
