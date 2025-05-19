package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type UserIdScalar int64

func (u UserIdScalar) MarshalJSON() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *UserIdScalar) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("could not parse userId as JSON string: %w", err)
	}
	uid, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid userId %q: %w", s, err)
	}
	*u = UserIdScalar(uid)
	return nil
}

func (u UserIdScalar) String() string {
	return fmt.Sprintf("%q", strconv.FormatInt(int64(u), 10))
}

func StringToUserIdScalar(id string) (UserIdScalar, error) {
	// TODO: This won't be required if UserIdScalar is mapped to a string in genqlient.yaml.
	// Based on current configuration:
	// `id` read from the api is json encoded.
	// `id` supplied via tf manifest may be an unquoted string.
	// sanitize before converting into a UserIdScalar
	id = strconv.Quote(strings.Trim(id, "\""))

	var uid UserIdScalar
	if err := uid.UnmarshalJSON([]byte(id)); err != nil {
		return uid, err
	}
	return uid, nil
}
