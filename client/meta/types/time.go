package types

import (
	"encoding/json"
	"fmt"
	"time"
)

type TimeScalar time.Time

const timeScalarFmt = time.RFC3339 // all characters in timeScalarFmt should be ASCII characters

func (t TimeScalar) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(timeScalarFmt)+10+2)
	b = append(b, '"')
	b = append(b, []byte(t.String())...)
	b = append(b, '"')
	return b, nil
}

func (t *TimeScalar) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("could not parse time as JSON string: %w", err)
	}
	tm, err := time.Parse(timeScalarFmt, s)
	if err != nil {
		return fmt.Errorf("invalid time %q: %w", s, err)
	}
	*t = TimeScalar(tm)
	return nil
}

func (t TimeScalar) String() string {
	return time.Time(t).Format(timeScalarFmt)
}
