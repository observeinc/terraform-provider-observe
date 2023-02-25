package types

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Value is a GraphQL type representing a value that may be one of many supported types (ValueType).
// When represented as JSON, exactly one field must be set.
// A null value of a given type is represented with null value in JSON, e.g.,: {"bool": null}
type Value struct {
	// ValueData holds the concrete value in one field based on the given type
	ValueData

	// NullValueType stores the type of a Value that is null, since this cannot be
	// inferred from the other fields, which will all be nil.
	NullValueType ValueType `json:"-"`
}

// ValueData holds a concrete non-null value of one of the supported types.
type ValueData struct {
	Bool       *bool            `json:"bool,omitempty"`
	Float64    *float64         `json:"float64,omitempty"`
	Int64      *Int64Scalar     `json:"int64,omitempty"`
	String     *string          `json:"string,omitempty"`
	Timestamp  *TimeScalar      `json:"timestamp,omitempty"`
	Duration   *DurationScalar  `json:"duration,omitempty"`
	Array      *ValueArray      `json:"array,omitempty"`
	Link       *ValueLink       `json:"link,omitempty"`
	Datasetref *ValueDatasetref `json:"datasetref,omitempty"`
}

// NewNullValue returns a Value that is null of the given type.
func NewNullValue(t ValueType) *Value {
	return &Value{NullValueType: t}
}

// MustNewValue returns a Value for the given Go type. It panics if the type is not supported.
func MustNewValue(v any) *Value {
	if reflect.TypeOf(v).Kind() == reflect.Slice {
		slice := reflect.ValueOf(v)
		array := ValueArray{}

		for i := 0; i < slice.Len(); i++ {
			array.Value = append(array.Value, MustNewValue(slice.Index(i).Interface()))
		}

		return &Value{ValueData: ValueData{Array: &array}}
	}

	switch v := v.(type) {
	case bool:
		return &Value{ValueData: ValueData{Bool: &v}}
	case float64:
		return &Value{ValueData: ValueData{Float64: &v}}
	case int64:
		i := Int64Scalar(v)
		return &Value{ValueData: ValueData{Int64: &i}}
	case string:
		return &Value{ValueData: ValueData{String: &v}}
	case TimeScalar:
		return &Value{ValueData: ValueData{Timestamp: &v}}
	case DurationScalar:
		return &Value{ValueData: ValueData{Duration: &v}}
	case ValueLink:
		return &Value{ValueData: ValueData{Link: &v}}
	case ValueDatasetref:
		return &Value{ValueData: ValueData{Datasetref: &v}}
	default:
		panic(fmt.Sprintf("unsupported type: %T", v))
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It handles null values of a given type, e.g.,: {"bool": null}
// It also strips extra nulls for backwards compatibility:
// {"bool": true, "float64": null, ...}
//
// It rejects values that have more than one field set or where all fields are null.
func (v *Value) UnmarshalJSON(b []byte) error {
	var obj map[ValueType]*json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("failed to unmarshal value from JSON: %w", err)
	}

	valueType, err := ValueTypeOf(obj)
	if err != nil {
		return fmt.Errorf("failed to determine value type: %w", err)
	}

	value := obj[valueType]
	if value == nil {
		v.NullValueType = valueType
		return nil
	}

	return json.Unmarshal(b, &v.ValueData)
}

// MarshalJSON implements the json.Marshaler interface.
// If the value is null, it returns a null value for the given type, e.g.,: {"bool": null}
// Otherwise, it returns the value in the corresponding field.
func (v Value) MarshalJSON() ([]byte, error) {
	if v.NullValueType != "" {
		return json.Marshal(map[ValueType]any{v.NullValueType: nil})
	}

	return json.Marshal(v.ValueData)
}

// ValueType is an supported type of a Value.
type ValueType string

const (
	ValueTypeBool       ValueType = "bool"
	ValueTypeFloat64    ValueType = "float64"
	ValueTypeInt64      ValueType = "int64"
	ValueTypeString     ValueType = "string"
	ValueTypeTimestamp  ValueType = "timestamp"
	ValueTypeDuration   ValueType = "duration"
	ValueTypeArray      ValueType = "array"
	ValueTypeObject     ValueType = "object"
	ValueTypeLink       ValueType = "link"
	ValueTypeDatasetref ValueType = "datasetref"
)

type ValueArray struct {
	Value []*Value `json:"value,omitempty"`
}

type ValueLink struct {
	DatasetId       string           `json:"datasetId,omitempty"`
	PrimaryKeyValue []*ValueKeyValue `json:"primaryKeyValue,omitempty"`
	StoredLabel     *string          `json:"storedLabel,omitempty"`
}

type ValueKeyValue struct {
	Name  string `json:"name,omitempty"`
	Value *Value `json:"value,omitempty"`
}

type ValueDatasetref struct {
	DatasetId   *string `json:"datasetId,omitempty"`
	DatasetPath *string `json:"datasetPath,omitempty"`
	StageId     *string `json:"stageId,omitempty"`
}

// ValueTypeOf is a generic function that returns the ValueType of a map based on its non-nil values.
// It returns an error if too many or too few values are set.
func ValueTypeOf[V any](m map[ValueType]*V) (ValueType, error) {
	if len(m) == 1 {
		for k := range m {
			return k, nil
		}
	}

	var types []ValueType
	for k, v := range m {
		if v != nil {
			types = append(types, k)
		}
	}

	if len(types) == 0 {
		return "", fmt.Errorf("expected at least one value type to be set, got 0")
	}

	if len(types) > 1 {
		return "", fmt.Errorf("expected exactly one value type to be set, got %d: %v", len(types), types)
	}

	return types[0], nil
}
