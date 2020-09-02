package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
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
			"group": &schema.Schema{
				Type: schema.TypeString,
				// API currently restricts permissions on moving bookmark, so
				// just delete old bookmark, create new
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeBookmarkGroup),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"target": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeDataset),
				DiffSuppressFunc: diffSuppressOIDVersion,
			},
		},
	}
}

func newBookmarkConfig(data *schema.ResourceData) (config *observe.BookmarkConfig, diags diag.Diagnostics) {

	var (
		groupOid, _  = observe.NewOID(data.Get("group").(string))
		targetOid, _ = observe.NewOID(data.Get("target").(string))
	)

	config = &observe.BookmarkConfig{
		Name:     data.Get("name").(string),
		TargetID: targetOid.ID,
		GroupID:  groupOid.ID,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		icon := v.(string)
		config.IconURL = &icon
	}

	return config, diags
}

func bookmarkToResourceData(b *observe.Bookmark, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("name", b.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if b.Config.IconURL != nil {
		if err := data.Set("icon_url", b.Config.IconURL); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("group", b.GroupOID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("target", b.TargetOID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if diags.HasError() {
		return diags
	}

	if err := data.Set("oid", b.OID().String()); err != nil {
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

	result, err := client.CreateBookmark(config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create bookmark",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.ID)
	return append(diags, resourceBookmarkRead(ctx, data, meta)...)
}

func resourceBookmarkRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetBookmark(data.Id())
	if err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve dataset [id=%s]", data.Id()),
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

	result, err := client.UpdateBookmark(data.Id(), config)
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
	if err := client.DeleteBookmark(data.Id()); err != nil {
		return diag.Errorf("failed to delete dataset: %s", err)
	}
	return diags
}
