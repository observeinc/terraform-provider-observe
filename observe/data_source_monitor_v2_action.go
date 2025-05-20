package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
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
				ValidateDiagFunc: validateID(),
				Description:      descriptions.Get("common", "schema", "id") + " One of either `id` or `name` must be provided.",
			},
			"workspace": { // ObjectId!
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				Description:      descriptions.Get("common", "schema", "workspace") + " Must be specified if looking up by name.",
			},
			"name": { // String!
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"name", "id"},
				RequiredWith: []string{"workspace"},
				Description:  descriptions.Get("monitor_v2_action", "schema", "name") + " One of either `id` or `name` must be provided.",
			},
			// fields of MonitorV2ActionInput
			"type": { // MonitorV2ActionType!
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_action", "schema", "type"),
			},
			"email": { // MonitorV2EmailDestinationInput
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        monitorV2EmailActionDatasource(),
				Description: descriptions.Get("monitor_v2_action", "schema", "email"),
			},
			"webhook": { // MonitorV2WebhookDestinationInput
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        monitorV2WebhookActionDatasource(),
				Description: descriptions.Get("monitor_v2_action", "schema", "webhook"),
			},
			"description": { // String
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("monitor_v2_action", "schema", "description"),
			},
			// end of monitorV2ActionInput
			"oid": { // ObjectId!
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
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
		if err != nil {
			return diag.FromErr(err)
		}
	} else if name != "" {
		workspaceID, _ := data.Get("workspace").(string)
		if workspaceID != "" {
			actions, err := client.SearchMonitorV2Action(ctx, &workspaceID, &name)
			if err != nil {
				return diag.FromErr(err)
			} else if len(actions) != 1 {
				return diag.Errorf("found %d monitor actions with name %q", len(actions), name)
			}
			act = &actions[0]
		}
	}

	if act == nil {
		return diag.Errorf("failed to lookup monitor action from provided get/search parameters")
	}

	data.SetId(act.Id)
	return resourceMonitorV2ActionRead(ctx, data, meta)
}
