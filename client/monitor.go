package client

import (
	"fmt"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type (
	AggregateFunction      = meta.AggregateFunction
	ChangeType             = meta.ChangeType
	CompareFunction        = meta.CompareFunction
	FacetFunction          = meta.FacetFunction
	NotificationImportance = meta.NotificationImportance
	NotificationMerge      = meta.NotificationMerge
	TimeFunction           = meta.TimeFunction
	MonitorGroupInfo       = meta.MonitorGroupInfo
)

var (
	AggregateFunctions = []AggregateFunction{
		meta.AggregateFunctionAvg,
		meta.AggregateFunctionSum,
		meta.AggregateFunctionMin,
		meta.AggregateFunctionMax,
	}

	ChangeTypes = []ChangeType{
		meta.ChangeTypeAbsolute,
		meta.ChangeTypeRelative,
	}

	CompareFunctions = []CompareFunction{
		meta.CompareFunctionEqual,
		meta.CompareFunctionNotEqual,
		meta.CompareFunctionGreater,
		meta.CompareFunctionGreaterOrEqual,
		meta.CompareFunctionLess,
		meta.CompareFunctionLessOrEqual,
		meta.CompareFunctionBetweenHalfOpen,
		meta.CompareFunctionNotBetweenHalfOpen,
		meta.CompareFunctionIsNull,
		meta.CompareFunctionIsNotNull,
	}

	FacetFunctions = []FacetFunction{
		meta.FacetFunctionEquals,
		meta.FacetFunctionNotEqual,
		meta.FacetFunctionContains,
		meta.FacetFunctionDoesNotContain,
		meta.FacetFunctionIsNull,
		meta.FacetFunctionIsNotNull,
	}

	NotificationImportances = []NotificationImportance{
		meta.NotificationImportanceInformational,
		meta.NotificationImportanceImportant,
	}

	NotificationMerges = []NotificationMerge{
		meta.NotificationMergeMerged,
		meta.NotificationMergeSeparate,
	}

	TimeFunctions = []TimeFunction{
		meta.TimeFunctionNever,
		meta.TimeFunctionAtLeastOnce,
		meta.TimeFunctionAtAllTimes,
		meta.TimeFunctionAtLeastPercentageTime,
		meta.TimeFunctionLessThanPercentageTime,
	}
)

// Monitor creates notifications from an input query and a trigger rule
type Monitor struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	Config      *MonitorConfig `json:"config"`
}

// MonitorConfig contains configurable elements associated to Monitor
type MonitorConfig struct {
	*Query
	Name                string                  `json:"name"`
	Description         *string                 `json:"description"`
	IconURL             *string                 `json:"iconUrl"`
	Freshness           *time.Duration          `json:"freshness"`
	UseDefaultFreshness *bool                   `json:"useDefaultFreshness"`
	Disabled            bool                    `json:"disabled"`
	Rule                *MonitorRuleConfig      `json:"rule"`
	NotificationSpec    *NotificationSpecConfig `json:"notificationSpec"`
}

type NotificationSpecConfig struct {
	Importance *NotificationImportance `json:"importance"`
	Merge      *NotificationMerge      `json:"merge"`
}

type MonitorRuleConfig struct {
	SourceColumn  *string                     `json:"sourceColumn"`
	GroupByGroups []MonitorGroupInfo          `json:"groupByGroups"`
	ChangeRule    *MonitorRuleChangeConfig    `json:"change"`
	CountRule     *MonitorRuleCountConfig     `json:"count"`
	FacetRule     *MonitorRuleFacetConfig     `json:"facet"`
	ThresholdRule *MonitorRuleThresholdConfig `json:"threshold"`
	PromoteRule   *MonitorRulePromoteConfig   `json:"promote"`
}

func (m *Monitor) OID() *OID {
	return &OID{
		Type: TypeMonitor,
		ID:   m.ID,
	}
}

func (c *MonitorConfig) toGQL() (*meta.MonitorInput, error) {
	queryInput, err := c.Query.toGQL()
	if err != nil {
		return nil, err
	}

	ruleInput, err := c.Rule.toGQL()
	if err != nil {
		return nil, err
	}

	monitorInput := &meta.MonitorInput{
		Name:            &c.Name,
		IconUrl:         c.IconURL,
		Description:     c.Description,
		Disabled:        c.Disabled,
		Query:           queryInput,
		Rule:            ruleInput,
		OverwriteSource: true,
		NotificationSpec: &meta.NotificationSpecificationInput{
			Importance: c.NotificationSpec.Importance,
			Merge:      c.NotificationSpec.Merge,
		},
	}

	if c.Freshness != nil {
		i := fmt.Sprintf("%d", c.Freshness.Nanoseconds())
		monitorInput.FreshnessGoal = &i
		useDefaultFreshness := false
		monitorInput.UseDefaultFreshness = &useDefaultFreshness
	}

	return monitorInput, nil
}

func (c *MonitorRuleConfig) toGQL() (*meta.MonitorRuleInput, error) {
	ruleInput := &meta.MonitorRuleInput{
		SourceColumn:  c.SourceColumn,
		GroupByGroups: c.GroupByGroups,
	}

	var err error

	switch {
	case c.ChangeRule != nil:
		ruleInput.ChangeRule, err = c.ChangeRule.toGQL()
	case c.CountRule != nil:
		ruleInput.CountRule, err = c.CountRule.toGQL()
	case c.FacetRule != nil:
		ruleInput.FacetRule, err = c.FacetRule.toGQL()
	case c.ThresholdRule != nil:
		ruleInput.ThresholdRule, err = c.ThresholdRule.toGQL()
	case c.PromoteRule != nil:
		ruleInput.PromoteRule, err = c.PromoteRule.toGQL()
	default:
		err = fmt.Errorf("no rule found")
	}

	return ruleInput, err
}

func newRuleConfig(gqlRule *meta.MonitorRule) (*MonitorRuleConfig, error) {
	config := &MonitorRuleConfig{
		SourceColumn:  gqlRule.SourceColumn,
		GroupByGroups: gqlRule.GroupByGroups,
	}

	var err error
	switch gqlRule.Type {
	case "MonitorRuleCount":
		err = gqlRule.DecodeType(&config.CountRule)
	case "MonitorRuleChange":
		err = gqlRule.DecodeType(&config.ChangeRule)
	case "MonitorRuleFacet":
		err = gqlRule.DecodeType(&config.FacetRule)
	case "MonitorRuleThreshold":
		err = gqlRule.DecodeType(&config.ThresholdRule)
	case "MonitorRulePromote":
		err = gqlRule.DecodeType(&config.PromoteRule)
	default:
		err = fmt.Errorf("unhandled rule type %s", gqlRule.Type)
	}

	if err != nil {
		return nil, err
	}
	return config, nil
}

func newMonitor(gqlMonitor *meta.Monitor) (m *Monitor, err error) {

	m = &Monitor{
		ID:          gqlMonitor.Id.String(),
		WorkspaceID: gqlMonitor.WorkspaceId.String(),
		Config: &MonitorConfig{
			Name:             gqlMonitor.Name,
			Disabled:         gqlMonitor.Disabled,
			NotificationSpec: &NotificationSpecConfig{},
		},
	}

	if gqlMonitor.Description != "" {
		m.Config.Description = &gqlMonitor.Description
	}

	if gqlMonitor.IconUrl != "" {
		m.Config.IconURL = &gqlMonitor.IconUrl
	}

	m.Config.Query, err = newQuery(gqlMonitor.Query)
	if err != nil {
		return nil, err
	}

	m.Config.Rule, err = newRuleConfig(gqlMonitor.Rule)
	if err != nil {
		return nil, err
	}

	m.Config.NotificationSpec.Merge = &gqlMonitor.NotificationSpec.Merge
	m.Config.NotificationSpec.Importance = &gqlMonitor.NotificationSpec.Importance

	return m, nil
}

type MonitorRuleChangeConfig struct {
	ChangeType        ChangeType         `json:"changeType"`
	AggregateFunction *AggregateFunction `json:"aggregateFunction"`
	CompareFunction   *CompareFunction   `json:"compareFunction"`
	CompareValues     []float64          `json:"compareValues"`
	LookbackTime      *time.Duration     `json:"lookbackTime"`
	BaselineTime      *time.Duration     `json:"baselineTime"`
}

func (c *MonitorRuleChangeConfig) toGQL() (*meta.MonitorRuleChangeInput, error) {
	input := &meta.MonitorRuleChangeInput{
		ChangeType:        &c.ChangeType,
		AggregateFunction: c.AggregateFunction,
		CompareFunction:   c.CompareFunction,
	}

	for _, v := range c.CompareValues {
		input.CompareValues = append(input.CompareValues, meta.NumberScalar(v))
	}

	if c.LookbackTime != nil {
		i := fmt.Sprintf("%d", c.LookbackTime.Nanoseconds())
		input.LookbackTime = &i
	}

	if c.BaselineTime != nil {
		i := fmt.Sprintf("%d", c.BaselineTime.Nanoseconds())
		input.BaselineTime = &i
	}

	return input, nil
}

type MonitorRuleCountConfig struct {
	CompareFunction *CompareFunction `json:"compareFunction"`
	CompareValues   []float64        `json:"compareValues"`
	LookbackTime    *time.Duration   `json:"lookbackTime"`
}

func (c *MonitorRuleCountConfig) toGQL() (*meta.MonitorRuleCountInput, error) {
	input := &meta.MonitorRuleCountInput{
		CompareFunction: c.CompareFunction,
	}

	for _, v := range c.CompareValues {
		input.CompareValues = append(input.CompareValues, meta.NumberScalar(v))
	}

	if c.LookbackTime != nil {
		i := fmt.Sprintf("%d", c.LookbackTime.Nanoseconds())
		input.LookbackTime = &i
	}

	return input, nil
}

type MonitorRuleFacetConfig struct {
	FacetFunction *FacetFunction `json:"facetFunction"`
	FacetValues   []string       `json:"facetValues"`
	TimeFunction  *TimeFunction  `json:"timeFunction"`
	TimeValue     *float64       `json:"timeValue,omitempty"`
	LookbackTime  *time.Duration `json:"lookbackTime"`
}

func (c *MonitorRuleFacetConfig) toGQL() (*meta.MonitorRuleFacetInput, error) {
	input := &meta.MonitorRuleFacetInput{
		FacetFunction: c.FacetFunction,
		FacetValues:   c.FacetValues,
		TimeFunction:  c.TimeFunction,
	}

	if f := c.TimeValue; f != nil {
		s := meta.NumberScalar(*f)
		input.TimeValue = &s
	}

	if c.LookbackTime != nil {
		i := fmt.Sprintf("%d", c.LookbackTime.Nanoseconds())
		input.LookbackTime = &i
	}

	return input, nil
}

type MonitorRuleThresholdConfig struct {
	CompareFunction *CompareFunction `json:"compareFunction"`
	CompareValues   []float64        `json:"compareValues"`
	LookbackTime    *time.Duration   `json:"lookbackTime"`
}

func (c *MonitorRuleThresholdConfig) toGQL() (*meta.MonitorRuleThresholdInput, error) {
	input := &meta.MonitorRuleThresholdInput{
		CompareFunction: c.CompareFunction,
	}

	for _, v := range c.CompareValues {
		input.CompareValues = append(input.CompareValues, meta.NumberScalar(v))
	}

	if c.LookbackTime != nil {
		i := fmt.Sprintf("%d", c.LookbackTime.Nanoseconds())
		input.LookbackTime = &i
	}

	return input, nil
}

type MonitorRulePromoteConfig struct {
	KindField        *string  `json:"kindField"`
	DescriptionField *string  `json:"descriptionField"`
	PrimaryKey       []string `json:"primaryKey"`
}

func (c *MonitorRulePromoteConfig) toGQL() (*meta.MonitorRulePromoteInput, error) {
	input := &meta.MonitorRulePromoteInput{
		PrimaryKey:       c.PrimaryKey,
		KindField:        c.KindField,
		DescriptionField: c.DescriptionField,
	}

	return input, nil
}
