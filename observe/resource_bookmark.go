package observe

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceBookmark() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("bookmark", "description"),
		CreateContext: resourceBookmarkCreate,
		ReadContext:   resourceBookmarkRead,
		UpdateContext: resourceBookmarkUpdate,
		DeleteContext: resourceBookmarkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"group": {
				Type: schema.TypeString,
				// API currently restricts permissions on moving bookmark, so
				// just delete old bookmark, create new
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeBookmarkGroup),
				Description:      descriptions.Get("bookmark", "schema", "group"),
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("bookmark", "schema", "name"),
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"target": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      descriptions.Get("bookmark", "schema", "target"),
				ValidateDiagFunc: validateOID(oid.TypeDataset, oid.TypeDashboard),
				DiffSuppressFunc: diffSuppressOIDVersion,
			},
			"bookmark_kind": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      describeEnums(gql.AllBookmarkKindTypes, descriptions.Get("bookmark", "schema", "bookmark_kind")),
				ValidateDiagFunc: validateEnums(gql.AllBookmarkKindTypes),
				DiffSuppressFunc: diffSuppressEnums,
			},
			"entity_tags": {
				Type:             schema.TypeMap,
				Optional:         true,
				DiffSuppressFunc: diffSuppressEntityTagValues,
				Description:      descriptions.Get("common", "schema", "entity_tags"),
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func newBookmarkConfig(data *schema.ResourceData) (input *gql.BookmarkInput, diags diag.Diagnostics) {
	var (
		name         = data.Get("name").(string)
		groupOid, _  = oid.NewOID(data.Get("group").(string))
		targetOid, _ = oid.NewOID(data.Get("target").(string))
	)

	input = &gql.BookmarkInput{
		Name:     &name,
		TargetId: &targetOid.Id,
		GroupId:  &groupOid.Id,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("bookmark_kind"); ok {
		bookmarkKind := gql.BookmarkKind(toCamel(v.(string)))
		input.BookmarkKind = &bookmarkKind
	}

	// Always set EntityTags, even if empty, to allow clearing tags
	if v, ok := data.GetOk("entity_tags"); ok {
		input.EntityTags = expandEntityTagsFromMap(v.(map[string]interface{}))
	} else {
		input.EntityTags = []gql.EntityTagMappingInput{}
	}

	return input, diags
}

func bookmarkToResourceData(b *gql.Bookmark, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", b.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if b.IconUrl != "" {
		if err := data.Set("icon_url", b.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("group", oid.BookmarkGroupOid(b.GroupId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	targetOid := oid.OID{
		Id:   b.TargetId,
		Type: oid.Type(strings.ToLower(string(b.TargetIdKind))),
	}
	if err := data.Set("target", targetOid.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if diags.HasError() {
		return diags
	}

	if err := data.Set("oid", b.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("bookmark_kind", toSnake(string(b.GetBookmarkKind()))); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("entity_tags", flattenEntityTagsToMap(b.EntityTags)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceBookmarkCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newBookmarkConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateBookmark(ctx, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create bookmark",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceBookmarkRead(ctx, data, meta)...)
}

func resourceBookmarkRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetBookmark(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve bookmark [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return bookmarkToResourceData(result, data)
}

func resourceBookmarkUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	config, diags := newBookmarkConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateBookmark(ctx, data.Id(), config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update bookmark [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return bookmarkToResourceData(result, data)
}

func resourceBookmarkDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteBookmark(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete bookmark: %s", err)
	}
	return diags
}
