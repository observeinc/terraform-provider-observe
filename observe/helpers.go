package observe

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
	oid "github.com/observeinc/terraform-provider-observe/client/oid"
)

var (
	errObjectIDInvalid         = errors.New("object id is invalid")
	errNameMissing             = errors.New("name not set")
	errInputsMissing           = errors.New("no inputs defined")
	errStagesMissing           = errors.New("no stages defined")
	errInputNameMissing        = errors.New("name not set")
	errInputEmpty              = errors.New("dataset not set")
	errNameConflict            = errors.New("name already declared")
	errStageInputUnresolved    = errors.New("input could not be resolved")
	errStageInputMissing       = errors.New("input missing")
	errMoreThanOneOutputStages = errors.New("too many output stages")

	stringType = reflect.TypeOf("")

	idRegex = regexp.MustCompile(`^\d+$`)
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

func validateFilePath(extension *string) schema.SchemaValidateDiagFunc {
	return func(i interface{}, _ cty.Path) diag.Diagnostics {
		v := i.(string)
		_, err := filepath.Abs(v)
		if err != nil {
			return diag.Errorf("failed to parse as file path: %s", err)
		}
		if _, err := os.Stat(v); os.IsNotExist(err) {
			return diag.Errorf("file does not exist")
		}
		if extension != nil {
			if !strings.EqualFold(filepath.Ext(v), *extension) {
				return diag.Errorf("file must have extension %q", *extension)
			}
		}
		return nil
	}
}

func validateIsString() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) (diags diag.Diagnostics) {
		if v, ok := i.(string); !ok {
			return append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("invalid map value: %v", v),
				AttributePath: path,
			})
		}
		return diags
	}
}

// Verify OID matches type
func validateOID(types ...oid.Type) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) (diags diag.Diagnostics) {
		id, err := oid.NewOID(i.(string))
		if err != nil {
			return append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       err.Error(),
				AttributePath: path,
			})
		}
		for _, t := range types {
			if id.Type == t {
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

func validateOIDType(i interface{}, path cty.Path) (diags diag.Diagnostics) {
	t := oid.Type(i.(string))
	if !t.IsValid() {
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "invalid OID type",
			Detail:        fmt.Sprintf("invalid oid type: %s", t),
			AttributePath: path,
		})
	}

	return diags
}

const (
	CustomerIdMul          int64  = 137
	MinCustomerId          int64  = 100000000000
	MaxCustomerId          int64  = 200000000000
	MinUserId              int64  = 1000
	MaxUserId              int64  = 9999999
	InvalidObjectNameChars string = `:"\`
	MaxNameLength          int    = 127
)

func validateCID() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		v, ok := i.(string)
		if !ok {
			return diag.Errorf("expected type of customer id to be string, got %v", i)
		}
		cid, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return diag.Errorf("expected customer id to be valid integer, got %s", v)
		}
		switch {
		case cid == 101 || cid == 102 || cid == 123:
			break //valid
		case cid >= MinCustomerId && cid <= MaxCustomerId && ((cid-MinCustomerId)%CustomerIdMul == 0):
			break // valid
		default:
			return diag.Errorf("customer id %s is not valid", v)
		}
		return nil
	}
}

func validateUID() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		v, ok := i.(string)
		if !ok {
			return diag.Errorf("expected type of user id to be string, got %v", i)
		}
		// Trimming quotes, as in types.StringToUserIdScalar
		uid, err := strconv.ParseInt(strings.Trim(v, `"`), 10, 64)
		if err != nil {
			return diag.Errorf("expected user id to be valid integer, got %s", v)
		}
		if uid < MinUserId || uid > MaxUserId {
			return diag.Errorf("user id %s is not valid", v)
		}
		return nil
	}
}

func validateID() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringMatch(idRegex, "expected ID to be valid integer"))
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

func validateStringIsTimezone(i interface{}, path cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok {
		return diag.Errorf("expected type of %s to be string", i)
	}

	_, err := time.LoadLocation(v)
	if err != nil {
		return diag.Errorf("%q is not a valid timezone specifier: %s", v, err)
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

	id, err := oid.NewOID(d.Get("oid").(string))
	if err != nil || id.Version == nil {
		return false
	}

	inputs, ok := d.Get("inputs").(map[string]interface{})
	if ok {
		for _, v := range inputs {
			input, err := oid.NewOID(v.(string))
			if err == nil && input.Version != nil && *input.Version > *id.Version {
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

func diffSuppressTimeDurationZeroDistinctFromEmpty(k, prv, nxt string, d *schema.ResourceData) bool {
	o, e1 := time.ParseDuration(prv)
	n, e2 := time.ParseDuration(nxt)
	return o == n && e1 == e2 // the e1 == e2 check distinguishes "0" from ""
}

func diffSuppressJSON(k, prv, nxt string, d *schema.ResourceData) bool {
	var prvValue, nxtValue interface{}
	if err := json.Unmarshal([]byte(prv), &prvValue); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(nxt), &nxtValue); err != nil {
		return false
	}
	return reflect.DeepEqual(prvValue, nxtValue)
}

func diffSuppressStageQueryInput(k, prv, nxt string, d *schema.ResourceData) bool {
	prvValue := make([]gql.StageQueryInput, 0)
	nxtValue := make([]gql.StageQueryInput, 0)
	if err := json.Unmarshal([]byte(prv), &prvValue); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(nxt), &nxtValue); err != nil {
		return false
	}
	layoutTransform := cmp.Transformer("layoutStringToMap", func(o types.JsonObject) map[string]interface{} {
		result, _ := o.Map()
		return result
	})
	return cmp.Equal(prvValue, nxtValue, layoutTransform)
}

func diffSuppressParameters(k, prv, nxt string, d *schema.ResourceData) bool {
	prvValue := make([]gql.ParameterSpecInput, 0)
	nxtValue := make([]gql.ParameterSpecInput, 0)
	if err := json.Unmarshal([]byte(prv), &prvValue); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(nxt), &nxtValue); err != nil {
		return false
	}
	return cmp.Equal(prvValue, nxtValue)
}

func diffSuppressParameterValues(k, prv, nxt string, d *schema.ResourceData) bool {
	prvValue := make([]gql.ParameterBindingInput, 0)
	nxtValue := make([]gql.ParameterBindingInput, 0)
	if err := json.Unmarshal([]byte(prv), &prvValue); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(nxt), &nxtValue); err != nil {
		return false
	}
	return cmp.Equal(prvValue, nxtValue)
}

func diffSuppressOIDVersion(k, prv, nxt string, d *schema.ResourceData) bool {
	o, err := oid.NewOID(prv)
	if err != nil {
		return false
	}

	n, err := oid.NewOID(nxt)
	if err != nil {
		return false
	}

	return o.Type == n.Type && o.Id == n.Id
}

func diffSuppressPipeline(k, prv, nxt string, d *schema.ResourceData) bool {
	prvTrimmed := strings.TrimRightFunc(prv, unicode.IsSpace)
	nxtTrimmed := strings.TrimRightFunc(nxt, unicode.IsSpace)
	return prvTrimmed == nxtTrimmed
}

func diffSuppressWhenWorkspace(k, prv, nxt string, d *schema.ResourceData) bool {
	// If the user has specified a workspace, we suppress the diff unconditionally
	// since the folder effectively becomes a computed attribute
	if _, ok := d.GetOk("workspace"); ok {
		return true
	}
	// Otherwise, diff like normal
	return prv == nxt
}

var link = regexp.MustCompile("(^[A-Za-z])|_([A-Za-z])")

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func snakeCased(rawSlice interface{}) []string {
	var snakeCased []string

	switch reflect.TypeOf(rawSlice).Kind() {
	case reflect.Slice:
		stringsSlice := reflect.ValueOf(rawSlice)
		for i := 0; i < stringsSlice.Len(); i++ {
			stringRaw := stringsSlice.Index(i)
			var stringParsed string
			if stringer, ok := stringRaw.Interface().(fmt.Stringer); ok {
				stringParsed = stringer.String()
			} else if stringRaw.CanConvert(stringType) {
				stringParsed = stringRaw.Convert(stringType).String()
			}
			snakeCased = append(snakeCased, toSnake(stringParsed))
		}
	default:
		panic("validateEnums only accepts slice of stringers or types convertible to string")
	}

	return snakeCased
}

// take list of stringers which represent camel cased enums, compare
// against a snake case input
func validateEnums(stringerSlice interface{}) schema.SchemaValidateDiagFunc {
	return validateStringInSlice(snakeCased(stringerSlice), true)
}

func diffSuppressEnums(k, prv, nxt string, d *schema.ResourceData) bool {
	return toSnake(prv) == toSnake(nxt)
}

func describeEnums(stringerSlice interface{}, desc string) string {
	enums := snakeCased(stringerSlice)
	for i, e := range enums {
		enums[i] = fmt.Sprintf("`%s`", e)
	}
	return fmt.Sprintf("%s Accepted values: %s", desc, strings.Join(enums, ", "))
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

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func maybeString(val any, ok bool) string {
	if val == nil || !ok {
		return ""
	}
	str, is := val.(string)
	if !is {
		return ""
	}
	return str
}

func maybeOID(val any, ok bool) *oid.OID {
	ms := maybeString(val, ok)
	if ms == "" {
		return nil
	}
	ret, err := oid.NewOID(ms)
	if err != nil {
		//	It's a string but not an oid? Shouldn't have passed validation.
		panic(err)
	}
	return ret
}

func validateDatasetName() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.All(
		validation.StringLenBetween(1, MaxNameLength),
		validation.StringDoesNotContainAny(InvalidObjectNameChars),
	))
}

func validateDatastreamName() schema.SchemaValidateDiagFunc {
	return validateDatasetName()
}

func validateReferenceTableName() schema.SchemaValidateDiagFunc {
	return validateDatasetName()
}

func asPointer[T any](val T) *T {
	return &val
}

func sliceContains[T comparable](slice []T, val T) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
