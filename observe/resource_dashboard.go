package observe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
)

const (
	schemaDashboardWorkspaceDescription       = "OID of workspace dashboard is contained in."
	schemaDashboardNameDescription            = "Dashboard name. Must be unique within workspace."
	schemaDashboardDescriptionDescription     = "Dashboard description."
	schemaDashboardIconDescription            = "Icon image."
	schemaDashboardJSONDescription            = "Dashboard stages in JSON format."
	schemaDashboardLayoutDescription          = "Dashboard layout in JSON format."
	schemaDashboardOIDDescription             = "The Observe ID for dashboard."
	schemaDashboardParametersDescription      = "Dashboard parameters in JSON format."
	schemaDashboardParameterValuesDescription = "Dashboard parameter values in JSON format."
	schemaDashboardSchemaVersionDescription   = "Dashboard content model version. Set to `2` or higher to manage the dashboard through its `definition` document; when unset (or less than `2`), the dashboard is managed through `stages`, `layout`, and `parameters`. **Alpha:** managing a dashboard through `definition` (`schema_version` >= `2`) is an alpha feature that is not generally available and is subject to backwards-incompatible changes in future releases. It must be enabled for your account; otherwise API requests are rejected. Not recommended for production dashboards."
	schemaDashboardDefinitionDescription      = "Dashboard content as a single JSON document. Required when `schema_version` is `2` or higher, and mutually exclusive with `stages`, `layout`, and `parameters`. **Alpha:** part of the `definition` content model (`schema_version` >= `2`); the document shape is not generally available and may change in backwards-incompatible ways. Must be enabled for your account."
)

func resourceDashboard() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages an Observe dashboard, which predefines visualizations of Observe data in a grid of cards.",
		CreateContext: resourceDashboardCreate,
		ReadContext:   resourceDashboardRead,
		UpdateContext: resourceDashboardUpdate,
		DeleteContext: resourceDashboardDelete,
		CustomizeDiff: dashboardCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validateOID(oid.TypeWorkspace),
				DiffSuppressFunc: diffSuppressWorkspace,
				Deprecated:       "workspace is no longer required and will be ignored. It may be removed in a future version.",
				Description:      schemaDashboardWorkspaceDescription,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: schemaDashboardNameDescription,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDashboardDescriptionDescription,
			},
			"icon_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: schemaDashboardIconDescription,
			},
			"schema_version": {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: validateDashboardSchemaVersion,
				Description:      schemaDashboardSchemaVersionDescription,
			},
			"definition": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaDashboardDefinitionDescription,
			},
			"stages": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressStageQueryInput,
				Description:      schemaDashboardJSONDescription,
			},
			"layout": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressJSON,
				Description:      schemaDashboardLayoutDescription,
			},
			"parameters": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressParameters,
				Description:      schemaDashboardParametersDescription,
			},
			"parameter_values": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateStringIsJSON,
				DiffSuppressFunc: diffSuppressParameterValues,
				Description:      schemaDashboardParameterValuesDescription,
			},
			"object_tags": objectTagsSchemaFieldOptional(),
			"entity_tags": entityTagsSchemaFieldOptional(),
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: schemaDashboardOIDDescription,
			},
		},
	}
}

// validateDashboardSchemaVersion warns that schema_version >= 2 opts into an alpha
// content model whose contract may still change. It surfaces at plan/validate time
// (including before the first apply) and never errors; the content-model split is
// enforced in dashboardCustomizeDiff.
func validateDashboardSchemaVersion(i interface{}, path cty.Path) diag.Diagnostics {
	v, ok := i.(int)
	if !ok || !dashboardUsesRestAPI(v) {
		return nil
	}
	return diag.Diagnostics{{
		Severity:      diag.Warning,
		Summary:       "schema_version >= 2 is an alpha feature",
		Detail:        "Setting schema_version >= 2 (and the 'definition' field) opts into an alpha dashboard content model that is not generally available, must be enabled for your account, and whose behavior and schema may change in backwards-incompatible ways in future releases. Avoid using it for production dashboards.",
		AttributePath: path,
	}}
}

// dashboardUsesRestAPI reports whether a dashboard is managed through the REST
// `definition` document rather than the legacy GraphQL stages/layout/parameters
// fields. schema_version < 2 is legacy; 2 and above is the new REST-backed model.
func dashboardUsesRestAPI(schemaVersion int) bool {
	return schemaVersion >= 2
}

// dashboardConfigHasAttr reports whether the practitioner wrote a value for attr in HCL
// config, independent of whether that value is currently known. d.GetOk is unsuitable
// for this: it treats an attribute whose value isn't yet resolvable at plan time (e.g. it
// interpolates an attribute of another resource created in the same apply) as absent,
// which is wrong here — the attribute is set, just not yet computable. GetRawConfig
// reflects the literal configuration instead of the resolved value.
func dashboardConfigHasAttr(d *schema.ResourceDiff, attr string) bool {
	raw := d.GetRawConfig()
	if raw.IsNull() {
		return false
	}
	return !raw.GetAttr(attr).IsNull()
}

// dashboardCustomizeDiff enforces the legacy/new content-model split. A new-model
// dashboard (schema_version >= 2) is described entirely by definition; the legacy
// content fields (stages/layout/parameters) are cleared by the backend and must not be
// set. A legacy dashboard (schema_version < 2) uses stages and must not set definition.
func dashboardCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	schemaVersion := d.Get("schema_version").(int)
	hasDefinition := dashboardConfigHasAttr(d, "definition")

	if dashboardUsesRestAPI(schemaVersion) {
		if !hasDefinition {
			return fmt.Errorf("schema_version >= 2 requires 'definition' to be set")
		}
		for _, field := range []string{"stages", "layout", "parameters", "parameter_values", "icon_url"} {
			if dashboardConfigHasAttr(d, field) {
				return fmt.Errorf("'%s' must not be set when schema_version >= 2; the dashboard content is carried entirely by 'definition'", field)
			}
		}
		return nil
	}

	// legacy: schema_version < 2
	if hasDefinition {
		return fmt.Errorf("'definition' can only be set when schema_version >= 2")
	}
	if !dashboardConfigHasAttr(d, "stages") {
		return fmt.Errorf("'stages' is required for legacy dashboards (schema_version < 2)")
	}
	return nil
}

// newDashboardRestCreateInput builds the REST create body for a schema_version >= 2
// dashboard. definition is embedded verbatim as a JSON object.
func newDashboardRestCreateInput(data *schema.ResourceData) (*rest.DashboardCreateInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	input := &rest.DashboardCreateInput{
		SchemaVersion: data.Get("schema_version").(int),
		Name:          data.Get("name").(string),
		Definition:    json.RawMessage(data.Get("definition").(string)),
	}

	if v, ok := data.GetOk("description"); ok {
		input.Description = stringPtr(v.(string))
	}

	// Always send objectTags so create matches the legacy GraphQL behavior of
	// establishing the full tag set. An empty map is omitted (no tags).
	input.ObjectTags = objectTagsMapFromReader(data)

	return input, diags
}

// newDashboardRestPatchInput builds a merge-patch (RFC 7396) body containing only the
// fields the practitioner changed. Omitted fields are left unchanged by the backend.
func newDashboardRestPatchInput(data *schema.ResourceData) (*rest.DashboardPatchInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	input := &rest.DashboardPatchInput{}

	if data.HasChange("name") {
		input.Name = stringPtr(data.Get("name").(string))
	}
	if data.HasChange("description") {
		input.Description = stringPtr(data.Get("description").(string))
	}
	if data.HasChange("definition") {
		input.Definition = json.RawMessage(data.Get("definition").(string))
	}
	if data.HasChange("schema_version") {
		v := data.Get("schema_version").(int)
		input.SchemaVersion = &v
	}
	if data.HasChange("object_tags") || data.HasChange("entity_tags") {
		tags := objectTagsMapFromReader(data)
		input.ObjectTags = &tags
	}

	return input, diags
}

func newDashboardConfig(data *schema.ResourceData) (input *gql.DashboardInput, diags diag.Diagnostics) {
	name := data.Get("name").(string)
	input = &gql.DashboardInput{
		Name: &name,
	}

	{
		// always reset to empty string if description not set
		input.Description = stringPtr(data.Get("description").(string))
	}

	if v, ok := data.GetOk("icon_url"); ok {
		input.IconUrl = stringPtr(v.(string))
	}

	if v, ok := data.GetOk("stages"); ok {
		data := v.(string)
		if err := json.Unmarshal([]byte(data), &input.Stages); err != nil {
			diagErr := fmt.Errorf("failed to parse 'stages' request field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		}
	}

	if v, ok := data.GetOk("layout"); ok {
		input.Layout = types.JsonObject(v.(string)).Ptr()
	}

	if v, ok := data.GetOk("parameters"); ok {
		data := v.(string)
		if err := json.Unmarshal([]byte(data), &input.Parameters); err != nil {
			diagErr := fmt.Errorf("failed to parse 'parameters' request field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		}
	}

	if v, ok := data.GetOk("parameter_values"); ok {
		data := v.(string)
		if err := json.Unmarshal([]byte(data), &input.ParameterValues); err != nil {
			diagErr := fmt.Errorf("failed to parse 'parameter_values' request field: %w", err)
			diags = append(diags, diag.FromErr(diagErr)...)
		}
	}

	// Always set ObjectTags, even if empty, to allow clearing tags
	input.ObjectTags = objectTagsInputFromReader(data)

	input.Visibility = asPointer(gql.ObjectVisibilityListed)

	return input, diags
}

func dashboardToResourceData(d *gql.Dashboard, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("workspace", oid.WorkspaceOid(d.WorkspaceId).String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := data.Set("name", d.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if d.Description != nil {
		if err := data.Set("description", *d.Description); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.IconUrl != nil {
		if err := data.Set("icon_url", *d.IconUrl); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if err := data.Set("schema_version", d.SchemaVersion); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// The content model is version-gated: the legacy model (schema_version < 2) carries
	// content in stages/layout/parameters and leaves definition null; the new model
	// (schema_version >= 2) carries everything in definition and leaves the legacy fields
	// empty. We only populate the fields for the active representation and clear the
	// other, so a dashboard whose schema_version changes in place doesn't leave stale
	// content (e.g. an empty stages "[]") behind in state and produce a perpetual diff.
	if d.SchemaVersion >= 2 {
		definition := ""
		if d.Definition != nil {
			definition = d.Definition.String()
		}
		for field, value := range map[string]string{
			"definition":       definition,
			"stages":           "",
			"layout":           "",
			"parameters":       "",
			"parameter_values": "",
		} {
			if err := data.Set(field, value); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}
	} else {
		if d.Stages != nil {
			// Hack hack hack hack hack
			for i, stage := range d.Stages {
				if stage.Id != nil && *stage.Id == "" {
					d.Stages[i].Id = nil
				}
				for j, input := range stage.Input {
					if input.StageId != nil && *input.StageId == "" {
						d.Stages[i].Input[j].StageId = nil
					}
				}
				if stage.Params != nil && *stage.Params == types.JsonObject("null") {
					d.Stages[i].Params = nil
				} else if stage.Params != nil && string(*stage.Params) == "" {
					d.Stages[i].Params = nil
				}
			}
			if stagesRaw, err := json.Marshal(d.Stages); err != nil {
				diagErr := fmt.Errorf("failed to parse 'stages' response field: %w", err)
				diags = append(diags, diag.FromErr(diagErr)...)
			} else if err := data.Set("stages", string(stagesRaw)); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}

		if d.Parameters != nil {
			if parametersRaw, err := json.Marshal(d.Parameters); err != nil {
				diagErr := fmt.Errorf("failed to parse 'parameters' response field: %w", err)
				diags = append(diags, diag.FromErr(diagErr)...)
			} else if err := data.Set("parameters", string(parametersRaw)); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}

		if d.ParameterValues != nil {
			if parameterValuesRaw, err := json.Marshal(d.ParameterValues); err != nil {
				diagErr := fmt.Errorf("failed to parse 'parameter_values' response field: %w", err)
				diags = append(diags, diag.FromErr(diagErr)...)
			} else if err := data.Set("parameter_values", string(parameterValuesRaw)); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}

		if d.Layout != nil {
			if err := data.Set("layout", d.Layout); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}

		if err := data.Set("definition", ""); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if tagDiags, err := setObjectTagsFromAPI(data, d.ObjectTags); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	} else {
		diags = append(diags, tagDiags...)
	}

	if err := data.Set("oid", d.Oid().String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceDashboardCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	if dashboardUsesRestAPI(data.Get("schema_version").(int)) {
		input, configDiags := newDashboardRestCreateInput(data)
		diags = append(diags, configDiags...)
		if diags.HasError() {
			return diags
		}
		result, err := client.CreateDashboardRest(ctx, input)
		if err != nil {
			return append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "failed to create dashboard",
				Detail:   err.Error(),
			})
		}
		data.SetId(result.Id)
		return append(diags, resourceDashboardRead(ctx, data, meta)...)
	}

	config, configDiags := newDashboardConfig(data)
	diags = append(diags, configDiags...)
	if diags.HasError() {
		return diags
	}

	wsid, err := client.ResolveWorkspaceID(ctx, maybeString(data.GetOk("workspace")))
	if err != nil {
		return diag.FromErr(err)
	}
	result, err := client.CreateDashboard(ctx, wsid, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create dashboard",
			Detail:   err.Error(),
		})
		return diags
	}

	data.SetId(result.Id)
	return append(diags, resourceDashboardRead(ctx, data, meta)...)
}

func resourceDashboardRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetDashboard(ctx, data.Id())
	if err != nil {
		if gql.HasErrorCode(err, "NOT_FOUND") {
			data.SetId("")
			return nil
		}
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to retrieve dashboard [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
	}

	return dashboardToResourceData(result, data)
}

func resourceDashboardUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	if dashboardUsesRestAPI(data.Get("schema_version").(int)) {
		input, configDiags := newDashboardRestPatchInput(data)
		diags = append(diags, configDiags...)
		if diags.HasError() {
			return diags
		}
		if _, err := client.UpdateDashboardRest(ctx, data.Id(), input); err != nil {
			return append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("failed to update dashboard [id=%s]", data.Id()),
				Detail:   err.Error(),
			})
		}
		return append(diags, resourceDashboardRead(ctx, data, meta)...)
	}

	config, configDiags := newDashboardConfig(data)
	diags = append(diags, configDiags...)
	if diags.HasError() {
		return diags
	}

	wsid, err := client.ResolveWorkspaceID(ctx, maybeString(data.GetOk("workspace")))
	if err != nil {
		return diag.FromErr(err)
	}
	result, err := client.UpdateDashboard(ctx, data.Id(), wsid, config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update dashboard [id=%s]", data.Id()),
			Detail:   err.Error(),
		})
		return diags
	}

	return dashboardToResourceData(result, data)
}

func resourceDashboardDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteDashboard(ctx, data.Id()); err != nil {
		return diag.Errorf("failed to delete dashboard: %s", err)
	}
	return diags
}
