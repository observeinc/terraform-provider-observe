package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestValueJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		json  string
		value *Value
	}{
		{
			name:  "bool",
			json:  `{"bool":true}`,
			value: MustNewValue(true),
		},
		{
			name:  "null bool",
			json:  `{"bool":null}`,
			value: NewNullValue(ValueTypeBool),
		},
		{
			name:  "float64",
			json:  `{"float64":1.1}`,
			value: MustNewValue(1.1),
		},
		{
			name:  "null float64",
			json:  `{"float64":null}`,
			value: NewNullValue(ValueTypeFloat64),
		},
		{
			name:  "int64",
			json:  `{"int64":1}`,
			value: MustNewValue(int64(1)),
		},
		{
			name:  "null int64",
			json:  `{"int64":null}`,
			value: NewNullValue(ValueTypeInt64),
		},
		{
			name:  "string",
			json:  `{"string":"foo"}`,
			value: MustNewValue("foo"),
		},
		{
			name:  "null string",
			json:  `{"string":null}`,
			value: NewNullValue(ValueTypeString),
		},
		{
			name:  "timestamp",
			json:  `{"timestamp":"0001-01-01T00:00:00Z"}`,
			value: MustNewValue(TimeScalar(time.Time{})),
		},
		{
			name:  "null timestamp",
			json:  `{"timestamp":null}`,
			value: NewNullValue(ValueTypeTimestamp),
		},
		{
			name:  "duration",
			json:  `{"duration":"1000000000"}`,
			value: MustNewValue(DurationScalar(time.Second)),
		},
		{
			name:  "null duration",
			json:  `{"duration":null}`,
			value: NewNullValue(ValueTypeDuration),
		},
		{
			name:  "array",
			json:  `{"array":{"value":[{"string":"foo"},{"string":"bar"}]}}`,
			value: MustNewValue([]string{"foo", "bar"}),
		},
		{
			name:  "null array",
			json:  `{"array":null}`,
			value: NewNullValue(ValueTypeArray),
		},
		{
			name: "link",
			json: `{"link":{"datasetId":"123456","primaryKeyValue":[{"name":"foo","value":{"string":"bar"}}],"storedLabel":"l"}}`,
			value: MustNewValue(ValueLink{
				DatasetId: "123456",
				PrimaryKeyValue: []*ValueKeyValue{
					{Name: "foo", Value: MustNewValue("bar")},
				},
				StoredLabel: MustNewValue("l").String,
			}),
		},
		{
			name:  "null link",
			json:  `{"link":null}`,
			value: NewNullValue(ValueTypeLink),
		},
		{
			name: "datasetref",
			json: `{"datasetref":{"datasetId":"123456","datasetPath":"foo","stageId":"stage-f3fb2657"}}`,
			value: MustNewValue(ValueDatasetref{
				DatasetId:   MustNewValue("123456").String,
				DatasetPath: MustNewValue("foo").String,
				StageId:     MustNewValue("stage-f3fb2657").String,
			}),
		},
		{
			name:  "null datasetref",
			json:  `{"datasetref":null}`,
			value: NewNullValue(ValueTypeDatasetref),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				var got Value
				if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
					t.Fatal(err)
				}

				cmpopts := []cmp.Option{
					cmp.Comparer(func(x, y TimeScalar) bool {
						return time.Time(x).Equal(time.Time(y))
					}),
				}

				if diff := cmp.Diff(got, *tc.value, cmpopts...); diff != "" {
					t.Errorf("unexpected unmarshaled value -want/+got: %s", diff)
				}
			})

			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				got, err := json.Marshal(tc.value)
				if err != nil {
					t.Fatal(err)
				}

				if string(got) != tc.json {
					t.Errorf("unexpected marshaled value -want/+got: %s", cmp.Diff(tc.json, string(got)))
				}
			})
		})
	}
}

func TestValueUnmarshalJSONIgnoreNulls(t *testing.T) {
	value := Value{}
	if err := json.Unmarshal([]byte(`{"bool":null,"string":"foo"}`), &value); err != nil {
		t.Fatal(err)
	}

	if value.Bool != nil {
		t.Errorf("expected bool to be nil, got %v", value.Bool)
	}

	if vt := value.NullValueType; vt != "" {
		t.Errorf("expected null value type to be empty, got %q", vt)
	}

	expected := "foo"
	if s := *value.String; s != expected {
		t.Errorf("expected string to be %q, got %q", expected, s)
	}
}

func TestValueUnmarshalJSONErrors(t *testing.T) {
	cases := []struct {
		name  string
		json  string
		error string
	}{
		{
			name:  "invalid type",
			json:  `123`,
			error: "failed to unmarshal",
		},
		{
			name:  "too few",
			json:  `{}`,
			error: "expected at least one value type",
		},
		{
			name:  "too many",
			json:  `{"bool":true,"float64":1.1}`,
			error: "expected exactly one value type",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got Value
			err := json.Unmarshal([]byte(tc.json), &got)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tc.error) {
				t.Errorf("unexpected error to contain %q, got %q", tc.error, err.Error())
			}
		})
	}
}
