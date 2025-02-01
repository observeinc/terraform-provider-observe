package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceAppDataSource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAppDataSourceCreate,
		UpdateContext: resourceAppDataSourceUpdate,
		ReadContext:   resourceAppDataSourceRead,
		DeleteContext: resourceAppDataSourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: descriptions.Get("app_datasource", "description"),
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("app_datasource", "schema", "name"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("app_datasource", "schema", "description"),
			},
			"source_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("app_datasource", "schema", "source_url"),
			},
			//TODO: api requires it currently. But this should be optionally inferred via the app than a mandatory input.
			"instructions": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("app_datasource", "schema", "instructions"),
			},
			"app": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeApp),
				Description:      descriptions.Get("app_datasource", "schema", "app"),
			},
			"variables": {
				Type:             schema.TypeMap,
				Required:         true,
				ValidateDiagFunc: validateMapValues(validateIsString()),
				Description:      descriptions.Get("app_datasource", "schema", "variables"),
			},
		},
	}
}

func newAppDataSourceInput(data *schema.ResourceData) (config *gql.AppDataSourceInput, diags diag.Diagnostics) {
	app, _ := oid.NewOID(data.Get("app").(string))

	variables := makeAppVariableInputs(data.Get("variables").(map[string]interface{}))
	config = &gql.AppDataSourceInput{
		Name:         data.Get("name").(string),
		SourceUrl:    data.Get("source_url").(string),
		Instructions: data.Get("instructions").(string),
		AppId:        app.Id,
		Variables:    variables,
	}
	if v, ok := data.GetOk("description"); ok {
		config.Description = stringPtr(v.(string))
	}
	return
}

func resourceAppDataSourceCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newAppDataSourceInput(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateAppDataSource(ctx, input)
	if err != nil {
		return diag.Errorf("failed to create appdatasource: %s", err.Error())
	}

	data.SetId(result.Id)
	diags = append(diags, resourceAppDataSourceRead(ctx, data, meta)...)
	if diags.HasError() {
		return diags
	}
	return nil
}

func resourceAppDataSourceUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newAppDataSourceInput(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateAppDataSource(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update appdatasource: %s", err.Error())
	}

	return append(diags, resourceAppDataSourceRead(ctx, data, meta)...)
}

func resourceAppDataSourceRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	appdatasource, err := client.GetAppDataSource(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read appdatasource: %s", err.Error())
	}

	return appDataSourceToResourceData(appdatasource, data)
}

func appDataSourceToResourceData(appdatasource *gql.AppDataSource, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("oid", appdatasource.Oid().String()); err != nil {
		return diag.FromErr(err)
	}

	if err := data.Set("name", appdatasource.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if appdatasource.Description != nil {
		if err := data.Set("description", *appdatasource.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("source_url", appdatasource.SourceUrl); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("instructions", appdatasource.Instructions); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if appdatasource.ManagedById != nil {
		appOID := oid.AppOid(*appdatasource.ManagedById)
		if err := data.Set("app", appOID.String()); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	//TODO: variables?

	return diags
}

func resourceAppDataSourceDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteAppDataSource(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete appdatasource: %s", err.Error())
	}
	return diags
}
