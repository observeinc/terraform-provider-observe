package meta

import (
	"errors"
	"fmt"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

var AllBoardType = []BoardType{
	BoardTypeSet,
	BoardTypeSingleton,
}

var AllCompareFunctions = []CompareFunction{
	CompareFunctionEqual,
	CompareFunctionNotequal,
	CompareFunctionGreater,
	CompareFunctionGreaterorequal,
	CompareFunctionLess,
	CompareFunctionLessorequal,
	CompareFunctionIsnull,
	CompareFunctionIsnotnull,
}

var AllMonitorV2ComparisonFunctions = []MonitorV2ComparisonFunction{
	MonitorV2ComparisonFunctionContains,
	MonitorV2ComparisonFunctionEqual,
	MonitorV2ComparisonFunctionGreater,
	MonitorV2ComparisonFunctionGreaterorequal,
	MonitorV2ComparisonFunctionIsnotnull,
	MonitorV2ComparisonFunctionIsnull,
	MonitorV2ComparisonFunctionLess,
	MonitorV2ComparisonFunctionLessorequal,
	MonitorV2ComparisonFunctionNotcontains,
	MonitorV2ComparisonFunctionNotequal,
	MonitorV2ComparisonFunctionNotstartswith,
	MonitorV2ComparisonFunctionStartswith,
}

var AllFacetFunctions = []FacetFunction{
	FacetFunctionEquals,
	FacetFunctionNotequal,
	FacetFunctionContains,
	FacetFunctionDoesnotcontain,
	FacetFunctionIsnull,
	FacetFunctionIsnotnull,
}

var AllChangeTypes = []ChangeType{
	ChangeTypeAbsolute,
	ChangeTypeRelative,
}

var AllAggregateFunctions = []AggregateFunction{
	AggregateFunctionAvg,
	AggregateFunctionSum,
	AggregateFunctionMin,
	AggregateFunctionMax,
}

var AllTimeFunctions = []TimeFunction{
	TimeFunctionNever,
	TimeFunctionAtleastonce,
	TimeFunctionAtalltimes,
	TimeFunctionAtleastpercentagetime,
	TimeFunctionLessthanpercentagetime,
	TimeFunctionNoevents,
	TimeFunctionAllevents,
	TimeFunctionCounttimes,
}

var AllNotificationImportances = []NotificationImportance{
	NotificationImportanceInformational,
	NotificationImportanceImportant,
}

var AllNotificationMerges = []NotificationMerge{
	NotificationMergeMerged,
	NotificationMergeSeparate,
}

var AllThresholdAggFunctions = []ThresholdAggFunction{
	ThresholdAggFunctionAtalltimes,
	ThresholdAggFunctionAtleastonce,
	ThresholdAggFunctionOnaverage,
	ThresholdAggFunctionIntotal,
}

var AllRbacRoles = []RbacRole{
	RbacRoleManager,
	RbacRoleEditor,
	RbacRoleViewer,
	RbacRoleIngester,
	RbacRoleLister,
}

var AllPollerHTTPRequestAuthSchemes = []PollerHTTPRequestAuthScheme{
	PollerHTTPRequestAuthSchemeBasic,
	PollerHTTPRequestAuthSchemeDigest,
}

var AllPollerHTTPTimestampFormats = []PollerHTTPTimestampFormatScheme{
	PollerHTTPTimestampFormatSchemeAnsic,
	PollerHTTPTimestampFormatSchemeUnixdate,
	PollerHTTPTimestampFormatSchemeRubydate,
	PollerHTTPTimestampFormatSchemeRfc822,
	PollerHTTPTimestampFormatSchemeRfc822z,
	PollerHTTPTimestampFormatSchemeRfc850,
	PollerHTTPTimestampFormatSchemeRfc1123,
	PollerHTTPTimestampFormatSchemeRfc1123z,
	PollerHTTPTimestampFormatSchemeRfc3339,
	PollerHTTPTimestampFormatSchemeRfc3339nano,
	PollerHTTPTimestampFormatSchemeKitchen,
	PollerHTTPTimestampFormatSchemeUnix,
	PollerHTTPTimestampFormatSchemeUnixmilli,
	PollerHTTPTimestampFormatSchemeUnixmicro,
	PollerHTTPTimestampFormatSchemeUnixmano,
}

// AllBookmarkKindTypes This list is incomplete and will be filled in
// as we support more types of bookmarks in the terraform provider
var AllBookmarkKindTypes = []BookmarkKind{
	BookmarkKindDataset,
	BookmarkKindDashboard,
	BookmarkKindLogexplorer,
	BookmarkKindMetricexplorer,
}

var AllMonitorV2RuleKinds = []MonitorV2RuleKind{
	MonitorV2RuleKindCount,
	MonitorV2RuleKindPromote,
	MonitorV2RuleKindThreshold,
}

var AllMonitorV2AlarmLevels = []MonitorV2AlarmLevel{
	MonitorV2AlarmLevelCritical,
	MonitorV2AlarmLevelError,
	MonitorV2AlarmLevelInformational,
	MonitorV2AlarmLevelNone,
	MonitorV2AlarmLevelWarning,
	MonitorV2AlarmLevelNodata,
}

var AllMonitorV2ValueAggregations = []MonitorV2ValueAggregation{
	MonitorV2ValueAggregationAllof,
	MonitorV2ValueAggregationAnyof,
	MonitorV2ValueAggregationAvgof,
	MonitorV2ValueAggregationMax,
	MonitorV2ValueAggregationMin,
	MonitorV2ValueAggregationSumof,
}

var AllMonitorV2RollupStatuses = []MonitorV2RollupStatus{
	MonitorV2RollupStatusDegraded,
	MonitorV2RollupStatusFailed,
	MonitorV2RollupStatusInactive,
	MonitorV2RollupStatusRunning,
	MonitorV2RollupStatusTriggering,
}

var AllMonitorV2ActionTypes = []MonitorV2ActionType{
	MonitorV2ActionTypeEmail,
	MonitorV2ActionTypePagerduty,
	MonitorV2ActionTypeSlack,
	MonitorV2ActionTypeWebhook,
}

var AllMonitorV2HttpTypes = []MonitorV2HttpType{
	MonitorV2HttpTypePost,
	MonitorV2HttpTypePut,
}

var AllMonitorV2BooleanOperators = []MonitorV2BooleanOperator{
	MonitorV2BooleanOperatorAnd,
	MonitorV2BooleanOperatorOr,
}

var AllAccelerationDisabledSource = []AccelerationDisabledSource{
	AccelerationDisabledSourceEmpty,
	AccelerationDisabledSourceMonitor,
	AccelerationDisabledSourceView,
}

var AllRematerializationModes = []RematerializationMode{
	RematerializationModeRematerialize,
	RematerializationModeSkiprematerialization,
}

const (
	ErrNotFound = "NOT_FOUND"
)

type resultStatusResponse interface {
	GetResultStatus() ResultStatus
}

type optionalResultStatusResponse interface {
	GetResultStatus() *ResultStatus
}

func resultStatusError(r resultStatusResponse, err error) error {
	if err != nil {
		return err
	}
	rs := r.GetResultStatus()
	return extractResultStatusError(rs)
}

func optionalResultStatusError(r optionalResultStatusResponse, err error) error {
	if err != nil {
		return err
	}
	rs := r.GetResultStatus()
	if rs == nil {
		return nil
	}
	return extractResultStatusError(*rs)
}

func extractResultStatusError(rs ResultStatus) error {
	if rs.GetSuccess() {
		return nil
	}
	msg := rs.GetErrorMessage()
	if msg != "" {
		return fmt.Errorf("request failed: %q", msg)
	}
	return errors.New("request failed")
}

func HasErrorCode(err error, code string) bool {
	if err == nil {
		return false
	}
	var errList gqlerror.List
	if errors.As(err, &errList) {
		for _, err := range errList {
			if err.Extensions["code"] == code {
				return true
			}
		}
	}
	return false
}
