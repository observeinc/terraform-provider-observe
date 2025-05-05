package observe

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceLink() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("link", "description"),
		CreateContext: resourceLinkCreate,
		ReadContext:   resourceLinkRead,
		UpdateContext: resourceLinkUpdate,
		DeleteContext: resourceLinkDelete,

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
			"source": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				Description:      descriptions.Get("link", "schema", "source"),
			},
			"target": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				Description:      descriptions.Get("link", "schema", "target"),
			},
			"fields": {
				Type:             schema.TypeList,
				Required:         true,
				Elem:             &schema.Schema{Type: schema.TypeString},
				DiffSuppressFunc: diffSuppressFields,
				Description:      descriptions.Get("link", "schema", "fields"),
			},
			"label": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("link", "schema", "label"),
			},
		},
	}
}

func newLinkConfig(data *schema.ResourceData) (input *gql.DeferredForeignKeyInput, diags diag.Diagnostics) {
	var (
		source, _ = oid.NewOID(data.Get("source").(string))
		target, _ = oid.NewOID(data.Get("target").(string))
		fields    = data.Get("fields").([]interface{})
	)

	input = &gql.DeferredForeignKeyInput{
		SourceDataset: &gql.DeferredDatasetReferenceInput{
			DatasetId: &source.Id,
		},
		TargetDataset: &gql.DeferredDatasetReferenceInput{
			DatasetId: &target.Id,
		},
	}

	if v, ok := data.GetOk("label"); ok {
		input.Label = stringPtr(v.(string))
	}

	input.SrcFields, input.DstFields = unpackFields(fields)
	return
}

func resourceLinkCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newLinkConfig(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateForeignKey(ctx, id.Id, config)
	if err != nil {
		return diag.Errorf("failed to create foreign key: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceLinkRead(ctx, data, meta)...)
}

func resourceLinkUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newLinkConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateForeignKey(ctx, data.Id(), config)
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			diags = resourceLinkCreate(ctx, data, meta)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.Errorf("failed to update foreign key: %s", err.Error())
	}

	return append(diags, resourceLinkRead(ctx, data, meta)...)
}

func resourceLinkRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	link, err := client.GetForeignKey(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		if link == nil {
			return diag.Errorf("failed to read link: %s", err.Error())
		}

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  err.Error(),
		})
	}

	var fields []string
	for i, src := range link.SrcFields {
		dst := link.DstFields[i]
		if src == dst {
			fields = append(fields, src)
		} else {
			fields = append(fields, src+":"+dst)
		}
	}

	if err := data.Set("workspace", oid.WorkspaceOid(link.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// TODO: we may need to set source and target, but if we do we must pass
	// through version info in OID

	if err := data.Set("fields", fields); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("label", link.Label); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", link.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceLinkDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteForeignKey(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete foreign key: %s", err.Error())
	}
	return diags
}

func unpackFields(fields []interface{}) (srcFields, dstFields []string) {
	for _, field := range fields {
		s := field.(string)
		if tuple := strings.SplitN(s, ":", 2); len(tuple) == 1 {
			srcFields = append(srcFields, s)
			dstFields = append(dstFields, s)
		} else {
			srcFields = append(srcFields, tuple[0])
			dstFields = append(dstFields, tuple[1])
		}
	}
	return
}

func diffSuppressFields(k, old, new string, d *schema.ResourceData) bool {
	if old == new {
		return true
	}

	/* fields accepts a pair of source / target column names,
	* e.g.: "id:fooId"
	* If both source and target column names are the same, the
	* target can be elided, therefore "id:id" and "id" are
	* equivalent.
	 */
	if tuple := strings.SplitN(new, ":", 2); len(tuple) == 2 {
		return tuple[0] == tuple[1] && tuple[0] == old
	}

	if tuple := strings.SplitN(old, ":", 2); len(tuple) == 2 {
		return tuple[0] == tuple[1] && tuple[0] == new
	}

	return false
}
