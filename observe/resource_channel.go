package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourceChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelCreate,
		ReadContext:   resourceChannelRead,
		UpdateContext: resourceChannelUpdate,
		DeleteContext: resourceChannelDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"monitors": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateOID(oid.TypeMonitor),
				},
			},
		},
	}
}

func newChannelConfig(data *schema.ResourceData) (input *gql.ChannelInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	input = &gql.ChannelInput{
		Name: &name,
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	return
}

func monitors(data *schema.ResourceData) (monitors []string, diags diag.Diagnostics) {
	if s, ok := data.GetOk("monitors"); ok {
		for _, v := range s.(*schema.Set).List() {
			id, _ := oid.NewOID(v.(string))
			monitors = append(monitors, id.Id)
		}
	}
	return
}

func resourceChannelCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newChannelConfig(data)
	if diags.HasError() {
		return diags
	}
	monitors, diags := monitors(data)
	if diags.HasError() {
		return diags
	}

	id, _ := oid.NewOID(data.Get("workspace").(string))
	result, err := client.CreateChannel(ctx, id.Id, config, monitors)
	if err != nil {
		return diag.Errorf("failed to create channel: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceChannelRead(ctx, data, meta)...)
}

func resourceChannelUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newChannelConfig(data)
	if diags.HasError() {
		return diags
	}
	monitors, diags := monitors(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateChannel(ctx, data.Id(), config, monitors)
	if err != nil {
		return diag.Errorf("failed to update channel: %s", err.Error())
	}

	return append(diags, resourceChannelRead(ctx, data, meta)...)
}

func resourceChannelRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	channel, err := client.GetChannel(ctx, data.Id())
	if err != nil {
		return diag.Errorf("failed to read channel: %s", err.Error())
	}

	if err := data.Set("workspace", oid.WorkspaceOid(channel.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", channel.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", channel.IconUrl); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", channel.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	monitors := make([]string, 0)
	for _, monitor := range channel.Monitors {
		monitors = append(monitors, oid.MonitorOid(monitor.Id).String())
	}
	if err := data.Set("monitors", monitors); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", channel.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceChannelDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteChannel(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete channel: %s", err.Error())
	}
	return diags
}
