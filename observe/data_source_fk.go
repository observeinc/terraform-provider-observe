package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func dataSourceForeignKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceForeignKeyRead,

		Schema: map[string]*schema.Schema{
			"source": {
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
		},
	}
}

func dataSourceForeignKeyRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		fields = data.Get("fields").([]interface{})
	)

	source, _ := observe.NewOID(data.Get("source").(string))
	target, _ := observe.NewOID(data.Get("target").(string))

	defer func() {
		// right now SDK does not report where this error happened,
		// so we need to provide a little extra context
		for i := range diags {
			diags[i].Detail = fmt.Sprintf("foreign key %s -> %s %q", source.ID, target.ID, fields)
		}
		return
	}()

	srcFields, dstFields := unpackFields(fields)
	fk, err := client.LookupForeignKey(ctx, source.ID, target.ID, srcFields, dstFields)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(*fk.Config.Source + "/" + *fk.Config.Label)
	return nil
}
