package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

func newQuery(data *schema.ResourceData) (query *observe.Query, diags diag.Diagnostics) {
	query = &observe.Query{
		Inputs: make(map[string]*observe.Input),
	}

	for k, v := range data.Get("inputs").(map[string]interface{}) {
		oid, _ := observe.NewOID(v.(string))
		query.Inputs[k] = &observe.Input{
			Dataset: &oid.ID,
		}
	}

	for i := range data.Get("stage").([]interface{}) {
		var stage observe.Stage

		if v, ok := data.GetOk(fmt.Sprintf("stage.%d.alias", i)); ok {
			s := v.(string)
			stage.Alias = &s
		}

		if v, ok := data.GetOk(fmt.Sprintf("stage.%d.input", i)); ok {
			s := v.(string)
			stage.Input = &s
		}

		if v, ok := data.GetOk(fmt.Sprintf("stage.%d.pipeline", i)); ok {
			stage.Pipeline = v.(string)
		}
		query.Stages = append(query.Stages, &stage)
	}

	return query, diags
}
