package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Int64Scalar int64

func (n Int64Scalar) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf(`"%d"`, int64(n))
	return []byte(s), nil
}

func (n *Int64Scalar) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("could not parse time as JSON string: %w", err)
	}
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid int64 %q: %w", s, err)
	}
	*n = Int64Scalar(parsed)
	return nil
}

func (n Int64Scalar) String() string {
	return strconv.FormatInt(int64(n), 10)
}

func (n Int64Scalar) Duration() time.Duration {
	return time.Duration(n)
}

func (n Int64Scalar) Ptr() *Int64Scalar {
	return &n
}

func (n *Int64Scalar) IntPtr() *int {
	if n == nil {
		return nil
	} else {
		result := int(*n)
		return &result
	}
}
