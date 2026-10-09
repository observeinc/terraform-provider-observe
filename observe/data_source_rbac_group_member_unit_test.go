package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

func TestRbacGroupmemberDataToResourceDataMapsUserMember(t *testing.T) {
	data := schema.TestResourceDataRaw(t, dataSourceRbacGroupmember().Schema, map[string]interface{}{})

	userId := types.UserIdScalar(123)
	diags := rbacGroupmemberDataToResourceData(&gql.RbacGroupmember{
		Id:           "1",
		Description:  "example",
		GroupId:      "41030001",
		MemberUserId: &userId,
	}, data)

	if diags.HasError() {
		t.Fatalf("map rbacgroupmember: %v", diags)
	}
	if got, want := data.Id(), "1"; got != want {
		t.Errorf("id = %q, want %q", got, want)
	}
	if got := data.Get("description"); got != "example" {
		t.Errorf("description = %q, want %q", got, "example")
	}
	if got, want := data.Get("group"), "o:::rbacgroup:41030001"; got != want {
		t.Errorf("group = %q, want %q", got, want)
	}
	if got := data.Get("member.#"); got != 1 {
		t.Errorf("member.# = %v, want 1", got)
	}
	if got, want := data.Get("member.0.user"), "o:::user:123"; got != want {
		t.Errorf("member.0.user = %q, want %q", got, want)
	}
	if got := data.Get("member.0.group"); got != "" {
		t.Errorf("member.0.group = %q, want unset", got)
	}
}

func TestRbacGroupmemberDataToResourceDataMapsGroupMember(t *testing.T) {
	data := schema.TestResourceDataRaw(t, dataSourceRbacGroupmember().Schema, map[string]interface{}{})

	diags := rbacGroupmemberDataToResourceData(&gql.RbacGroupmember{
		Id:            "2",
		GroupId:       "41030001",
		MemberGroupId: strPtr("41030002"),
	}, data)

	if diags.HasError() {
		t.Fatalf("map rbacgroupmember: %v", diags)
	}
	if got, want := data.Get("member.0.group"), "o:::rbacgroup:41030002"; got != want {
		t.Errorf("member.0.group = %q, want %q", got, want)
	}
	if got := data.Get("member.0.user"); got != "" {
		t.Errorf("member.0.user = %q, want unset", got)
	}
}
