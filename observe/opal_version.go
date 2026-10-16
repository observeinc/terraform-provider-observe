package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

var allOpalVersions = []gql.OpalVersion{
	gql.OpalVersionV20210101,
	gql.OpalVersionV20260930,
}

func opalVersionPtr(version gql.OpalVersion) *gql.OpalVersion {
	return &version
}

func opalVersionStrings() []string {
	values := make([]string, len(allOpalVersions))
	for i, version := range allOpalVersions {
		values[i] = string(version)
	}
	return values
}

func validateOpalVersion() schema.SchemaValidateDiagFunc {
	// The API enum is lowercase. validateEnums is case-insensitive and would
	// accept V2026_09_30, which then fails at apply or plans forever.
	return validateStringInSlice(opalVersionStrings(), false)
}

func opalVersionSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ValidateDiagFunc: validateOpalVersion(),
		Description:      descriptions.Get("common", "schema", "opal_version"),
	}
}

func opalVersionDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: descriptions.Get("common", "schema", "opal_version"),
	}
}

func customizeOpalVersionDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}, next schema.CustomizeDiffFunc) error {
	// Set the provider default before next. Dataset and log-derived metric dry
	// runs, and datasetRecomputeOID, read the diff inside next. Setting the
	// value afterward leaves an opal_version-only plan unvalidated and the
	// versioned oid unchanged.
	if client, ok := meta.(*observeclient.Client); ok {
		if defaultVersion := clientDefaultOpalVersion(client); defaultVersion != nil && *defaultVersion != "" {
			if _, configured := configuredOpalVersion(diff); !configured {
				_, planned := diff.GetChange("opal_version")
				if planned != *defaultVersion {
					if err := diff.SetNew("opal_version", *defaultVersion); err != nil {
						return err
					}
				}
			}
		}
	}

	if next != nil {
		return next(ctx, diff, meta)
	}
	return nil
}

func configuredOpalVersion(data ResourceReader) (*gql.OpalVersion, bool) {
	raw, ok := data.(rawConfigGetter)
	if !ok {
		return nil, false
	}

	config := raw.GetRawConfig()
	if !config.IsKnown() {
		return nil, true
	}
	if config.IsNull() {
		return nil, false
	}

	value, exists := config.AsValueMap()["opal_version"]
	if !exists || value.IsNull() {
		return nil, false
	}
	if !value.IsKnown() || value.AsString() == "" {
		return nil, true
	}

	version := gql.OpalVersion(value.AsString())
	return &version, true
}

// resolveOpalVersion returns the resource-level pin when explicitly configured,
// then the provider default, then the version read into state. The state fallback
// prevents full-replacement APIs from resetting an existing object to the
// earliest OPAL version when neither configuration level specifies a pin.
func resolveOpalVersion(defaultVersion *string, data ResourceReader) *gql.OpalVersion {
	if version, configured := configuredOpalVersion(data); configured {
		return version
	}

	if defaultVersion != nil && *defaultVersion != "" {
		version := gql.OpalVersion(*defaultVersion)
		return &version
	}

	if value, ok := data.GetOk("opal_version"); ok {
		version := gql.OpalVersion(value.(string))
		return &version
	}

	return nil
}

func resolveOpalVersionString(defaultVersion *string, data ResourceReader) *string {
	version := resolveOpalVersion(defaultVersion, data)
	if version == nil {
		return nil
	}
	value := string(*version)
	return &value
}

func clientDefaultOpalVersion(client *observeclient.Client) *string {
	if client == nil || client.Config == nil {
		return nil
	}
	return client.DefaultOpalVersion
}

func setOpalVersion(data *schema.ResourceData, version *gql.OpalVersion) error {
	if version == nil {
		return nil
	}
	return data.Set("opal_version", string(*version))
}
