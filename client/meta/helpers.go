package meta

import (
	"errors"
	"fmt"
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
	CompareFunctionIninterval,
	CompareFunctionNotininterval,
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
