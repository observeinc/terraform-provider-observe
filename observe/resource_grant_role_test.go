package observe

import (
	"testing"

	gql "github.com/observeinc/terraform-provider-observe/client/meta"
)

func TestGrantRoleOauthTokenCreator(t *testing.T) {
	got, err := OauthTokenCreator.ToRbacRole()
	if err != nil {
		t.Fatal(err)
	}
	if got != gql.RbacRoleOauthtokencreator {
		t.Fatalf("got %q, want %q", got, gql.RbacRoleOauthtokencreator)
	}
	if toSnake(string(OauthTokenCreator)) != "oauth_token_creator" {
		t.Fatalf("terraform role name = %q", toSnake(string(OauthTokenCreator)))
	}
}

func TestGrantRoleMappingCoversRbacV2Roles(t *testing.T) {
	// Editor/Viewer/Manager are expressed as dataset_editor etc. / administrator.
	// Ingester and Lister are unused in RBAC v2.
	skip := map[gql.RbacRole]struct{}{
		gql.RbacRoleEditor:   {},
		gql.RbacRoleViewer:   {},
		gql.RbacRoleManager:  {},
		gql.RbacRoleIngester: {},
		gql.RbacRoleLister:   {},
	}

	// genqlient does not generate a slice containing every enum value. Keep
	// this list synchronized with the RbacRole enum in types.generated.graphql.
	all := []gql.RbacRole{
		gql.RbacRoleApitokencreate,
		gql.RbacRoleBookmarkmanager,
		gql.RbacRoleDashboardvisibilityeditor,
		gql.RbacRoleDatasetaccelerator,
		gql.RbacRoleEditor,
		gql.RbacRoleIcebergcatalogviewer,
		gql.RbacRoleIcebergcatalogwriter,
		gql.RbacRoleIngester,
		gql.RbacRoleInvestigatorglobal,
		gql.RbacRoleLister,
		gql.RbacRoleManager,
		gql.RbacRoleMonitoractioncreator,
		gql.RbacRoleMonitorglobalmute,
		gql.RbacRoleOauthtokencreator,
		gql.RbacRoleOnlineevaluationmanager,
		gql.RbacRoleReferencetablecreator,
		gql.RbacRoleReportmanager,
		gql.RbacRoleServiceaccountcreator,
		gql.RbacRoleShareinmanager,
		gql.RbacRoleShareinviewer,
		gql.RbacRoleSkillvisibilityeditor,
		gql.RbacRoleStorageintegrationuser,
		gql.RbacRoleUserdelete,
		gql.RbacRoleUserinvite,
		gql.RbacRoleViewer,
		gql.RbacRoleWorksheetvisibilityeditor,
	}

	for _, role := range all {
		if _, skipRole := skip[role]; skipRole {
			continue
		}
		if _, ok := reverseRoleMapping[role]; !ok {
			t.Errorf("RBAC role %q is not mapped to an observe_grant role", role)
		}
	}
}
