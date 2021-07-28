package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
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
			"workspace": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				ValidateDiagFunc: validateOID(observe.TypeWorkspace),
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
					ValidateDiagFunc: validateOID(observe.TypeMonitor),
				},
			},
		},
	}
}

func newChannelConfig(data *schema.ResourceData) (config *observe.ChannelConfig, diags diag.Diagnostics) {
	config = &observe.ChannelConfig{
		Name: data.Get("name").(string),
	}

	if v, ok := data.GetOk("icon_url"); ok {
		s := v.(string)
		config.IconURL = &s
	}

	if v, ok := data.GetOk("description"); ok {
		s := v.(string)
		config.Description = &s
	}

	if s, ok := data.GetOk("monitors"); ok {
		for _, v := range s.(*schema.Set).List() {
			oid, _ := observe.NewOID(v.(string))
			config.Monitors = append(config.Monitors, oid)
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

	oid, _ := observe.NewOID(data.Get("workspace").(string))
	result, err := client.CreateChannel(ctx, oid.ID, config)
	if err != nil {
		return diag.Errorf("failed to create channel: %s", err.Error())
	}

	data.SetId(result.ID)
	return append(diags, resourceChannelRead(ctx, data, meta)...)
}

func resourceChannelUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newChannelConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.UpdateChannel(ctx, data.Id(), config)
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

	workspaceOID := observe.OID{
		Type: observe.TypeWorkspace,
		ID:   channel.WorkspaceID,
	}

	if err := data.Set("workspace", workspaceOID.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", channel.Config.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("icon_url", channel.Config.IconURL); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", channel.Config.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	toIdList := func(l []*observe.OID) (ret []string) {
		for _, el := range l {
			ret = append(ret, el.String())
		}
		return
	}

	if err := data.Set("monitors", toIdList(channel.Config.Monitors)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("oid", channel.OID().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceChannelDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteChannel(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete action: %s", err.Error())
	}
	return diags
}
