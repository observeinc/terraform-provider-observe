package observe

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/meta/types"
)

func ptrString(s string) *string { return &s }

func ptrInt64Scalar(v int64) *types.Int64Scalar {
	s := types.Int64Scalar(v)
	return &s
}

func TestMonitorV2FlattenAnomalyRule(t *testing.T) {
	testcases := []struct {
		name     string
		input    gql.MonitorV2AnomalyRule
		expected []interface{}
	}{
		{
			name:  "empty anomaly rule",
			input: gql.MonitorV2AnomalyRule{},
			expected: []interface{}{
				map[string]interface{}{},
			},
		},
		{
			name: "with compare_percentage only",
			input: gql.MonitorV2AnomalyRule{
				ComparePercentage: ptrInt64Scalar(50),
			},
			expected: []interface{}{
				map[string]interface{}{
					"compare_percentage": 50,
				},
			},
		},
		{
			name: "with compare_percentage zero",
			input: gql.MonitorV2AnomalyRule{
				ComparePercentage: ptrInt64Scalar(0),
			},
			expected: []interface{}{
				map[string]interface{}{
					"compare_percentage": 0,
				},
			},
		},
		{
			name: "with compare_percentage 100",
			input: gql.MonitorV2AnomalyRule{
				ComparePercentage: ptrInt64Scalar(100),
			},
			expected: []interface{}{
				map[string]interface{}{
					"compare_percentage": 100,
				},
			},
		},
		{
			name: "with compare_groups",
			input: gql.MonitorV2AnomalyRule{
				CompareGroups: []gql.MonitorV2ColumnComparison{
					{
						Column: gql.MonitorV2Column{
							ColumnPath: &gql.MonitorV2ColumnPath{
								Name: "groupcol",
							},
						},
						CompareValues: []gql.MonitorV2Comparison{
							{
								CompareFn: gql.MonitorV2ComparisonFunctionEqual,
								CompareValue: gql.PrimitiveValue{
									String: ptrString("testval"),
								},
							},
						},
					},
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"compare_groups": []interface{}{
						map[string]interface{}{
							"column": []interface{}{
								map[string]interface{}{
									"column_path": []interface{}{
										map[string]interface{}{
											"name": "groupcol",
										},
									},
								},
							},
							"compare_values": []interface{}{
								map[string]interface{}{
									"compare_fn":      "equal",
									"value_bool":      []interface{}{},
									"value_duration":  []interface{}{},
									"value_float64":   []interface{}{},
									"value_int64":     []interface{}{},
									"value_string":    []interface{}{"testval"},
									"value_timestamp": []interface{}{},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "with both compare_percentage and compare_groups",
			input: gql.MonitorV2AnomalyRule{
				ComparePercentage: ptrInt64Scalar(75),
				CompareGroups: []gql.MonitorV2ColumnComparison{
					{
						Column: gql.MonitorV2Column{
							ColumnPath: &gql.MonitorV2ColumnPath{
								Name: "region",
							},
						},
						CompareValues: []gql.MonitorV2Comparison{
							{
								CompareFn: gql.MonitorV2ComparisonFunctionEqual,
								CompareValue: gql.PrimitiveValue{
									String: ptrString("us-west-2"),
								},
							},
						},
					},
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"compare_percentage": 75,
					"compare_groups": []interface{}{
						map[string]interface{}{
							"column": []interface{}{
								map[string]interface{}{
									"column_path": []interface{}{
										map[string]interface{}{
											"name": "region",
										},
									},
								},
							},
							"compare_values": []interface{}{
								map[string]interface{}{
									"compare_fn":      "equal",
									"value_bool":      []interface{}{},
									"value_duration":  []interface{}{},
									"value_float64":   []interface{}{},
									"value_int64":     []interface{}{},
									"value_string":    []interface{}{"us-west-2"},
									"value_timestamp": []interface{}{},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := monitorV2FlattenAnomalyRule(tt.input)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("monitorV2FlattenAnomalyRule() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestMonitorV2FlattenNoDataRuleWithAnomaly(t *testing.T) {
	thirtyMin := types.DurationScalar(30 * time.Minute)

	testcases := []struct {
		name     string
		input    gql.MonitorV2NoDataRule
		expected interface{}
	}{
		{
			name:     "empty no_data_rule",
			input:    gql.MonitorV2NoDataRule{},
			expected: map[string]interface{}{},
		},
		{
			name: "with expiration only",
			input: gql.MonitorV2NoDataRule{
				Expiration: &thirtyMin,
			},
			expected: map[string]interface{}{
				"expiration": "30m0s",
			},
		},
		{
			name: "with anomaly empty",
			input: gql.MonitorV2NoDataRule{
				Expiration: &thirtyMin,
				Anomaly:    &gql.MonitorV2AnomalyRule{},
			},
			expected: map[string]interface{}{
				"expiration": "30m0s",
				"anomaly": []interface{}{
					map[string]interface{}{},
				},
			},
		},
		{
			name: "with anomaly and compare_percentage",
			input: gql.MonitorV2NoDataRule{
				Expiration: &thirtyMin,
				Anomaly: &gql.MonitorV2AnomalyRule{
					ComparePercentage: ptrInt64Scalar(50),
				},
			},
			expected: map[string]interface{}{
				"expiration": "30m0s",
				"anomaly": []interface{}{
					map[string]interface{}{
						"compare_percentage": 50,
					},
				},
			},
		},
		{
			name: "anomaly without expiration",
			input: gql.MonitorV2NoDataRule{
				Anomaly: &gql.MonitorV2AnomalyRule{
					ComparePercentage: ptrInt64Scalar(80),
				},
			},
			expected: map[string]interface{}{
				"anomaly": []interface{}{
					map[string]interface{}{
						"compare_percentage": 80,
					},
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := monitorV2FlattenNoDataRule(tt.input)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("monitorV2FlattenNoDataRule() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestMonitorV2FlattenNoDataRulesAnomaly(t *testing.T) {
	thirtyMin := types.DurationScalar(30 * time.Minute)

	testcases := []struct {
		name     string
		input    []gql.MonitorV2NoDataRule
		expected []interface{}
	}{
		{
			name:     "nil rules",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty rules",
			input:    []gql.MonitorV2NoDataRule{},
			expected: nil,
		},
		{
			name: "single anomaly rule",
			input: []gql.MonitorV2NoDataRule{
				{
					Expiration: &thirtyMin,
					Anomaly: &gql.MonitorV2AnomalyRule{
						ComparePercentage: ptrInt64Scalar(90),
					},
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"expiration": "30m0s",
					"anomaly": []interface{}{
						map[string]interface{}{
							"compare_percentage": 90,
						},
					},
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := monitorV2FlattenNoDataRules(tt.input)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("monitorV2FlattenNoDataRules() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestMonitorV2FlattenRuleWithAnomaly(t *testing.T) {
	testcases := []struct {
		name     string
		input    gql.MonitorV2Rule
		expected interface{}
	}{
		{
			name: "anomaly rule with compare_percentage",
			input: gql.MonitorV2Rule{
				Level: gql.MonitorV2AlarmLevelCritical,
				Anomaly: &gql.MonitorV2AnomalyRule{
					ComparePercentage: ptrInt64Scalar(50),
				},
			},
			expected: map[string]interface{}{
				"level": "critical",
				"anomaly": []interface{}{
					map[string]interface{}{
						"compare_percentage": 50,
					},
				},
			},
		},
		{
			name: "anomaly rule with compare_groups",
			input: gql.MonitorV2Rule{
				Level: gql.MonitorV2AlarmLevelWarning,
				Anomaly: &gql.MonitorV2AnomalyRule{
					ComparePercentage: ptrInt64Scalar(80),
					CompareGroups: []gql.MonitorV2ColumnComparison{
						{
							Column: gql.MonitorV2Column{
								ColumnPath: &gql.MonitorV2ColumnPath{
									Name: "service",
								},
							},
							CompareValues: []gql.MonitorV2Comparison{
								{
									CompareFn: gql.MonitorV2ComparisonFunctionEqual,
									CompareValue: gql.PrimitiveValue{
										String: ptrString("web"),
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"level": "warning",
				"anomaly": []interface{}{
					map[string]interface{}{
						"compare_percentage": 80,
						"compare_groups": []interface{}{
							map[string]interface{}{
								"column": []interface{}{
									map[string]interface{}{
										"column_path": []interface{}{
											map[string]interface{}{
												"name": "service",
											},
										},
									},
								},
								"compare_values": []interface{}{
									map[string]interface{}{
										"compare_fn":      "equal",
										"value_bool":      []interface{}{},
										"value_duration":  []interface{}{},
										"value_float64":   []interface{}{},
										"value_int64":     []interface{}{},
										"value_string":    []interface{}{"web"},
										"value_timestamp": []interface{}{},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "anomaly rule empty (all fields optional)",
			input: gql.MonitorV2Rule{
				Level:   gql.MonitorV2AlarmLevelInformational,
				Anomaly: &gql.MonitorV2AnomalyRule{},
			},
			expected: map[string]interface{}{
				"level": "informational",
				"anomaly": []interface{}{
					map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := monitorV2FlattenRule(tt.input)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("monitorV2FlattenRule() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
