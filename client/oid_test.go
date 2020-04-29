package client

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func strptr(s string) *string {
	return &s
}

func TestOid(t *testing.T) {

	testcases := []struct {
		Input  string
		Expect *OID
	}{
		{
			Input: "o:::dataset:123456",
			Expect: &OID{
				Type: TypeDataset,
				ID:   "123456",
			},
		},
		{
			Input: "o:::dataset:123456/2020-01-16T21:06:19Z",
			Expect: &OID{
				Type:    TypeDataset,
				ID:      "123456",
				Version: strptr("2020-01-16T21:06:19Z"),
			},
		},
		{
			Input: "o:::workspace:123458",
			Expect: &OID{
				Type: TypeWorkspace,
				ID:   "123458",
			},
		},
	}

	for _, tt := range testcases {

		result, err := NewOID(tt.Input)
		if err != nil {
			t.Fatal(err)
		}

		if s := cmp.Diff(result, tt.Expect); s != "" {
			t.Fatalf("OID parsing does not match: %s", s)
		}

		if s := cmp.Diff(result.String(), tt.Input); s != "" {
			t.Fatalf("OID string does not match input: %s", s)
		}

	}
}
