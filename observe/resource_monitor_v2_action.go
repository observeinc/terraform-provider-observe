package observe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourceMonitorV2Action() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMonitorV2Create,
		ReadContext:   resourceMonitorV2Read,
		UpdateContext: resourceMonitorV2Update,
		DeleteContext: resourceMonitorV2Delete,
		Schema: map[string]*schema.Schema{
			// needed as input to CreateMonitorV2Action
			"workspace_id": { // ObjectId!
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			// fields of MonitorV2ActionInput
			"inline": { // Boolean
				Type:     schema.TypeBool,
				Optional: true,
			},
			"type": { // MonitorV2ActionType!
				Type:             schema.TypeString,
				ValidateDiagFunc: validateEnums(gql.AllMonitorV2ActionTypes),
				Required:         true,
			},
			"email": { // MonitorV2EmailDestinationInput
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem:     monitorV2EmailActionInput(),
			},
			"webhook": { // MonitorV2WebhookDestinationInput
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem:     monitorV2WebhookActionInput(),
			},
			"name": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			// end of monitorV2ActionInput
			"id": { // ObjectId!
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func monitorV2EmailActionInput() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subject": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"body": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"fragments": { // JsonObject
				Type:             schema.TypeString,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Optional:         true,
			},
		},
	}
}

func monitorV2WebhookActionInput() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"headers": { // [MonitorV2WebhookHeaderInput!]
				Type:     schema.TypeList,
				Optional: true,
				Elem:     monitorV2WebhookHeaderInput(),
			},
			"body": { // String
				Type:     schema.TypeString,
				Optional: true,
			},
			"fragments": { // JsonObject
				Type:             schema.TypeString,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Optional:         true,
			},
		},
	}
}

func monitorV2WebhookHeaderInput() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"header": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
			"value": { // String!
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceMonitorV2ActionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2ActionInput(data)
	if diags.HasError() {
		return diags
	}

	workspaceID, _ := oid.NewOID(data.Get("workspace_id").(string))
	result, err := client.CreateMonitorV2Action(ctx, workspaceID.Id, input)
	if err != nil {
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceMonitorV2ActionRead(ctx, data, meta)...)
}

func resourceMonitorV2ActionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	input, diags := newMonitorV2ActionInput(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateMonitorV2Action(ctx, data.Id(), input)
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			diags = resourceMonitorV2ActionCreate(ctx, data, meta)
			if diags.HasError() {
				return diags
			}
			return nil
		}
		return diag.Errorf("failed to create monitor action: %s", err.Error())
	}

	return append(diags, resourceMonitorV2ActionRead(ctx, data, meta)...)
}

func resourceMonitorV2ActionDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteMonitorV2Action(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete monitor action: %s", err.Error())
	}
	return diags
}

func resourceMonitorV2ActionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	// TODO
	return diags
}

func newMonitorV2ActionInput(data *schema.ResourceData) (input *gql.MonitorV2ActionInput, diags diag.Diagnostics) {
	// required
	actionType := toCamel(data.Get("type").(string))
	name := data.Get("name").(string)

	// instantiation
	input = &gql.MonitorV2ActionInput{
		Type: meta.MonitorV2ActionType(actionType),
		Name: name,
	}

	// optionals
	if v, ok := data.GetOk("inline"); ok {
		boolVal := v.(bool)
		input.Inline = &boolVal
	}
	if _, ok := data.GetOk("email"); ok {
		email, diags := newMonitorV2EmailActionInput(data, "email.0.")
		if diags.HasError() {
			return nil, diags
		}
		input.Email = email
	}
	if _, ok := data.GetOk("webhook"); ok {
		webhook, diags := newMonitorV2WebhookActionInput(data, "webhook.0.")
		if diags.HasError() {
			return nil, diags
		}
		input.Webhook = webhook
	}
	if v, ok := data.GetOk("icon_url"); ok {
		iconURL := v.(string)
		input.IconUrl = &iconURL
	}
	if v, ok := data.GetOk("description"); ok {
		description := v.(string)
		input.Description = &description
	}

	return nil, diags
}

func newMonitorV2EmailActionInput(data *schema.ResourceData, path string) (email *gql.MonitorV2EmailActionInput, diags diag.Diagnostics) {
	// instantiation
	email = &gql.MonitorV2EmailActionInput{}

	// optionals
	if v, ok := data.GetOk(fmt.Sprintf("%ssubject", path)); ok {
		subject := v.(string)
		email.Subject = &subject
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sbody", path)); ok {
		body := v.(string)
		email.Body = &body
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		email.Fragments = types.JsonObject(v.(string)).Ptr()
	}

	return email, diags
}

func newMonitorV2WebhookActionInput(data *schema.ResourceData, path string) (webhook *gql.MonitorV2WebhookActionInput, diags diag.Diagnostics) {
	// instantiation
	webhook = &gql.MonitorV2WebhookActionInput{}

	// optionals
	if v, ok := data.GetOk(fmt.Sprintf("%sbody", path)); ok {
		body := v.(string)
		webhook.Body = &body
	}
	if _, ok := data.GetOk(fmt.Sprintf("%sheaders", path)); ok {
		webhook.Headers = make([]gql.MonitorV2WebhookHeaderInput, 0)
		for i := range data.Get(fmt.Sprintf("%sheaders", path)).([]interface{}) {
			header, diags := newMonitorV2WebhookHeaderInput(data, fmt.Sprintf("%sheaders.%d.", path, i))
			if diags.HasError() {
				return nil, diags
			}
			webhook.Headers = append(webhook.Headers, *header)
		}
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		fragmentsStr := v.(string)
		err := json.Unmarshal([]byte(fragmentsStr), webhook.Fragments)
		if err != nil {
			return nil, diag.Errorf(err.Error())
		}
	}
	if v, ok := data.GetOk(fmt.Sprintf("%sfragments", path)); ok {
		webhook.Fragments = types.JsonObject(v.(string)).Ptr()
	}

	return webhook, diags
}

func newMonitorV2WebhookHeaderInput(data *schema.ResourceData, path string) (header *gql.MonitorV2WebhookHeaderInput, diags diag.Diagnostics) {
	// required
	headerStr := data.Get(fmt.Sprintf("%sheader", path)).(string)
	valueStr := data.Get(fmt.Sprintf("%svalue", path)).(string)

	// instantiation
	header = &gql.MonitorV2WebhookHeaderInput{
		Header: headerStr,
		Value:  valueStr,
	}

	return header, diags
}
