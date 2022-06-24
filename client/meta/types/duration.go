package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type DurationScalar time.Duration

func (d DurationScalar) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%#v"`, d)), nil
}

func (d *DurationScalar) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("could not parse duration as JSON string: %w", err)
	}
	if parsed, err := ParseDurationScalar(s); err == nil {
		*d = *parsed
	} else {
		return err
	}
	return nil
}

func (d DurationScalar) String() string {
	return time.Duration(d).String()
}

func (d DurationScalar) Ptr() *DurationScalar {
	return &d
}

func ParseDurationScalar(s string) (*DurationScalar, error) {
	//preferred representation is nanoseconds-as-integer
	if parsed, err := strconv.ParseInt(s, 10, 64); err == nil {
		result := DurationScalar(time.Duration(parsed))
		return &result, nil
	} else {
		//but we can accept whatever golang can accept (3m4.001s or whatever)
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		result := DurationScalar(parsed)
		return &result, nil
	}
}
