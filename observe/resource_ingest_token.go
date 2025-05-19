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

func resourceIngestToken() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("ingest_token", "description"),
		CreateContext: resourceIngestTokenCreate,
		ReadContext:   resourceIngestTokenRead,
		UpdateContext: resourceIngestTokenUpdate,
		DeleteContext: resourceIngestTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("ingest_token", "schema", "name"),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"datastream_ids": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: descriptions.Get("ingest_token", "schema", "datastream_ids"),
			},
			"secret": {
				Type:      schema.TypeString,
				Sensitive: true,
				Computed:  true,
			},
		},
	}
}

type IngestTokenResource struct {
	name          *string
	description   *string
	disabled      *bool
	datastreamIds []string
}

func newIngestTokenResource(data *schema.ResourceData) IngestTokenResource {
	var name *string
	if v, ok := data.GetOk("name"); ok {
		ingestTokenName := v.(string)
		name = &ingestTokenName
	}
	var description *string
	if v, ok := data.GetOk("description"); ok {
		ingestTokenDescription := v.(string)
		description = &ingestTokenDescription
	}
	var disabled *bool
	if v, ok := data.GetOk("disabled"); ok {
		ingestTokenDisabled := v.(bool)
		disabled = &ingestTokenDisabled
	}
	var datastreamIds []string
	if v, ok := data.GetOk("datastream_ids"); ok {
		for _, elem := range v.([]interface{}) {
			datastreamIds = append(datastreamIds, elem.(string))
		}
	}

	return IngestTokenResource{
		name:          name,
		description:   description,
		disabled:      disabled,
		datastreamIds: datastreamIds,
	}
}

func ingestTokenToResourceData(ingestToken *gql.IngestToken, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(ingestToken.Id).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("oid", oid.IngestTokenOid(ingestToken.Id).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("name", ingestToken.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("description", ingestToken.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("disabled", ingestToken.Disabled); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("secret", *ingestToken.Secret); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceIngestTokenCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	handleError := func(id string, detail string) diag.Diagnostics {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to create ingest token [id=%s]", id),
			Detail:   detail,
		})
		return diags
	}

	client := meta.(*observe.Client)
	workspace, _ := oid.NewOID(data.Get("workspace").(string))
	resource := newIngestTokenResource(data)
	if resource.disabled != nil && *resource.disabled {
		return handleError("?", "ingest token cannot be disabled on creation")
	}

	ingestToken, err := client.CreateIngestToken(ctx, workspace.Id, resource.name, resource.description)
	if err != nil {
		return handleError(ingestToken.Id, err.Error())
	}
	if ingestToken.Secret == nil {
		// This should never happen, we should always return an error if we were unable to generate
		// a secret. The only case we allow for secrets to be nil are password based legacy
		// authtokens, which should never apply here.
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to create ingest token [id=%s]", data.Id()),
			Detail:   "failed to create secret",
		})
		return diags
	}
	data.SetId(ingestToken.Id)

	err = client.UpdateIngestTokenAssocation(ctx, ingestToken.Id, resource.datastreamIds)
	if err != nil {
		return handleError(ingestToken.Id, err.Error())
	}

	return append(diags, resourceIngestTokenRead(ctx, data, meta)...)
}

func resourceIngestTokenRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	ingestToken, err := client.GetIngestToken(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve ingest token [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	// TODO Lookup `ingest_token_routing` table to find any datastreams associated with this ingest
	// token.

	return ingestTokenToResourceData(ingestToken, data)
}

func resourceIngestTokenUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config := newIngestTokenResource(data)
	ingestToken, err := client.UpdateIngestToken(ctx, data.Id(), gql.IngestTokenInput{
		Name:        config.name,
		Description: config.description,
		Disabled:    config.disabled,
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update ingest token [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	// TODO Update `ingest_token_routing` table if the set of datastreams associated with this
	// ingest token has changed.

	return ingestTokenToResourceData(ingestToken, data)
}

func resourceIngestTokenDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	err := client.DeleteIngestToken(ctx, data.Id())
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to delete ingest token [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}
	return diags
}
