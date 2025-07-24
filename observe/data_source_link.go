package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func dataSourceLink() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("link", "description"),
		ReadContext: dataSourceLinkRead,
		Schema: map[string]*schema.Schema{
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
			diags[i].Detail = fmt.Sprintf("link %s -> %s %q", source.Id, target.Id, fields)
		}
	}()

	srcFields, dstFields := unpackFields(fields)
	link, err := client.LookupForeignKey(ctx, source.Id, target.Id, srcFields, dstFields)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(source.Id + "/" + *link.Id)
	return nil
}
