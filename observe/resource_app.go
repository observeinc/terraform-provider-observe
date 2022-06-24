package observe

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
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
			"folder": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeFolder),
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
			"outputs": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func newAppInput(data *schema.ResourceData) (config *gql.AppInput, diags diag.Diagnostics) {
	folder, _ := oid.NewOID(data.Get("folder").(string))

	variables := make([]gql.AppVariableInput, 0)
	for k, v := range makeStringMap(data.Get("variables").(map[string]interface{})) {
		variable := gql.AppVariableInput{
			Name:  k,
			Value: v,
		}
		variables = append(variables, variable)
	}

	config = &gql.AppInput{
		ModuleId:  data.Get("module_id").(string),
		Version:   data.Get("version").(string),
		FolderId:  folder.Version,
		Variables: variables,
	}

	return
}

func resourceAppCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newAppInput(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("folder").(string))
	result, err := client.CreateApp(ctx, id.Id, input)
	if err != nil {
		return diag.Errorf("failed to create app: %s", err.Error())
	}

	data.SetId(result.Id)
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

	config, diags := newAppInput(data)
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

func appToResourceData(app *gql.App, data *schema.ResourceData) (diags diag.Diagnostics) {
	folderId := oid.OID{
		Type:    oid.TypeFolder,
		Id:      app.WorkspaceId,
		Version: &app.FolderId,
	}
	if err := data.Set("folder", folderId.String()); err != nil {
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

	if err := data.Set("oid", app.Oid().String()); err != nil {
		return diag.FromErr(err)
	}

	if app.Outputs != nil {
		out, err := json.Marshal(app.Outputs)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := data.Set("outputs", string(out)); err != nil {
			return diag.FromErr(err)
		}
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
