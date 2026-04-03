package observe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwschema "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	metatypes "github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

var (
	_ resource.Resource                = &datasetResource{}
	_ resource.ResourceWithConfigure   = &datasetResource{}
	_ resource.ResourceWithModifyPlan  = &datasetResource{}
	_ resource.ResourceWithImportState = &datasetResource{}
)

type datasetResource struct {
	client *observe.Client
}

func NewDatasetResource() resource.Resource {
	return &datasetResource{}
}

type datasetResourceModel struct {
	ID                            types.String   `tfsdk:"id"`
	OID                           types.String   `tfsdk:"oid"`
	Workspace                     types.String   `tfsdk:"workspace"`
	Name                          types.String   `tfsdk:"name"`
	Description                   types.String   `tfsdk:"description"`
	IconURL                       types.String   `tfsdk:"icon_url"`
	PathCost                      types.Int64    `tfsdk:"path_cost"`
	OnDemandMaterializationLength types.String   `tfsdk:"on_demand_materialization_length"`
	Freshness                     types.String   `tfsdk:"freshness"`
	AccelerationDisabled          types.Bool     `tfsdk:"acceleration_disabled"`
	AccelerationDisabledSource    types.String   `tfsdk:"acceleration_disabled_source"`
	Inputs                        types.Map      `tfsdk:"inputs"`
	DataTableViewState            types.String   `tfsdk:"data_table_view_state"`
	StorageIntegration            types.String   `tfsdk:"storage_integration"`
	Stage                         []fwStageModel `tfsdk:"stage"`
	RematerializationMode         types.String   `tfsdk:"rematerialization_mode"`
	EntityTags                    types.Map      `tfsdk:"entity_tags"`
}

func (r *datasetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *datasetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: descriptions.Get("dataset", "description"),
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
			"workspace": schema.StringAttribute{
				Required:    true,
				Description: descriptions.Get("common", "schema", "workspace"),
				Validators:  []fwschema.String{validateFWOID(oid.TypeWorkspace)},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: descriptions.Get("dataset", "schema", "name"),
				Validators:  []fwschema.String{validateFWDatasetName()},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "description"),
			},
			"icon_url": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"path_cost": schema.Int64Attribute{
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "path_cost"),
			},
			"on_demand_materialization_length": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "on_demand_materialization_length"),
				Validators:  []fwschema.String{validateFWTimeDuration()},
				PlanModifiers: []planmodifier.String{
					&timeDurationPlanModifier{ceilDays: true},
				},
			},
			"freshness": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("transform", "schema", "freshness"),
				Validators:  []fwschema.String{validateFWTimeDuration()},
				PlanModifiers: []planmodifier.String{
					&timeDurationPlanModifier{},
				},
			},
			"acceleration_disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "acceleration_disabled"),
			},
			"acceleration_disabled_source": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "acceleration_disabled_source"),
				Validators:  []fwschema.String{validateFWEnums(gql.AllAccelerationDisabledSource)},
				PlanModifiers: []planmodifier.String{
					&enumPlanModifier{},
				},
			},
			"inputs": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: descriptions.Get("transform", "schema", "inputs"),
			},
			"data_table_view_state": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "data_table_view_state"),
				Validators:  []fwschema.String{validateFWJSON()},
				PlanModifiers: []planmodifier.String{
					&jsonPlanModifier{},
				},
			},
			"storage_integration": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "storage_integration"),
				Validators:  []fwschema.String{validateFWOID(oid.TypeStorageIntegration)},
			},
			"rematerialization_mode": schema.StringAttribute{
				Optional:    true,
				Description: descriptions.Get("dataset", "schema", "rematerialization_mode"),
				Validators:  []fwschema.String{validateFWEnums(AllRematerializationModes)},
				PlanModifiers: []planmodifier.String{
					&enumPlanModifier{},
				},
			},
			"entity_tags": schema.MapAttribute{
				Optional:    true,
				Description: descriptions.Get("common", "schema", "entity_tags"),
				ElementType: entityTagsAttrType,
			},
		},
		Blocks: map[string]schema.Block{
			"stage": schema.ListNestedBlock{
				Description: descriptions.Get("transform", "schema", "stage", "description"),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"alias": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
						"input": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "input"),
						},
						"pipeline": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "pipeline"),
							PlanModifiers: []planmodifier.String{
								&pipelinePlanModifier{},
							},
						},
						"output_stage": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "output_stage"),
						},
					},
				},
			},
		},
	}
}

func (r *datasetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *datasetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy plan — nothing to validate
	if req.Plan.Raw.IsNull() {
		return
	}

	// If there's no state yet (create), mark OID as unknown since we can't predict it
	if req.State.Raw.IsNull() {
		return
	}

	var plan datasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state datasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Suppress alias diff on last stage (mirrors SDKv2 DiffSuppressFunc)
	if len(plan.Stage) > 0 {
		lastIdx := len(plan.Stage) - 1
		if len(state.Stage) > lastIdx {
			planAlias := plan.Stage[lastIdx].Alias.ValueString()
			stateAlias := state.Stage[lastIdx].Alias.ValueString()
			if planAlias != stateAlias {
				plan.Stage[lastIdx].Alias = state.Stage[lastIdx].Alias
			}
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validatePlan(ctx, &plan, &state, &resp.Diagnostics)
}

func (r *datasetResource) validatePlan(ctx context.Context, plan *datasetResourceModel, state *datasetResourceModel, diags *diag.Diagnostics) {
	if r.client == nil || r.client.SkipDatasetDryRuns {
		return
	}

	// Only validate if inputs, stage, or name changed
	inputsChanged := !plan.Inputs.Equal(state.Inputs)
	stagesChanged := !stagesEqual(plan.Stage, state.Stage)
	nameChanged := plan.Name.ValueString() != state.Name.ValueString()
	if !inputsChanged && !stagesChanged && !nameChanged {
		return
	}

	// Skip if any values are unknown (can't do dry-run without fully known config)
	if plan.Inputs.IsUnknown() || plan.Workspace.IsUnknown() || plan.Name.IsUnknown() {
		return
	}
	for _, s := range plan.Stage {
		if s.Pipeline.IsUnknown() || s.Input.IsUnknown() || s.Alias.IsUnknown() {
			return
		}
	}

	input, queryInput := r.buildDatasetInput(ctx, plan, diags)
	if diags.HasError() {
		return
	}

	wsOid, _ := oid.NewOID(plan.Workspace.ValueString())
	if !state.ID.IsNull() && !state.ID.IsUnknown() {
		id := state.ID.ValueString()
		input.Id = &id
	}

	result, err := r.client.SaveDatasetDryRun(ctx, wsOid.Id, input, queryInput)
	if err != nil {
		diags.AddError("Dataset save dry-run failed", err.Error())
		return
	}

	rematerializationMode := r.getRematerializationMode(plan)
	if len(result.DematerializedDatasets) > 0 {
		msg := rematerializationErrorStr(result.DematerializedDatasets)
		switch rematerializationMode {
		case RematerializationModeMustSkipRematerialization:
			diags.AddError("Rematerialization would occur", msg)
		case RematerializationModeSkipRematerialization:
			diags.AddWarning("Rematerialization cannot be avoided", msg)
		}
	}
}

func stagesEqual(a, b []fwStageModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Pipeline.ValueString() != b[i].Pipeline.ValueString() ||
			a[i].Input.ValueString() != b[i].Input.ValueString() ||
			a[i].Alias.ValueString() != b[i].Alias.ValueString() ||
			a[i].OutputStage.ValueBool() != b[i].OutputStage.ValueBool() {
			return false
		}
	}
	return true
}

func (r *datasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan datasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, queryInput := r.buildDatasetInput(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	dependencyHandling := gql.DefaultDependencyHandling()
	if !plan.RematerializationMode.IsNull() && !plan.RematerializationMode.IsUnknown() {
		rematerializationMode := gql.RematerializationMode(toCamel(plan.RematerializationMode.ValueString()))
		dependencyHandling.RematerializationMode = &rematerializationMode
		resp.Diagnostics.AddWarning("rematerialization_mode on a new dataset is a no-op", "")
	}

	wsOid, _ := oid.NewOID(plan.Workspace.ValueString())
	result, err := r.client.SaveDataset(ctx, wsOid.Id, input, queryInput, dependencyHandling)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create dataset", err.Error())
		return
	}

	// Re-read to match SDKv2 behavior
	readResult, err := r.client.GetDataset(ctx, result.Id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read dataset after create", err.Error())
		return
	}

	r.datasetToModel(ctx, readResult, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *datasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state datasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetDataset(ctx, state.ID.ValueString())
	if err != nil {
		if gql.HasErrorCode(err, gql.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to retrieve dataset [id=%s]", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	r.datasetToModel(ctx, result, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *datasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan datasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, queryInput := r.buildDatasetInput(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.ID.ValueString()
	input.Id = &id
	wsOid, _ := oid.NewOID(plan.Workspace.ValueString())

	// Double-check rematerialization constraint at apply time (plan may have been generated earlier)
	rematerializationMode := r.getRematerializationMode(&plan)
	if rematerializationMode == RematerializationModeMustSkipRematerialization {
		if result, err := r.client.SaveDatasetDryRun(ctx, wsOid.Id, input, queryInput); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to update dataset [id=%s]", id), err.Error())
			return
		} else if len(result.DematerializedDatasets) > 0 {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update dataset [id=%s]", id),
				rematerializationErrorStr(result.DematerializedDatasets),
			)
			return
		}
	}

	dependencyHandling := gql.DefaultDependencyHandling()
	switch rematerializationMode {
	case RematerializationModeSkipRematerialization, RematerializationModeMustSkipRematerialization:
		mode := gql.RematerializationModeSkiprematerialization
		dependencyHandling.RematerializationMode = &mode
	}

	_, err := r.client.SaveDataset(ctx, wsOid.Id, input, queryInput, dependencyHandling)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update dataset [id=%s]", id), err.Error())
		return
	}

	// Re-read to match SDKv2 behavior
	result, err := r.client.GetDataset(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read dataset after update", err.Error())
		return
	}

	r.datasetToModel(ctx, result, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *datasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDataset(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete dataset", err.Error())
	}
}

func (r *datasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	result, err := r.client.GetDataset(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import dataset", err.Error())
		return
	}

	var model datasetResourceModel
	r.datasetToModel(ctx, result, &model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *datasetResource) buildDatasetInput(ctx context.Context, model *datasetResourceModel, diags *diag.Diagnostics) (*gql.DatasetInput, *gql.MultiStageQueryInput) {
	// Extract inputs map
	var inputsMap map[string]string
	diags.Append(model.Inputs.ElementsAs(ctx, &inputsMap, false)...)
	if diags.HasError() {
		return nil, nil
	}

	queryInput, queryDiags := buildMultiStageQuery(inputsMap, model.Stage)
	diags.Append(queryDiags...)
	if diags.HasError() {
		return nil, nil
	}

	overwriteSource := true
	input := &gql.DatasetInput{
		OverwriteSource: &overwriteSource,
		Label:           model.Name.ValueString(),
	}

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		s := model.Description.ValueString()
		input.Description = &s
	} else {
		s := ""
		input.Description = &s
	}

	if !model.Freshness.IsNull() && !model.Freshness.IsUnknown() {
		t, _ := time.ParseDuration(model.Freshness.ValueString())
		input.FreshnessDesired = metatypes.Int64Scalar(t).Ptr()
	}

	if !model.OnDemandMaterializationLength.IsNull() && !model.OnDemandMaterializationLength.IsUnknown() {
		t, _ := time.ParseDuration(model.OnDemandMaterializationLength.ValueString())
		input.OnDemandMaterializationLength = metatypes.Int64Scalar(t).Ptr()
	}

	if !model.IconURL.IsNull() && !model.IconURL.IsUnknown() {
		input.IconUrl = stringPtr(model.IconURL.ValueString())
	}

	b := model.AccelerationDisabled.ValueBool()
	input.AccelerationDisabled = &b

	if !model.AccelerationDisabledSource.IsNull() && !model.AccelerationDisabledSource.IsUnknown() {
		c := gql.AccelerationDisabledSource(toCamel(model.AccelerationDisabledSource.ValueString()))
		input.AccelerationDisabledSource = &c
	}

	if !model.PathCost.IsNull() && !model.PathCost.IsUnknown() {
		input.PathCost = metatypes.Int64Scalar(model.PathCost.ValueInt64()).Ptr()
	} else {
		input.PathCost = metatypes.Int64Scalar(0).Ptr()
	}

	if !model.DataTableViewState.IsNull() && !model.DataTableViewState.IsUnknown() {
		input.DataTableViewState = metatypes.JsonObject(model.DataTableViewState.ValueString()).Ptr()
	} else {
		input.DataTableViewState = metatypes.JsonObject("null").Ptr()
	}

	if !model.StorageIntegration.IsNull() && !model.StorageIntegration.IsUnknown() {
		oidVal, _ := oid.NewOID(model.StorageIntegration.ValueString())
		input.StorageIntegrationId = stringPtr(oidVal.Id)
	}

	entityTags := expandEntityTagsFromTFMap(ctx, model.EntityTags, diags)
	if diags.HasError() {
		return nil, nil
	}
	if entityTags == nil {
		entityTags = []gql.EntityTagMappingInput{}
	}
	input.EntityTags = entityTags

	return input, queryInput
}

func (r *datasetResource) datasetToModel(ctx context.Context, d *gql.Dataset, model *datasetResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(d.Id)
	model.OID = types.StringValue(d.Oid().String())
	model.Workspace = types.StringValue(oid.WorkspaceOid(d.WorkspaceId).String())
	model.Name = types.StringValue(d.Name)

	if d.Description != nil {
		model.Description = types.StringValue(*d.Description)
	} else {
		model.Description = types.StringNull()
	}

	if d.IconUrl != nil {
		model.IconURL = types.StringValue(*d.IconUrl)
	} else {
		model.IconURL = types.StringNull()
	}

	model.AccelerationDisabled = types.BoolValue(d.AccelerationDisabled)
	model.AccelerationDisabledSource = types.StringValue(toSnake(string(d.AccelerationDisabledSource)))

	if d.FreshnessDesired != nil {
		model.Freshness = types.StringValue(d.FreshnessDesired.Duration().String())
	} else if !model.Freshness.IsNull() {
		model.Freshness = types.StringNull()
	}

	if d.OnDemandMaterializationLength != nil {
		model.OnDemandMaterializationLength = types.StringValue(d.OnDemandMaterializationLength.Duration().String())
	} else if !model.OnDemandMaterializationLength.IsNull() {
		model.OnDemandMaterializationLength = types.StringNull()
	}

	// Only update path_cost if it differs from current state, matching SDKv2 behavior
	if d.PathCost != nil {
		newCost := int64(*d.PathCost.IntPtr())
		currentCost := model.PathCost.ValueInt64()
		if newCost != currentCost {
			model.PathCost = types.Int64Value(newCost)
		}
	}

	if d.DataTableViewState != nil {
		model.DataTableViewState = types.StringValue(d.DataTableViewState.String())
	} else {
		model.DataTableViewState = types.StringNull()
	}

	if d.StorageIntegrationId != nil {
		model.StorageIntegration = types.StringValue(oid.StorageIntegrationOid(*d.StorageIntegrationId).String())
	} else {
		model.StorageIntegration = types.StringNull()
	}

	model.EntityTags = flattenEntityTagsToTFMap(d.EntityTags)

	// Flatten query (stages + inputs)
	if d.Transform != nil && d.Transform.Current != nil && d.Transform.Current.Query != nil {
		existingInputs := make(map[string]string)
		if !model.Inputs.IsNull() && !model.Inputs.IsUnknown() {
			diags.Append(model.Inputs.ElementsAs(ctx, &existingInputs, false)...)
			if diags.HasError() {
				return
			}
		}

		existingFirstStageInput := ""
		if len(model.Stage) > 0 {
			existingFirstStageInput = model.Stage[0].Input.ValueString()
		}

		inputs, stages, queryDiags := flattenQueryToModel(ctx, d.Transform.Current.Query.Stages, d.Transform.Current.Query.OutputStage, existingInputs, existingFirstStageInput)
		diags.Append(queryDiags...)
		if diags.HasError() {
			return
		}

		inputsMapValue, mapDiags := types.MapValueFrom(ctx, types.StringType, inputs)
		diags.Append(mapDiags...)
		if diags.HasError() {
			return
		}
		model.Inputs = inputsMapValue
		model.Stage = stages
	}
}

func (r *datasetResource) getRematerializationMode(model *datasetResourceModel) TerraformRematerializationMode {
	mode := RematerializationModeRematerialize
	if r.client.DefaultRematerializationMode != nil {
		mode = TerraformRematerializationMode(toCamel(*r.client.DefaultRematerializationMode))
	}
	if !model.RematerializationMode.IsNull() && !model.RematerializationMode.IsUnknown() {
		mode = TerraformRematerializationMode(toCamel(model.RematerializationMode.ValueString()))
	}
	return mode
}
