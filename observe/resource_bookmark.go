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
)

func resourceBookmark() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBookmarkCreate,
		ReadContext:   resourceBookmarkRead,
		UpdateContext: resourceBookmarkUpdate,
		DeleteContext: resourceBookmarkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"group": {
				Type: schema.TypeString,
				// API currently restricts permissions on moving bookmark, so
				// just delete old bookmark, create new
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeBookmarkGroup),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"target": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				DiffSuppressFunc: diffSuppressOIDVersion,
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
