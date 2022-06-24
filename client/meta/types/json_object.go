package types

import (
	"encoding/json"
)

type JsonObject string

func (o JsonObject) MarshalJSON() ([]byte, error) {
	return []byte(o), nil
}

func (o *JsonObject) UnmarshalJSON(data []byte) error {
	*o = JsonObject(string(data))
	return nil
}

func (o JsonObject) String() string {
	return string(o)
}

func (o JsonObject) Ptr() *JsonObject {
	return &o
}

func (o JsonObject) Map() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(o.String()), &result); err != nil {
		return nil, err
	}
	return result, nil
}
