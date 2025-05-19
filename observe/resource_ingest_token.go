package observe

import (
	"context"
	"fmt"
	"os"

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
			StateContext: resourceIngestTokenImport,
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
				Computed:    true,
				Description: descriptions.Get("ingest_token", "schema", "name"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("ingest_token", "schema", "description"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("ingest_token", "schema", "disabled"),
			},
			"secret": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
				Description: descriptions.Get("ingest_token", "schema", "secret"),
			},
		},
	}
}

func ingestTokenInputFromResource(data *schema.ResourceData) gql.IngestTokenInput {
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

	return gql.IngestTokenInput{
		Name:        name,
		Description: description,
		Disabled:    disabled,
	}
}

func ingestTokenToResourceData(ingestToken *gql.IngestToken, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(ingestToken.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("oid", oid.IngestTokenOid(ingestToken.Id).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("name", ingestToken.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if ingestToken.Description != nil {
		if err := data.Set("description", ingestToken.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	if ingestToken.Disabled != nil {
		if err := data.Set("disabled", ingestToken.Disabled); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
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
	workspaceId := data.Get("workspace").(string)
	ingestTokenInput := ingestTokenInputFromResource(data)
	if ingestTokenInput.Disabled != nil && *ingestTokenInput.Disabled {
		return handleError("?", "ingest token cannot be disabled on creation")
	}

	ingestToken, err := client.CreateIngestToken(ctx, workspaceId, ingestTokenInput)
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
	// Only populated on create response
	if err := data.Set("secret", ingestToken.Secret); err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
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

	return ingestTokenToResourceData(ingestToken, data)
}

func resourceIngestTokenUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	ingestTokenInput := ingestTokenInputFromResource(data)
	ingestToken, err := client.UpdateIngestToken(ctx, data.Id(), ingestTokenInput)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update ingest token [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

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

func resourceIngestTokenImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if secret := os.Getenv("SECRET"); secret != "" {
		if err := data.Set("secret", secret); err != nil {
			return nil, err
		}
	}

	return []*schema.ResourceData{data}, nil
}
