package observe

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

var (
	_ resource.Resource                = &bookmarkResource{}
	_ resource.ResourceWithImportState = &bookmarkResource{}
	_ resource.ResourceWithConfigure   = &bookmarkResource{}
)

type bookmarkResource struct {
	client *observe.Client
}

type bookmarkResourceModel struct {
	ID           types.String `tfsdk:"id"`
	OID          types.String `tfsdk:"oid"`
	Group        types.String `tfsdk:"group"`
	Name         types.String `tfsdk:"name"`
	IconURL      types.String `tfsdk:"icon_url"`
	Target       types.String `tfsdk:"target"`
	BookmarkKind types.String `tfsdk:"bookmark_kind"`
	EntityTags   types.Map    `tfsdk:"entity_tags"`
}

func NewBookmarkResource() resource.Resource {
	return &bookmarkResource{}
}

func (r *bookmarkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bookmark"
}

func (r *bookmarkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptions.Get("bookmark", "description"),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"oid": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"group": schema.StringAttribute{
				Required:    true,
				Description: descriptions.Get("bookmark", "schema", "group"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validateFWOID(oid.TypeBookmarkGroup),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: descriptions.Get("bookmark", "schema", "name"),
			},
			"icon_url": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"target": schema.StringAttribute{
				Required:    true,
				Description: descriptions.Get("bookmark", "schema", "target"),
				PlanModifiers: []planmodifier.String{
					&oidVersionPlanModifier{},
				},
				Validators: []validator.String{
					validateFWOID(oid.TypeDataset, oid.TypeDashboard),
				},
			},
			"bookmark_kind": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: describeEnums(gql.AllBookmarkKindTypes, descriptions.Get("bookmark", "schema", "bookmark_kind")),
				Validators: []validator.String{
					validateFWEnums(gql.AllBookmarkKindTypes),
				},
			},
			"entity_tags": schema.MapAttribute{
				Optional:    true,
				Description: descriptions.Get("common", "schema", "entity_tags"),
				ElementType: entityTagsAttrType,
			},
		},
	}
}

func (r *bookmarkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *bookmarkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bookmarkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := r.buildBookmarkInput(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateBookmark(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create bookmark", err.Error())
		return
	}

	// Re-read to match SDKv2 behavior
	result, err := r.client.GetBookmark(ctx, created.Id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bookmark after create", err.Error())
		return
	}

	r.bookmarkToModel(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bookmarkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bookmarkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetBookmark(ctx, state.ID.ValueString())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to retrieve bookmark [id=%s]", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	r.bookmarkToModel(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bookmarkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bookmarkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := r.buildBookmarkInput(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateBookmark(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to update bookmark [id=%s]", plan.ID.ValueString()),
			err.Error(),
		)
		return
	}

	// Re-read to match SDKv2 behavior
	result, err := r.client.GetBookmark(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bookmark after update", err.Error())
		return
	}

	r.bookmarkToModel(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bookmarkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bookmarkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteBookmark(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete bookmark", err.Error())
	}
}

func (r *bookmarkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *bookmarkResource) buildBookmarkInput(ctx context.Context, plan *bookmarkResourceModel, diags *diag.Diagnostics) *gql.BookmarkInput {
	groupOid, _ := oid.NewOID(plan.Group.ValueString())
	targetOid, _ := oid.NewOID(plan.Target.ValueString())
	name := plan.Name.ValueString()

	input := &gql.BookmarkInput{
		Name:     &name,
		TargetId: &targetOid.Id,
		GroupId:  &groupOid.Id,
	}

	if !plan.IconURL.IsNull() && !plan.IconURL.IsUnknown() {
		input.IconUrl = stringPtr(plan.IconURL.ValueString())
	}

	if !plan.BookmarkKind.IsNull() && !plan.BookmarkKind.IsUnknown() {
		bookmarkKind := gql.BookmarkKind(toCamel(plan.BookmarkKind.ValueString()))
		input.BookmarkKind = &bookmarkKind
	}

	input.EntityTags = expandEntityTagsFromTFMap(ctx, plan.EntityTags, diags)
	if diags.HasError() {
		return nil
	}

	return input
}

func (r *bookmarkResource) bookmarkToModel(b *gql.Bookmark, model *bookmarkResourceModel) {
	model.ID = types.StringValue(b.Id)
	model.Name = types.StringValue(b.Name)
	model.OID = types.StringValue(b.Oid().String())
	model.Group = types.StringValue(oid.BookmarkGroupOid(b.GroupId).String())

	targetOid := oid.OID{
		Id:   b.TargetId,
		Type: oid.Type(strings.ToLower(string(b.TargetIdKind))),
	}
	model.Target = types.StringValue(targetOid.String())

	if b.IconUrl != "" {
		model.IconURL = types.StringValue(b.IconUrl)
	} else {
		model.IconURL = types.StringNull()
	}

	model.BookmarkKind = types.StringValue(toSnake(string(b.GetBookmarkKind())))

	model.EntityTags = flattenEntityTagsToTFMap(b.EntityTags)
}
