package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceLink() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLinkRead,

		Schema: map[string]*schema.Schema{
			"source": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
				Description:      schemaLinkSourceDescription,
			},
			"target": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDataset),
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

func dataSourceLinkRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		fields = data.Get("fields").([]interface{})
	)

	source, _ := oid.NewOID(data.Get("source").(string))
	target, _ := oid.NewOID(data.Get("target").(string))

	defer func() {
		// right now SDK does not report where this error happened,
		// so we need to provide a little extra context
		for i := range diags {
			diags[i].Detail = fmt.Sprintf("foreign key %s -> %s %q", source.Id, target.Id, fields)
		}
	}()

	srcFields, dstFields := unpackFields(fields)
	link, err := client.LookupForeignKey(ctx, source.Id, target.Id, srcFields, dstFields)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(source.Id + "/" + *link.Label)
	return nil
}
