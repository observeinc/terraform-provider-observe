package oid

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func strptr(s string) *string {
	return &s
}

func TestOid(t *testing.T) {
	t.Skip()

	testcases := []struct {
		Input  string
		Expect *OID
	}{
		{
			Input: "o:::dataset:123456",
			Expect: &OID{
				Type: TypeDataset,
				Id:   "123456",
			},
		},
		{
			Input: "o:::dataset:123456/2020-01-16T21:06:19Z",
			Expect: &OID{
				Type:    TypeDataset,
				Id:      "123456",
				Version: strptr("2020-01-16T21:06:19Z"),
			},
		},
		{
			Input: "o:::workspace:123458",
			Expect: &OID{
				Type: TypeWorkspace,
				Id:   "123458",
			},
		},
		{
			Input: "o:::rbacgroup:o::123458:rbacgroup:8000002523",
			Expect: &OID{
				Type: TypeRbacGroup,
				Id:   "o::123458:rbacgroup:8000002523",
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

func TestExtractORN(t *testing.T) {
	t.Skip()
	testcases := []struct {
		InputOID  string
		OutputOID string
		OutputORN string
	}{
		{
			InputOID:  "o:::rbacgroup:o::120180709924:rbacgroup:8000002523",
			OutputOID: "o:::rbacgroup:8000002523",
			OutputORN: "o::120180709924:rbacgroup:8000002523",
		},
		{
			InputOID:  "o:::rbacgroup:o::120180709924:rbacgroup:8000002523/2020-01-16T21:06:19Z",
			OutputOID: "o:::rbacgroup:8000002523/2020-01-16T21:06:19Z",
			OutputORN: "o::120180709924:rbacgroup:8000002523",
		},
		{
			InputOID:  "o:::datastream:123458",
			OutputOID: "o:::datastream:123458",
		},
	}
	for _, tt := range testcases {
		orn, oid := extractORN(tt.InputOID)

		if orn != tt.OutputORN {
			t.Errorf("Invalid ORN. expected: %s got: %s", tt.OutputORN, orn)
		}
		if oid != tt.OutputOID {
			t.Errorf("Invalid OID. expected: %s got: %s", tt.OutputOID, oid)
		}
	}
}
