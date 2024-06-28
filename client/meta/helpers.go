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
	if errList, ok := err.(gqlerror.List); ok {
		for _, err := range errList {
			if err.Extensions["code"] == code {
				return true
			}
		}
	}
	return false
}
