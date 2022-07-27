package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
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
			"folder": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(observe.TypeFolder, observe.TypeWorkspace),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"source": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeDataset),
			},
			"step": &schema.Schema{
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"link": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateOID(observe.TypeLink),
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

func newPreferredPathConfig(data *schema.ResourceData) (config *observe.PreferredPathConfig, diags diag.Diagnostics) {
	var (
		folder, _ = observe.NewOID(data.Get("folder").(string))
		source, _ = observe.NewOID(data.Get("source").(string))
		steps     = data.Get("step").([]interface{})
	)

	config = &observe.PreferredPathConfig{
		Name:   data.Get("name").(string),
		Folder: folder,
		Source: source,
	}

	for _, el := range steps {
		step := el.(map[string]interface{})

		var link *observe.OID
		if v := step["link"]; v != nil {
			link, _ = observe.NewOID(v.(string))
		}

		config.Path = append(config.Path, observe.PreferredPathStep{
			Link:    link,
			Reverse: step["reverse"].(bool),
		})
	}

	return
}

func resourcePreferredPathCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newPreferredPathConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("folder").(string))
	result, err := client.CreatePreferredPath(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create preferred path: %s", err.Error())
	}

	data.SetId(result.ID)
	return append(diags, resourcePreferredPathRead(ctx, data, meta)...)
}

func resourcePreferredPathUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newPreferredPathConfig(data)
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

	if err := data.Set("folder", path.Config.Folder.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", path.Config.Name); err != nil {
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
