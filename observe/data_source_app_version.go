package observe

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
	"hash/crc32"
	"sort"
	"strconv"
)

func dataSourceAppVersion() *schema.Resource {
	return &schema.Resource{
		Description: descriptions.Get("app_version", "description"),
		ReadContext: dataSourceAppVersionRead,
		Schema: map[string]*schema.Schema{
			"module_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("app_version", "schema", "module_id"),
			},
			"version_constraint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("app_version", "schema", "version_constraint"),
			},
			"include_prerelease": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("app_version", "schema", "include_prerelease"),
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("app_version", "schema", "version"),
			},
		},
	}
}

func matchVersionString(moduleId string, moduleVersions []*meta.ModuleVersion, versionConstraint string, includePrerelease bool) (string, error) {
	constraints, err := version.NewConstraint(versionConstraint)
	if err != nil {
		return "", err
	}
	versions := make([]*version.Version, 0, len(moduleVersions))
	for _, raw := range moduleVersions {
		v, e := version.NewVersion(raw.Version)
		// If we get back bad version data from the server, we don't
		// need to error out since we can still find a matching version
		if e != nil {
			continue
		}
		// Only add the new versionConstraint to the versions slice
		// if there is no includePrerelease value or if the user
		// wants to include prerelease values
		if v.Prerelease() == "" || includePrerelease {
			versions = append(versions, v)
		}
	}
	sort.Sort(sort.Reverse(version.Collection(versions)))

	for _, v := range versions {
		if constraints.Check(v) {
			return v.Original(), nil
		}
	}

	return "", fmt.Errorf("no matching version found for module_id: %q and version_constraint: %q", moduleId, versionConstraint)
}

func dataSourceAppVersionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	var (
		client            = meta.(*observe.Client)
		moduleId          = data.Get("module_id").(string)
		versionConstraint = data.Get("version_constraint").(string)
		includePrerelease = data.Get("include_prerelease").(bool)
	)

	moduleVersions, err := client.LookupModuleVersions(ctx, moduleId)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}

	version, err := matchVersionString(moduleId, moduleVersions, versionConstraint, includePrerelease)
	if err != nil {
		diags = diag.FromErr(err)
		return
	}
	if err := data.Set("version", version); err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return
	}

	// Hash the input fields and set that as the ID
	data.SetId(strconv.FormatUint(uint64(crc32.ChecksumIEEE([]byte(moduleId+"/"+versionConstraint+"/"+strconv.FormatBool(includePrerelease)))), 10))

	return diags
}
