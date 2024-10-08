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

func resourceBookmarkGroup() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("bookmark_group", "description"),
		CreateContext: resourceBookmarkGroupCreate,
		ReadContext:   resourceBookmarkGroupRead,
		UpdateContext: resourceBookmarkGroupUpdate,
		DeleteContext: resourceBookmarkGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"workspace": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace"),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("bookmark_group", "schema", "name"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("bookmark_group", "schema", "description"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("bookmark_group", "schema", "icon_url"),
			},
			"presentation": {
				Type: schema.TypeString,
				ValidateDiagFunc: validateStringInSlice([]string{
					string(gql.BookmarkGroupPresentationPeruserworkspace),
					string(gql.BookmarkGroupPresentationPercustomerworkspace),
				}, false),
				Default:     gql.BookmarkGroupPresentationPercustomerworkspace,
				Optional:    true,
				Description: descriptions.Get("bookmark_group", "schema", "presentation"),
			},
			"is_home": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("bookmark_group", "schema", "is_home"),
			},
		},
	}
}

func newBookmarkGroupConfig(data *schema.ResourceData) (input *gql.BookmarkGroupInput, diags diag.Diagnostics) {
	input = &gql.BookmarkGroupInput{
		Name:        stringPtr(data.Get("name").(string)),
		Description: stringPtr(data.Get("description").(string)),
		IsHome:      boolPtr(data.Get("is_home").(bool)),
	}

	p := gql.BookmarkGroupPresentation(data.Get("presentation").(string))
	input.Presentation = &p

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	return input, diags
}

func bookmarkGroupToResourceData(bg *gql.BookmarkGroup, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", bg.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", bg.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if bg.IconUrl != "" {
		if err := data.Set("icon_url", bg.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("is_home", bg.IsHome); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if diags.HasError() {
		return diags
	}

	if err := data.Set("oid", bg.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceBookmarkGroupCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newBookmarkGroupConfig(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateBookmarkGroup(ctx, id.Id, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create bookmark group",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceBookmarkGroupRead(ctx, data, meta)...)
}

func resourceBookmarkGroupRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetBookmarkGroup(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve bookmark group [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return bookmarkGroupToResourceData(result, data)
}

func resourceBookmarkGroupUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newBookmarkGroupConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateBookmarkGroup(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update bookmark group [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return bookmarkGroupToResourceData(result, data)
}

func resourceBookmarkGroupDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteBookmarkGroup(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete bookmark group: %s", err)
	}
	return diags
}
