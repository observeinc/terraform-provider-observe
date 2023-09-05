package observe

import (
	"context"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
	"time"
)

func resourceFiledrop() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("filedrop", "description"),
		CreateContext: resourceFiledropCreate,
		ReadContext:   resourceFiledropRead,
		UpdateContext: resourceFiledropUpdate,
		DeleteContext: resourceFiledropDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("filedrop", "schema", "name"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("filedrop", "schema", "description"),
			},
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("filedrop", "schema", "status"),
			},
			"datastream": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDatastream),
				Description:      descriptions.Get("filedrop", "schema", "datastream"),
			},
			"config": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"format": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:             schema.TypeString,
										Required:         true,
										Description:      describeEnums(gql.AllFiledropFormatTypes, descriptions.Get("filedrop", "schema", "config", "format", "type")),
										ValidateDiagFunc: validateEnums(gql.AllFiledropFormatTypes),
										Deprecated:       "Filedrop now accepts all formats. Setting this field will have no effect",
									},
								},
							},
						},
						"provider": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"aws": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"region": {
													Type:        schema.TypeString,
													Required:    true,
													Description: descriptions.Get("filedrop", "schema", "config", "provider", "aws", "region"),
												},
												"role_arn": {
													Type:        schema.TypeString,
													Required:    true,
													Description: descriptions.Get("filedrop", "schema", "config", "provider", "aws", "role_arn"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"endpoint": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"s3": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"arn": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("filedrop", "schema", "endpoint", "s3", "arn"),
									},
									"bucket": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("filedrop", "schema", "endpoint", "s3", "bucket"),
									},
									"prefix": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: descriptions.Get("filedrop", "schema", "endpoint", "s3", "prefix"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newFiledropConfig(data *schema.ResourceData) (input *gql.FiledropInput, diags diag.Diagnostics) {
	input = &gql.FiledropInput{
		Config: expandFiledropConfig(data.Get("config.0").(map[string]interface{})),
	}

	if v, ok := data.GetOk("name"); ok {
		input.Name = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	return
}

func expandFiledropConfig(data map[string]interface{}) gql.FiledropConfigInput {
	provider := data["provider"].([]interface{})[0].(map[string]interface{})
	aws := provider["aws"].([]interface{})[0].(map[string]interface{})

	config := gql.FiledropConfigInput{
		ProviderAws: &gql.FiledropProviderAwsConfigInput{
			Region:  aws["region"].(string),
			RoleArn: aws["role_arn"].(string),
		},
	}

	return config
}

func resourceFiledropCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newFiledropConfig(data)

	if diags.HasError() {
		return diags
	}

	id, err := oid.NewOID(data.Get("workspace").(string))
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "failed to parse filedrop workspace ID",
				Detail:        err.Error(),
				AttributePath: cty.Path{cty.GetAttrStep{Name: "workspace"}},
			},
		}
	}

	datastreamId, err := oid.NewOID(data.Get("datastream").(string))
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "failed to parse filedrop datastream ID",
				Detail:        err.Error(),
				AttributePath: cty.Path{cty.GetAttrStep{Name: "datastream"}},
			},
		}
	}

	result, err := client.CreateFiledrop(ctx, id.Id, datastreamId.Id, config)

	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "failed to create filedrop",
				Detail:   err.Error(),
			},
		}
	}

	data.SetId(result.GetId())
	if result.Status != gql.FiledropStatusRunning {
		if diags := filedropWait(ctx, data, client); diags.HasError() {
			return diags
		}
	}
	return append(diags, resourceFiledropRead(ctx, data, meta)...)
}

func filedropWait(ctx context.Context, data *schema.ResourceData, client *observe.Client) (diags diag.Diagnostics) {
	createStateConf := &retry.StateChangeConf{
		Pending: []string{
			string(gql.FiledropStatusInitializing),
			string(gql.FiledropStatusUpdating),
		},
		Target: []string{
			string(gql.FiledropStatusRunning),
			string(gql.FiledropStatusDisabled),
		},
		Refresh: func() (any, string, error) {
			resp, err := client.GetFiledrop(ctx, data.Id())
			if err != nil {
				return 0, "", err
			}
			return resp, string(resp.Status), nil
		},
		Timeout:    data.Timeout(schema.TimeoutCreate) - time.Minute,
		Delay:      3 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err := createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("Error waiting for filedrop (%s) to be in final state: %s", data.Id(), err.Error())
	}
	return nil
}

func resourceFiledropRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	filedrop, err := client.GetFiledrop(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read filedrop: %s", err.Error())
	}

	if err := data.Set("workspace", oid.WorkspaceOid(filedrop.GetWorkspaceId()).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", filedrop.GetName()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", filedrop.GetIconUrl()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", filedrop.GetDescription()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("status", toSnake(string(filedrop.GetStatus()))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("datastream", oid.DatastreamOid(filedrop.GetDatastreamID()).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("endpoint", flattenFiledropEndpoint(filedrop.Endpoint)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("config", flattenFiledropConfig(filedrop.Config)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", filedrop.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func flattenFiledropConfig(config gql.FiledropConfig) interface{} {
	data := map[string]interface{}{}
	if awsConfig, ok := config.Provider.(*gql.FiledropConfigProviderFiledropProviderAwsConfig); ok {
		aws := map[string]interface{}{
			"role_arn": awsConfig.RoleArn,
			"region":   awsConfig.Region,
		}
		provider := map[string]interface{}{
			"aws": []interface{}{aws},
		}
		data["provider"] = []interface{}{provider}
	}

	return []interface{}{data}
}

func flattenFiledropEndpoint(endpoint gql.FiledropEndpoint) interface{} {
	var data map[string]interface{}
	if s3Endpoint, ok := endpoint.(*gql.FiledropEndpointFiledropS3Endpoint); ok {
		s3 := map[string]interface{}{
			"arn":    s3Endpoint.Arn,
			"prefix": s3Endpoint.Prefix,
			"bucket": s3Endpoint.Bucket,
		}
		data = map[string]interface{}{
			"s3": []interface{}{s3},
		}
	}

	return []interface{}{data}
}

func resourceFiledropUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newFiledropConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateFiledrop(ctx, data.Id(), config)
	if err != nil {
		return diag.Errorf("failed to update filedrop: %s", err.Error())
	}

	if result.Status != gql.FiledropStatusRunning {
		if diags := filedropWait(ctx, data, client); diags.HasError() {
			return diags
		}
	}

	return append(diags, resourceFiledropRead(ctx, data, meta)...)
}

func resourceFiledropDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteFiledrop(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete filedrop: %s", err.Error())
	}
	return diags
}
