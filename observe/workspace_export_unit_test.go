package observe

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

// TestExportedDataSourcesOmitWorkspace guards that the data sources used by
// export-to-Terraform leave the deprecated workspace attribute null, so
// `terraform show` omits it from the exported HCL, while the matching resources
// still record it. It inspects the raw state attributes because GetOk cannot
// tell a null attribute from one set to "".
func TestExportedDataSourcesOmitWorkspace(t *testing.T) {
	const (
		workspaceID  = "41000215"
		workspaceOID = "o:::workspace:41000215"
	)
	ctx := context.Background()

	dataset := &gql.Dataset{Id: "41000100", WorkspaceId: workspaceID, Name: "ds"}
	merge := gql.NotificationMergeMerged
	monitor := &gql.Monitor{
		Id:                  "41000101",
		WorkspaceId:         workspaceID,
		Name:                "m",
		UseDefaultFreshness: true,
		Rule:                &gql.MonitorRuleMonitorRuleCount{},
		NotificationSpec:    gql.MonitorNotificationSpecNotificationSpecification{Merge: &merge},
	}
	monitorV2 := &gql.MonitorV2{
		Id:          "41000102",
		WorkspaceId: workspaceID,
		Name:        "m2",
		RuleKind:    gql.MonitorV2RuleKindCount,
		Definition:  gql.MonitorV2Definition{LookbackTime: types.DurationScalar(30 * time.Minute).Ptr()},
	}
	action := &gql.MonitorV2Action{Id: "41000103", WorkspaceId: workspaceID, Name: "a", Type: gql.MonitorV2ActionTypeWebhook}
	worksheet := &gql.Worksheet{Id: "41000104", WorkspaceId: workspaceID, Label: "w"}
	dashboard := &gql.Dashboard{Id: "41000105", WorkspaceId: workspaceID, Name: "d"}

	cases := []struct {
		name       string
		dataSource *schema.Resource
		resource   *schema.Resource
		id         string
		fillData   func(*schema.ResourceData) diag.Diagnostics
		fillRes    func(*schema.ResourceData) diag.Diagnostics
	}{
		{
			name:       "dataset",
			dataSource: dataSourceDataset(),
			resource:   resourceDataset(),
			id:         dataset.Id,
			fillData:   func(d *schema.ResourceData) diag.Diagnostics { return datasetToResourceData(dataset, d, false) },
			fillRes:    func(d *schema.ResourceData) diag.Diagnostics { return resourceDatasetToResourceData(dataset, d, false) },
		},
		{
			name:       "monitor",
			dataSource: dataSourceMonitor(),
			resource:   resourceMonitor(),
			id:         monitor.Id,
			fillData:   func(d *schema.ResourceData) diag.Diagnostics { return monitorToResourceData(d, monitor) },
			fillRes:    func(d *schema.ResourceData) diag.Diagnostics { return resourceMonitorToResourceData(d, monitor) },
		},
		{
			name:       "monitor_v2",
			dataSource: dataSourceMonitorV2(),
			resource:   resourceMonitorV2(),
			id:         monitorV2.Id,
			fillData: func(d *schema.ResourceData) diag.Diagnostics {
				return monitorV2ToResourceData(ctx, monitorV2, d, nil, true)
			},
			fillRes: func(d *schema.ResourceData) diag.Diagnostics {
				return resourceMonitorV2ToResourceData(ctx, monitorV2, d, nil)
			},
		},
		{
			name:       "monitor_v2_action",
			dataSource: dataSourceMonitorV2Action(),
			resource:   resourceMonitorV2Action(),
			id:         action.Id,
			fillData:   func(d *schema.ResourceData) diag.Diagnostics { return monitorV2ActionToResourceData(action, d) },
			// resourceMonitorV2ActionRead fetches the action itself, so only the
			// data-source path is exercised here; the acceptance tests cover the resource.
		},
		{
			name:       "worksheet",
			dataSource: dataSourceWorksheet(),
			resource:   resourceWorksheet(),
			id:         worksheet.Id,
			fillData:   func(d *schema.ResourceData) diag.Diagnostics { return worksheetToResourceData(worksheet, d) },
			fillRes:    func(d *schema.ResourceData) diag.Diagnostics { return resourceWorksheetToResourceData(worksheet, d) },
		},
		{
			name:       "dashboard",
			dataSource: dataSourceDashboard(),
			resource:   resourceDashboard(),
			id:         dashboard.Id,
			fillData:   func(d *schema.ResourceData) diag.Diagnostics { return dashboardToResourceData(dashboard, d) },
			fillRes:    func(d *schema.ResourceData) diag.Diagnostics { return resourceDashboardToResourceData(dashboard, d) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := schema.TestResourceDataRaw(t, tc.dataSource.Schema, map[string]interface{}{"id": tc.id})
			ds.SetId(tc.id)
			if diags := tc.fillData(ds); diags.HasError() {
				t.Fatalf("data source fill: %v", diags)
			}
			if v, ok := ds.State().Attributes["workspace"]; ok {
				t.Errorf("data source state workspace = %q, want absent (null)", v)
			}

			if tc.fillRes == nil {
				return
			}
			res := schema.TestResourceDataRaw(t, tc.resource.Schema, map[string]interface{}{})
			res.SetId(tc.id)
			if diags := tc.fillRes(res); diags.HasError() {
				t.Fatalf("resource fill: %v", diags)
			}
			if got := res.State().Attributes["workspace"]; got != workspaceOID {
				t.Errorf("resource state workspace = %q, want %q", got, workspaceOID)
			}
		})
	}
}
