package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
)

func resourceDashboardLink() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a link between two dashboards, optionally for a specific card. This feature is in development and dashboard links are not yet displayed in the UI.",
		CreateContext: resourceDashboardLinkCreate,
		ReadContext:   resourceDashboardLinkRead,
		UpdateContext: resourceDashboardLinkUpdate,
		DeleteContext: resourceDashboardLinkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ExactlyOneOf:     []string{"folder", "workspace"},
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
			},
			"folder": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ExactlyOneOf:     []string{"folder", "workspace"},
				ValidateDiagFunc: validateOID(oid.TypeFolder),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"from_dashboard": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDashboard),
			},
			"to_dashboard": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeDashboard),
			},
			"from_card": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"link_label": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func newDashboardLinkConfig(data *schema.ResourceData) (input *gql.DashboardLinkInput, diags diag.Diagnostics) {
	var (
		name             = data.Get("name").(string)
		fromDashboard, _ = oid.NewOID(data.Get("from_dashboard").(string))
		fromCard         = maybeString(data.GetOk("from_card"))
		toDashboard, _   = oid.NewOID(data.Get("to_dashboard").(string))
		description      = maybeString(data.GetOk("description"))
		workspace        = maybeOID(data.GetOk("workspace"))
		folder           = maybeOID(data.GetOk("folder"))
		managedBy        = maybeOID(data.GetOk("managed_by"))
		linkLabel        = data.Get("link_label").(string)
	)

	input = &gql.DashboardLinkInput{
		Name:            &name,
		FromDashboardId: fromDashboard.Id,
		ToDashboardId:   toDashboard.Id,
		Description:     &description,
		LinkLabel:       linkLabel,
	}
	if fromCard != "" {
		input.FromCard = &fromCard
	}
	if managedBy != nil {
		input.ManagedById = &managedBy.Id
	}
	if folder != nil {
		if folder.Version == nil || *folder.Version == "" {
			diags = append(diags, diag.Errorf("folder %q must have an id", folder.Id)...)
		} else {
			input.FolderId = folder.Version
		}
		//	The Folder specification is scary; the Id is a workspace Id, and
		//	the Version is the folder Id.
		input.WorkspaceId = folder.Id
	}
	if workspace != nil {
		input.WorkspaceId = workspace.Id
	}

	return
}

func resourceDashboardLinkCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newDashboardLinkConfig(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.Meta.CreateDashboardLink(ctx, *config)
	if err != nil {
		return diag.Errorf("failed to create dashboard link: %s", err.Error())
	}

	data.SetId(result.Id)
	return append(diags, resourceDashboardLinkRead(ctx, data, meta)...)
}

func resourceDashboardLinkUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	config, diags := newDashboardLinkConfig(data)
	if diags.HasError() {
		return diags
	}

	_, err := client.Meta.UpdateDashboardLink(ctx, data.Id(), *config)
	if err != nil {
		return diag.Errorf("failed to update dashboard link: %s", err.Error())
	}

	return append(diags, resourceDashboardLinkRead(ctx, data, meta)...)
}

func resourceDashboardLinkRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	//	This updates data that were already filled in by create/update, so it's
	//	mainly used to update defaults / computed values.
	client := meta.(*observe.Client)

	link, err := client.Meta.GetDashboardLink(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to read dashboard link: %s", err.Error())
	}

	if err := data.Set("folder", oid.FolderOid(link.FolderId, link.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("workspace", oid.WorkspaceOid(link.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", link.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("description", link.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceDashboardLinkDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.Meta.DeleteDashboardLink(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete dashboard link: %s", err.Error())
	}
	return diags
}
