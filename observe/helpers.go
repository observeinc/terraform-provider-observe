package observe

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	observe "github.com/observeinc/terraform-provider-observe/client"
)

// apply ValidateDiagFunc to every value in map
func validateMapValues(fn schema.SchemaValidateDiagFunc) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) (diags diag.Diagnostics) {
		for k, v := range i.(map[string]interface{}) {
			diags = append(diags, fn(v, path.IndexString(k))...)
		}
		return diags
	}
}

// Verify OID matches type
func validateOID(types ...observe.Type) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) (diags diag.Diagnostics) {
		oid, err := observe.NewOID(i.(string))
		if err != nil {
			return append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       err.Error(),
				AttributePath: path,
			})
		}
		for _, t := range types {
			if oid.Type == t {
				return diags
			}
		}
		if len(types) > 0 {
			var s []string
			for _, t := range types {
				s = append(s, string(t))
			}
			return append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "wrong type",
				Detail:        fmt.Sprintf("oid type must be %s", strings.Join(s, ", ")),
				AttributePath: path,
			})
		}
		return
	}
}

func validateStringInSlice(valid []string, ignoreCase bool) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		v, ok := i.(string)
		if !ok {
			return diag.Errorf("expected type of %s to be string", i)
		}

		for _, str := range valid {
			if v == str || (ignoreCase && strings.ToLower(v) == strings.ToLower(str)) {
				return nil
			}
		}

		return diag.Errorf("expected %s to be one of %v, got %s", i, valid, v)
	}
}

func validateTimeDuration(i interface{}, path cty.Path) diag.Diagnostics {
	s := i.(string)
	if _, err := time.ParseDuration(s); err != nil {
		return diag.Diagnostics{diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Invalid field",
			Detail:        err.Error(),
			AttributePath: path,
		}}
	}
	return nil
}

func validateFlags(i interface{}, path cty.Path) diag.Diagnostics {
	if _, err := convertFlags(i.(string)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// input is a list of comma separated lowercase identifiers which may be negated, e.g. "feature-a,!enable-b"
func convertFlags(s string) (map[string]bool, error) {
	flags := make(map[string]bool)
	if s == "" {
		return flags, nil
	}

	pattern := `^\!?[a-z][0-9a-z-]*$`
	isFlag := regexp.MustCompile(pattern).MatchString

	for _, substr := range strings.Split(s, ",") {
		if !isFlag(substr) {
			return nil, fmt.Errorf("flag %q not match regexp %q", substr, pattern)
		}

		key := strings.Trim(substr, "!")
		value := substr[0] != byte('!')
		flags[key] = value
	}

	return flags, nil
}

// determine whether we expect a new dataset version
func datasetRecomputeOID(d *schema.ResourceDiff) bool {
	if len(d.GetChangedKeysPrefix("")) > 0 {
		return true
	}

	oid, err := observe.NewOID(d.Get("oid").(string))
	if err != nil || oid.Version == nil {
		return false
	}

	for _, v := range d.Get("inputs").(map[string]interface{}) {
		input, err := observe.NewOID(v.(string))
		if err == nil && input.Version != nil && *input.Version > *oid.Version {
			return true
		}
	}
	return false
}

func diffSuppressTimeDuration(k, old, new string, d *schema.ResourceData) bool {
	o, _ := time.ParseDuration(old)
	n, _ := time.ParseDuration(new)
	return o == n
}
