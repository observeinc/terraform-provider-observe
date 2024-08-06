package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func dataSourceMonitorV2Action() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMonitorV2ActionRead,
		Schema: map[string]*schema.Schema{
			// used to lookup the action
			"id": { // ObjectId!
				Type:             schema.TypeString,
				Optional:         true,
				ExactlyOneOf:     []string{"name", "id"},
				ValidateDiagFunc: validateOID(oid.TypeMonitorV2Action),
			},
			"workspace": { // ObjectId!
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"name": { // String!
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"name", "id"},
				Optional:     true,
			},
			// fields of MonitorV2ActionInput
			"type": { // MonitorV2ActionType!
				Type:     schema.TypeString,
				Computed: true,
			},
			"email": { // MonitorV2EmailDestinationInput
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     monitorV2EmailActionDatasource(),
			},
			"webhook": { // MonitorV2WebhookDestinationInput
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     monitorV2WebhookActionDatasource(),
			},
			"description": { // String
				Type:     schema.TypeString,
				Computed: true,
			},
			// end of monitorV2ActionInput
			"oid": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
			"destination": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func monitorV2EmailActionDatasource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subject": { // String
				Type:     schema.TypeString,
				Computed: true,
			},
			"body": { // String
				Type:     schema.TypeString,
				Computed: true,
			},
			"fragments": { // JsonObject
				Type:     schema.TypeString,
				Computed: true,
			},
			"users": { // [UserId!]
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"addresses": { // [String!]
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func monitorV2WebhookActionDatasource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"headers": { // [MonitorV2WebhookHeaderInput!]
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     monitorV2WebhookHeaderDatasource(),
			},
			"body": { // String
				Type:     schema.TypeString,
				Computed: true,
			},
			"fragments": { // JsonObject
				Type:     schema.TypeString,
				Computed: true,
			},
			"url": { // String!
				Type:     schema.TypeString,
				Computed: true,
			},
			"method": { // MonitorV2HttpType!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func monitorV2WebhookHeaderDatasource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"header": { // String!
				Type:     schema.TypeString,
				Computed: true,
			},
			"value": { // String!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceMonitorV2ActionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client = meta.(*observe.Client)
		name   = data.Get("name").(string)
		getID  = data.Get("id").(string)
	)

	var act *gql.MonitorV2Action
	var err error

	if getID != "" {
		act, err = client.GetMonitorV2Action(ctx, getID)
	} else if name != "" {
		workspaceID, _ := data.Get("workspace").(string)
		if workspaceID != "" {
			act, err = client.SearchMonitorV2Action(ctx, &workspaceID, &name)
		}
	}

	if err != nil {
		diags = diag.FromErr(err)
		return
	} else if act == nil {
		return diag.Errorf("failed to lookup monitor action from provided get/search parameters")
	}

	data.SetId(act.Id)
	return resourceMonitorV2ActionRead(ctx, data, meta)
}
