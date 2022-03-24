package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceApp() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAppCreate,
		UpdateContext: resourceAppUpdate,
		ReadContext:   resourceAppRead,
		DeleteContext: resourceAppDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"folder": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeFolder),
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"module_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"variables": {
				Type:             schema.TypeMap,
				Optional:         true,
				ValidateDiagFunc: validateMapValues(validateIsString()),
			},
		},
	}
}

func newAppConfig(data *schema.ResourceData) (config *observe.AppConfig, diags diag.Diagnostics) {

	folder, _ := observe.NewOID(data.Get("folder").(string))

	config = &observe.AppConfig{
		ModuleId:  data.Get("module_id").(string),
		Version:   data.Get("version").(string),
		Folder:    folder,
		Variables: makeStringMap(data.Get("variables").(map[string]interface{})),
	}

	return
}

func resourceAppCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newAppConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("folder").(string))
	result, err := client.CreateApp(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create app: %s", err.Error())
	}

	data.SetId(result.ID)
	diags = append(diags, resourceAppRead(ctx, data, meta)...)
	if diags.HasError() {
		return diags
	}

	if result.Status.State != "Installed" {
		return diag.Errorf("failed to install app")
	}
	return nil
}

func resourceAppUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newAppConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateApp(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update app: %s", err.Error())
	}

	return append(diags, resourceAppRead(ctx, data, meta)...)
}

func resourceAppRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	app, err := client.GetApp(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read app: %s", err.Error())
	}

	return appToResourceData(app, data)
}

func appToResourceData(app *observe.App, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("folder", app.Config.Folder.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	/*
		if err := data.Set("name", app.Name); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	*/

	if err := data.Set("module_id", app.Config.ModuleId); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("version", app.Config.Version); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", app.OID().String()); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceAppDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteApp(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete app: %s", err.Error())
	}
	return diags
}
