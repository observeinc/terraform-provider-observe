package observe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

func resourceSkill() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("skill", "description"),
		CreateContext: resourceSkillCreate,
		ReadContext:   resourceSkillRead,
		UpdateContext: resourceSkillUpdate,
		DeleteContext: resourceSkillDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
			"label": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("skill", "schema", "label"),
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("skill", "schema", "description"),
			},
			"content": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("skill", "schema", "content"),
			},
			"visibility": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          skillVisibilityWorkspace,
				ValidateDiagFunc: validateEnums([]string{skillVisibilityWorkspace, skillVisibilityPrivate}),
				Description:      descriptions.Get("skill", "schema", "visibility"),
			},
		},
	}
}

func skillToResourceData(skill *rest.SkillResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	setResourceData := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	data.SetId(skill.Id)
	setResourceData("oid", skill.Oid().String())
	setResourceData("label", skill.Label)
	setResourceData("description", skill.Description)
	setResourceData("content", skill.Content)

	tfVis, err := skillTerraformVisibilityFromAPI(skill.Visibility)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	setResourceData("visibility", tfVis)

	return diags
}

func skillCreateRequestFromResourceData(data *schema.ResourceData) (*rest.SkillCreateRequest, error) {
	apiVis, err := skillAPIVisibilityFromTerraform(data.Get("visibility").(string))
	if err != nil {
		return nil, err
	}
	return &rest.SkillCreateRequest{
		Label:       data.Get("label").(string),
		Description: data.Get("description").(string),
		Content:     data.Get("content").(string),
		Visibility:  apiVis,
	}, nil
}

// skillUpdateRequestFromResourceData builds a PATCH request containing only changed fields.
func skillUpdateRequestFromResourceData(data *schema.ResourceData) (*rest.SkillUpdateRequest, error) {
	req := &rest.SkillUpdateRequest{}
	if data.HasChange("label") {
		req.Label = stringPtr(data.Get("label").(string))
	}
	if data.HasChange("description") {
		req.Description = stringPtr(data.Get("description").(string))
	}
	if data.HasChange("content") {
		req.Content = stringPtr(data.Get("content").(string))
	}
	if data.HasChange("visibility") {
		apiVis, err := skillAPIVisibilityFromTerraform(data.Get("visibility").(string))
		if err != nil {
			return nil, err
		}
		req.Visibility = skillVisibilityPtr(apiVis)
	}
	return req, nil
}

func resourceSkillCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	req, err := skillCreateRequestFromResourceData(data)
	if err != nil {
		return diag.FromErr(err)
	}

	skill, err := client.CreateSkill(ctx, req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to create skill",
			Detail:   err.Error(),
		})
		return diags
	}
	data.SetId(skill.Id)
	return append(diags, skillToResourceData(skill, data)...)
}

func resourceSkillRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	skill, err := client.GetSkill(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve skill: %s", err.Error())
	}
	return skillToResourceData(skill, data)
}

func resourceSkillUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	req, err := skillUpdateRequestFromResourceData(data)
	if err != nil {
		return diag.FromErr(err)
	}
	if req.Label == nil && req.Description == nil && req.Content == nil && req.Visibility == nil {
		return resourceSkillRead(ctx, data, meta)
	}

	skill, err := client.UpdateSkill(ctx, data.Id(), req)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to update skill",
			Detail:   err.Error(),
		})
		return diags
	}
	return append(diags, skillToResourceData(skill, data)...)
}

func resourceSkillDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteSkill(ctx, data.Id()); err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			return diags
		}
		return diag.Errorf("failed to delete skill: %s", err.Error())
	}
	return diags
}

const (
	skillVisibilityWorkspace = "Workspace"
	skillVisibilityPrivate   = "Private"
)

func skillAPIVisibilityFromTerraform(tf string) (rest.SkillVisibility, error) {
	if tf == "" {
		tf = skillVisibilityWorkspace
	}
	switch tf {
	case skillVisibilityWorkspace:
		return rest.SkillVisibilityListed, nil
	case skillVisibilityPrivate:
		return rest.SkillVisibilityUnlisted, nil
	default:
		return rest.SkillVisibility(""), fmt.Errorf("invalid skill visibility %q (expected %q or %q)", tf, skillVisibilityWorkspace, skillVisibilityPrivate)
	}
}

func skillTerraformVisibilityFromAPI(api rest.SkillVisibility) (string, error) {
	switch api {
	case rest.SkillVisibilityListed:
		return skillVisibilityWorkspace, nil
	case rest.SkillVisibilityUnlisted:
		return skillVisibilityPrivate, nil
	default:
		return "", fmt.Errorf("unsupported API skill visibility %q", string(api))
	}
}

func skillVisibilityPtr(v rest.SkillVisibility) *rest.SkillVisibility {
	return &v
}
