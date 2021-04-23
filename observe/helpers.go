package observe

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

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

// Verify we were provided a valid URL path, without query parameters or bogus input
func validatePath(i interface{}, path cty.Path) (diags diag.Diagnostics) {
	v := i.(string)
	u, err := url.Parse(v)
	if err != nil {
		return diag.Errorf("failed to parse as URL: %s", err)
	}

	if u.Path != v {
		// query parameters would be overwritten by tags
		return diag.Errorf("path must not contain query parameters or other directives")
	}

	return nil
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

func validateEnum(fn func(interface{}) error) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		if err := fn(i); err != nil {
			return diag.FromErr(err)
		}
		return nil
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

func validateTimestamp(i interface{}, path cty.Path) diag.Diagnostics {
	s := i.(string)
	if _, err := time.Parse(time.RFC3339, s); err != nil {
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

func validateStringIsJSON(i interface{}, path cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok {
		return diag.Errorf("expected type of %s to be string", i)
	}

	var j interface{}
	err := json.Unmarshal([]byte(v), &j)
	if err != nil {
		return diag.Errorf("%q contains an invalid JSON: %s", v, err)
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

	inputs, ok := d.Get("inputs").(map[string]interface{})
	if ok {
		for _, v := range inputs {
			input, err := observe.NewOID(v.(string))
			if err == nil && input.Version != nil && *input.Version > *oid.Version {
				return true
			}
		}
	}
	return false
}

func diffSuppressTimeDuration(k, prv, nxt string, d *schema.ResourceData) bool {
	o, _ := time.ParseDuration(prv)
	n, _ := time.ParseDuration(nxt)
	return o == n
}

func diffSuppressJSON(k, prv, nxt string, d *schema.ResourceData) bool {
	var prvValue, nxtValue interface{}
	// no need to check for error, we've already validated inputs
	_ = json.Unmarshal([]byte(prv), &prvValue)
	_ = json.Unmarshal([]byte(nxt), &nxtValue)
	return reflect.DeepEqual(prvValue, nxtValue)
}

func diffSuppressOIDVersion(k, prv, nxt string, d *schema.ResourceData) bool {
	o, err := observe.NewOID(prv)
	if err != nil {
		return false
	}

	n, err := observe.NewOID(nxt)
	if err != nil {
		return false
	}

	return o.Type == n.Type && o.ID == n.ID
}

func diffSuppressCaseInsensitive(k, prv, nxt string, d *schema.ResourceData) bool {
	return strings.ToLower(nxt) == strings.ToLower(prv)
}

func diffSuppressPipeline(k, prv, nxt string, d *schema.ResourceData) bool {
	prvTrimmed := strings.TrimRightFunc(prv, unicode.IsSpace)
	nxtTrimmed := strings.TrimRightFunc(nxt, unicode.IsSpace)
	return prvTrimmed == nxtTrimmed
}

var link = regexp.MustCompile("(^[A-Za-z])|_([A-Za-z])")

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func snakeCased(stringerSlice interface{}) []string {
	var snakeCased []string

	switch reflect.TypeOf(stringerSlice).Kind() {
	case reflect.Slice:
		stringers := reflect.ValueOf(stringerSlice)
		for i := 0; i < stringers.Len(); i++ {
			stringer := stringers.Index(i).Interface().(fmt.Stringer)
			snakeCased = append(snakeCased, toSnake(stringer.String()))
		}
	default:
		panic("validateEnums only accepts slice of stringers")
	}

	return snakeCased
}

// take list of stringers which represent camel cased enums, compare
// against a snake case input
func validateEnums(stringerSlice interface{}) schema.SchemaValidateDiagFunc {
	return validateStringInSlice(snakeCased(stringerSlice), true)
}

func describeEnums(stringerSlice interface{}, desc string) string {
	return fmt.Sprintf("%s Accepted values: %s", desc, strings.Join(snakeCased(stringerSlice), ", "))
}

// convert to snake case
func toSnake(str string) string {
	s := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	s = matchAllCap.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}

// convert to camel case
func toCamel(str string) string {
	return link.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}
