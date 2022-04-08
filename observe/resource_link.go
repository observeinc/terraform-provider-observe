package observe

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

const (
	schemaLinkWorkspaceDescription = "OID of workspace link is contained in."
	schemaLinkSourceDescription    = "OID of source dataset."
	schemaLinkTargetDescription    = "OID of target dataset."
	schemaLinkFieldsDescription    = "Array of field mappings that provides a link between source and target datasets. " +
		"A mapping between a `source_field` and a `target_field` is represented using a colon separated \"<source_field>:<target_field>\" format. " +
		"If the source and target field share the same name, only \"<source_field>\"."
	schemaLinkLabelDescription = "Label describing link."
)

func resourceLink() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLinkCreate,
		ReadContext:   resourceLinkRead,
		UpdateContext: resourceLinkUpdate,
		DeleteContext: resourceLinkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
				Description:      schemaLinkWorkspaceDescription,
			},
			"source": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeDataset),
				Description:      schemaLinkSourceDescription,
			},
			"target": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeDataset),
				Description:      schemaLinkTargetDescription,
			},
			"fields": {
				Type:             schema.TypeList,
				Required:         true,
				Elem:             &schema.Schema{Type: schema.TypeString},
				DiffSuppressFunc: diffSuppressFields,
				Description:      schemaLinkFieldsDescription,
			},
			"label": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: schemaLinkLabelDescription,
			},
		},
	}
}

func newLinkConfig(data *schema.ResourceData) (config *observe.ForeignKeyConfig, diags diag.Diagnostics) {
	var (
		source, _ = observe.NewOID(data.Get("source").(string))
		target, _ = observe.NewOID(data.Get("target").(string))
		fields    = data.Get("fields").([]interface{})
	)

	config = &observe.ForeignKeyConfig{
		Source: &source.ID,
		Target: &target.ID,
	}

	if v, ok := data.GetOk("label"); ok {
		s := v.(string)
		config.Label = &s
	}

	config.SrcFields, config.DstFields = unpackFields(fields)
	return
}

func resourceLinkCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newLinkConfig(data)
	if diags.HasError() {
		return diags
	}

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateForeignKey(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create foreign key: %s", err.Error())
	}

	data.SetId(result.ID)
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
		return diag.Errorf("failed to update foreign key: %s", err.Error())
	}

	return append(diags, resourceLinkRead(ctx, data, meta)...)
}

func resourceLinkRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	link, err := client.GetForeignKey(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read foreign key: %s", err.Error())
	}

	var fields []string
	for i, src := range link.Config.SrcFields {
		dst := link.Config.DstFields[i]
		if src == dst {
			fields = append(fields, src)
		} else {
			fields = append(fields, src+":"+dst)
		}
	}

	workspaceOID := observe.OID{
		Type: observe.TypeWorkspace,
		ID:   link.Workspace,
	}

	if err := data.Set("workspace", workspaceOID.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// TODO: we may need to set source and target, but if we do we must pass
	// through version info in OID

	if err := data.Set("fields", fields); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("label", link.Config.Label); err != nil {
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
