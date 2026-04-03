package observe

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwschema "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

var (
	_ datasource.DataSource                   = &datasetDataSource{}
	_ datasource.DataSourceWithConfigure      = &datasetDataSource{}
	_ datasource.DataSourceWithValidateConfig = &datasetDataSource{}
)

type datasetDataSource struct {
	client *observe.Client
}

func NewDatasetDataSource() datasource.DataSource {
	return &datasetDataSource{}
}

type datasetDataSourceModel struct {
	ID                            types.String             `tfsdk:"id"`
	OID                           types.String             `tfsdk:"oid"`
	Workspace                     types.String             `tfsdk:"workspace"`
	Name                          types.String             `tfsdk:"name"`
	Description                   types.String             `tfsdk:"description"`
	IconURL                       types.String             `tfsdk:"icon_url"`
	PathCost                      types.Int64              `tfsdk:"path_cost"`
	OnDemandMaterializationLength types.String             `tfsdk:"on_demand_materialization_length"`
	Freshness                     types.String             `tfsdk:"freshness"`
	AccelerationDisabled          types.Bool               `tfsdk:"acceleration_disabled"`
	AccelerationDisabledSource    types.String             `tfsdk:"acceleration_disabled_source"`
	Inputs                        types.Map                `tfsdk:"inputs"`
	DataTableViewState            types.String             `tfsdk:"data_table_view_state"`
	StorageIntegration            types.String             `tfsdk:"storage_integration"`
	Stage                         []fwStageModel           `tfsdk:"stage"`
	CorrelationTag                []correlationTagDSModel  `tfsdk:"correlation_tag"`
	EntityTags                    types.Map                `tfsdk:"entity_tags"`
}

type correlationTagDSModel struct {
	Name   types.String `tfsdk:"name"`
	Column types.String `tfsdk:"column"`
	Path   types.String `tfsdk:"path"`
}

func (d *datasetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (d *datasetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches metadata for an existing Observe dataset.",
		Attributes: map[string]schema.Attribute{
			"workspace": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "workspace"),
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Validators:  []fwschema.String{validateFWStringNotEmpty()},
				Description: descriptions.Get("dataset", "schema", "name") +
					" One of `name` or `id` must be set. If `name` is provided, `workspace` must be set.",
			},
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Validators:  []fwschema.String{validateFWRegex(`^\d+$`, "expected ID to be valid integer")},
				Description: descriptions.Get("common", "schema", "id") +
					" One of `name` or `id` must be set.",
			},
			"oid": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "description"),
			},
			"icon_url": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("common", "schema", "icon_url"),
			},
			"acceleration_disabled": schema.BoolAttribute{
				Computed: true,
			},
			"acceleration_disabled_source": schema.StringAttribute{
				Computed: true,
			},
			"path_cost": schema.Int64Attribute{
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "path_cost"),
			},
			"on_demand_materialization_length": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "on_demand_materialization_length"),
			},
			"freshness": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("transform", "schema", "freshness"),
			},
			"inputs": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: descriptions.Get("transform", "schema", "inputs"),
			},
			"data_table_view_state": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "data_table_view_state"),
			},
			"storage_integration": schema.StringAttribute{
				Computed:    true,
				Description: descriptions.Get("dataset", "schema", "storage_integration"),
			},
			"entity_tags": schema.MapAttribute{
				Computed:    true,
				ElementType: entityTagsAttrType,
				Description: descriptions.Get("common", "schema", "entity_tags"),
			},
		},
		Blocks: map[string]schema.Block{
			"stage": schema.ListNestedBlock{
				Description: descriptions.Get("transform", "schema", "stage", "description"),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"alias": schema.StringAttribute{
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "alias"),
						},
						"input": schema.StringAttribute{
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "input"),
						},
						"pipeline": schema.StringAttribute{
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "pipeline"),
						},
						"output_stage": schema.BoolAttribute{
							Computed:    true,
							Description: descriptions.Get("transform", "schema", "stage", "output_stage"),
						},
					},
				},
			},
			"correlation_tag": schema.ListNestedBlock{
				Description: descriptions.Get("dataset", "schema", "correlation_tag", "description"),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: descriptions.Get("correlation_tag", "schema", "name"),
						},
						"column": schema.StringAttribute{
							Computed:    true,
							Description: descriptions.Get("correlation_tag", "schema", "column"),
						},
						"path": schema.StringAttribute{
							Computed:    true,
							Description: descriptions.Get("correlation_tag", "schema", "path"),
						},
					},
				},
			},
		},
	}
}

func (d *datasetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *datasetDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var model datasetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasName := !model.Name.IsNull() && !model.Name.IsUnknown()
	hasID := !model.ID.IsNull() && !model.ID.IsUnknown()
	hasWorkspace := !model.Workspace.IsNull() && !model.Workspace.IsUnknown()

	if !hasName && !hasID {
		resp.Diagnostics.AddError("Missing required attribute", "One of \"name\" or \"id\" must be set.")
	}
	if hasName && hasID {
		resp.Diagnostics.AddError("Conflicting attributes", "Only one of \"name\" or \"id\" may be set.")
	}
	if hasName && !hasWorkspace {
		resp.Diagnostics.AddError("Missing required attribute", "\"workspace\" is required when \"name\" is specified.")
	}
}

func (d *datasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model datasetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dataset *gql.Dataset
	var err error

	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != "" {
		dataset, err = d.client.GetDataset(ctx, model.ID.ValueString())
	} else {
		name := model.Name.ValueString()
		wsOid, _ := oid.NewOID(model.Workspace.ValueString())
		dataset, err = d.client.LookupDataset(ctx, wsOid.Id, name)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to read dataset %q", name),
				err.Error(),
			)
			return
		}
	}

	if err != nil {
		resp.Diagnostics.AddError("Failed to read dataset", err.Error())
		return
	}

	d.datasetToDataSourceModel(ctx, dataset, &model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (d *datasetDataSource) datasetToDataSourceModel(ctx context.Context, ds *gql.Dataset, model *datasetDataSourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(ds.Id)
	model.OID = types.StringValue(ds.Oid().String())
	model.Workspace = types.StringValue(oid.WorkspaceOid(ds.WorkspaceId).String())
	model.Name = types.StringValue(ds.Name)

	if ds.Description != nil {
		model.Description = types.StringValue(*ds.Description)
	} else {
		model.Description = types.StringNull()
	}

	if ds.IconUrl != nil {
		model.IconURL = types.StringValue(*ds.IconUrl)
	} else {
		model.IconURL = types.StringNull()
	}

	model.AccelerationDisabled = types.BoolValue(ds.AccelerationDisabled)
	model.AccelerationDisabledSource = types.StringValue(toSnake(string(ds.AccelerationDisabledSource)))

	if ds.FreshnessDesired != nil {
		model.Freshness = types.StringValue(ds.FreshnessDesired.Duration().String())
	} else {
		model.Freshness = types.StringNull()
	}

	if ds.OnDemandMaterializationLength != nil {
		model.OnDemandMaterializationLength = types.StringValue(ds.OnDemandMaterializationLength.Duration().String())
	} else {
		model.OnDemandMaterializationLength = types.StringNull()
	}

	if ds.PathCost != nil {
		model.PathCost = types.Int64Value(int64(*ds.PathCost.IntPtr()))
	} else {
		model.PathCost = types.Int64Null()
	}

	if ds.DataTableViewState != nil {
		model.DataTableViewState = types.StringValue(ds.DataTableViewState.String())
	} else {
		model.DataTableViewState = types.StringNull()
	}

	if ds.StorageIntegrationId != nil {
		model.StorageIntegration = types.StringValue(oid.StorageIntegrationOid(*ds.StorageIntegrationId).String())
	} else {
		model.StorageIntegration = types.StringNull()
	}

	model.EntityTags = flattenEntityTagsToTFMap(ds.EntityTags)

	// Flatten query
	if ds.Transform != nil && ds.Transform.Current != nil && ds.Transform.Current.Query != nil {
		inputs, stages, queryDiags := flattenQueryToModel(ctx, ds.Transform.Current.Query.Stages, ds.Transform.Current.Query.OutputStage, nil, "")
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

	// Correlation tags
	if ds.CorrelationTagMappings != nil {
		model.CorrelationTag = make([]correlationTagDSModel, len(ds.CorrelationTagMappings))
		for i, ct := range ds.CorrelationTagMappings {
			model.CorrelationTag[i] = correlationTagDSModel{
				Name:   types.StringValue(ct.Tag),
				Column: types.StringValue(ct.Path.Column),
			}
			if ct.Path.Path != nil {
				model.CorrelationTag[i].Path = types.StringValue(*ct.Path.Path)
			} else {
				model.CorrelationTag[i].Path = types.StringNull()
			}
		}
	}
}
