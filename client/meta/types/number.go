package types

import (
	"encoding/json"
	"fmt"
)

type NumberScalar float64

func (n NumberScalar) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.String())
	// TODO: restore when API correctly unmarshals JSON numbers
	// return []byte(n.String()), nil
}

func (n *NumberScalar) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("could not parse JSON number: %w", err)
	}
	*n = NumberScalar(f)
	return nil
}

func (n NumberScalar) String() string {
	return fmt.Sprintf("%f", n)
}
