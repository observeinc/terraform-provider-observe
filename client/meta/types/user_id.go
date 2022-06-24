package types

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type UserIdScalar int64

func (u UserIdScalar) MarshalJSON() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *UserIdScalar) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("could not parse time as JSON string: %w", err)
	}
	uid, err := strconv.ParseInt(s, 10, 64)
	if err != nil || uid < 1 {
		return fmt.Errorf("invalid userId %q: %w", s, err)
	}
	*u = UserIdScalar(uid)
	return nil
}

func (u UserIdScalar) String() string {
	return fmt.Sprintf("%q", strconv.FormatInt(int64(u), 10))
}
