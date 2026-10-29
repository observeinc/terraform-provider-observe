package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// Composite monitors are MonitorV2s with ruleKind Composite. GraphQL refuses to
// create or update them, so they are managed through the internal (alpha)
// /v1/composite-monitors REST API, which is gated per customer by the
// MonitorV2.compositeRestApi setting.

const compositeMonitorsPath = "/v1/composite-monitors"

type IdRef struct {
	Id string `json:"id"`
}

type CompositeMonitorConstituentInput struct {
	Alias               string `json:"alias"`
	Monitor             IdRef  `json:"monitor"`
	NotifyIndependently *bool  `json:"notifyIndependently,omitempty"`
}

type CompositeMonitorConstituent struct {
	Alias               string `json:"alias"`
	Monitor             IdRef  `json:"monitor"`
	NotifyIndependently bool   `json:"notifyIndependently"`
}

type MonitorComparisonInput struct {
	CompareFn    string  `json:"compareFn"`
	CompareValue *string `json:"compareValue,omitempty"`
	ValueType    *string `json:"valueType,omitempty"`
}

type MonitorComparison struct {
	CompareFn    string  `json:"compareFn"`
	CompareValue *string `json:"compareValue"`
	ValueType    *string `json:"valueType"`
}

// MonitorComparisonTermInput names its column with exactly one of Column, Link and Tag.
type MonitorComparisonTermInput struct {
	Column     *string                `json:"column,omitempty"`
	Path       *string                `json:"path,omitempty"`
	Link       *string                `json:"link,omitempty"`
	Tag        *string                `json:"tag,omitempty"`
	Comparison MonitorComparisonInput `json:"comparison"`
}

type MonitorComparisonTerm struct {
	Column     *string           `json:"column"`
	Path       *string           `json:"path"`
	Link       *string           `json:"link"`
	Tag        *string           `json:"tag"`
	Comparison MonitorComparison `json:"comparison"`
}

type MonitorComparisonExpressionInput struct {
	Operator       string                             `json:"operator"`
	CompareTerms   []MonitorComparisonTermInput       `json:"compareTerms,omitempty"`
	SubExpressions []MonitorComparisonExpressionInput `json:"subExpressions,omitempty"`
}

type MonitorComparisonExpression struct {
	Operator       string                        `json:"operator"`
	CompareTerms   []MonitorComparisonTerm       `json:"compareTerms"`
	SubExpressions []MonitorComparisonExpression `json:"subExpressions"`
}

type MonitorActionWebhookHeader struct {
	Header string `json:"header"`
	Value  string `json:"value"`
	Redact *bool  `json:"redact,omitempty"`
}

type MonitorActionEmailInput struct {
	Users     []IdRef         `json:"users,omitempty"`
	Addresses []string        `json:"addresses,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
	Body      *string         `json:"body,omitempty"`
	Fragments json.RawMessage `json:"fragments,omitempty"`
}

type MonitorActionWebhookInput struct {
	Url       string                       `json:"url"`
	Method    *string                      `json:"method,omitempty"`
	Headers   []MonitorActionWebhookHeader `json:"headers,omitempty"`
	Body      *string                      `json:"body,omitempty"`
	Fragments json.RawMessage              `json:"fragments,omitempty"`
}

// MonitorActionInlineInput is an action owned by this monitor alone. Id is set
// only on update, to change an existing inline action in place.
type MonitorActionInlineInput struct {
	Id      *string                    `json:"id,omitempty"`
	Type    string                     `json:"type"`
	Email   *MonitorActionEmailInput   `json:"email,omitempty"`
	Webhook *MonitorActionWebhookInput `json:"webhook,omitempty"`
}

// MonitorActionRuleInput binds exactly one of Action (a shared action) and InlineAction.
type MonitorActionRuleInput struct {
	Action                       *IdRef                            `json:"action,omitempty"`
	InlineAction                 *MonitorActionInlineInput         `json:"inlineAction,omitempty"`
	Levels                       []string                          `json:"levels,omitempty"`
	Conditions                   *MonitorComparisonExpressionInput `json:"conditions,omitempty"`
	SendEndNotifications         *bool                             `json:"sendEndNotifications,omitempty"`
	SendRemindersIntervalSeconds *float64                          `json:"sendRemindersIntervalSeconds,omitempty"`
}

type MonitorActionEmail struct {
	Users     []IdRef         `json:"users"`
	Addresses []string        `json:"addresses"`
	Subject   *string         `json:"subject"`
	Body      *string         `json:"body"`
	Fragments json.RawMessage `json:"fragments"`
}

type MonitorActionWebhook struct {
	Url       *string                      `json:"url"`
	Method    *string                      `json:"method"`
	Headers   []MonitorActionWebhookHeader `json:"headers"`
	Body      *string                      `json:"body"`
	Fragments json.RawMessage              `json:"fragments"`
}

type MonitorActionBrief struct {
	Label   *string               `json:"label"`
	Type    string                `json:"type"`
	Inline  bool                  `json:"inline"`
	Email   *MonitorActionEmail   `json:"email"`
	Webhook *MonitorActionWebhook `json:"webhook"`
}

// MonitorActionRef carries Record only when read with expand=true.
type MonitorActionRef struct {
	Id     string              `json:"id"`
	Record *MonitorActionBrief `json:"record,omitempty"`
}

type MonitorActionRule struct {
	Action                       MonitorActionRef             `json:"action"`
	Levels                       []string                     `json:"levels"`
	Conditions                   *MonitorComparisonExpression `json:"conditions"`
	SendEndNotifications         bool                         `json:"sendEndNotifications"`
	SendRemindersIntervalSeconds *float64                     `json:"sendRemindersIntervalSeconds"`
}

type AlertColumnPath struct {
	Name string  `json:"name"`
	Path *string `json:"path"`
}

type AlertLinkColumn struct {
	Name string `json:"name"`
}

type AlertCorrelationTag struct {
	Tag string `json:"tag"`
}

type AlertColumn struct {
	LinkColumn     *AlertLinkColumn     `json:"linkColumn"`
	ColumnPath     *AlertColumnPath     `json:"columnPath"`
	CorrelationTag *AlertCorrelationTag `json:"correlationTag"`
}

type CompositeMonitorWarning struct {
	Code     string  `json:"code"`
	Message  string  `json:"message"`
	Monitors []IdRef `json:"monitors"`
}

type CompositeMonitorResource struct {
	Id                        string                        `json:"id"`
	Label                     string                        `json:"label"`
	Description               *string                       `json:"description"`
	Disabled                  bool                          `json:"disabled"`
	MonitorVersion            string                        `json:"monitorVersion"`
	Constituents              []CompositeMonitorConstituent `json:"constituents"`
	Condition                 string                        `json:"condition"`
	Groupings                 []AlertColumn                 `json:"groupings"`
	EvaluationIntervalSeconds *int64                        `json:"evaluationIntervalSeconds"`
	Actions                   []MonitorActionRule           `json:"actions"`
	CustomVariables           map[string]string             `json:"customVariables"`
	Warnings                  []CompositeMonitorWarning     `json:"warnings,omitempty"`
	CreatedBy                 IdRef                         `json:"createdBy"`
	CreatedAt                 string                        `json:"createdAt"`
	UpdatedBy                 IdRef                         `json:"updatedBy"`
	UpdatedAt                 string                        `json:"updatedAt"`
}

type CompositeMonitorCreateRequest struct {
	Label           string                             `json:"label"`
	Description     *string                            `json:"description,omitempty"`
	Disabled        *bool                              `json:"disabled,omitempty"`
	Constituents    []CompositeMonitorConstituentInput `json:"constituents"`
	Condition       string                             `json:"condition"`
	Actions         []MonitorActionRuleInput           `json:"actions,omitempty"`
	CustomVariables map[string]string                  `json:"customVariables,omitempty"`
}

// CompositeMonitorUpdateRequest is a merge-patch body: nil fields are omitted
// and left unchanged. Description points at a nil *string to send an explicit
// null (clearing it), Actions points at an empty slice to clear all rules, and
// a nil CustomVariables value removes that key.
type CompositeMonitorUpdateRequest struct {
	Label           *string                            `json:"label,omitempty"`
	Description     **string                           `json:"description,omitempty"`
	Disabled        *bool                              `json:"disabled,omitempty"`
	Constituents    []CompositeMonitorConstituentInput `json:"constituents,omitempty"`
	Condition       *string                            `json:"condition,omitempty"`
	Actions         *[]MonitorActionRuleInput          `json:"actions,omitempty"`
	CustomVariables map[string]*string                 `json:"customVariables,omitempty"`
}

func (r *CompositeMonitorResource) Oid() oid.OID {
	return oid.MonitorV2Oid(r.Id)
}

func compositeMonitorPath(id string) string {
	return compositeMonitorsPath + "/" + url.PathEscape(id)
}

func decodeCompositeMonitor(resp *http.Response) (*CompositeMonitorResource, error) {
	defer resp.Body.Close()
	resource := &CompositeMonitorResource{}
	if err := json.NewDecoder(resp.Body).Decode(resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (client *Client) CreateCompositeMonitor(ctx context.Context, req *CompositeMonitorCreateRequest) (*CompositeMonitorResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.requestWithContext(ctx, http.MethodPost, compositeMonitorsPath, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return decodeCompositeMonitor(resp)
}

// GetCompositeMonitor reads with expand=true so inline actions come back with
// their payload.
func (client *Client) GetCompositeMonitor(ctx context.Context, id string) (*CompositeMonitorResource, error) {
	resp, err := client.requestWithContext(ctx, http.MethodGet, compositeMonitorPath(id)+"?expand=true", "", nil)
	if err != nil {
		return nil, err
	}
	return decodeCompositeMonitor(resp)
}

func (client *Client) UpdateCompositeMonitor(ctx context.Context, id string, req *CompositeMonitorUpdateRequest) (*CompositeMonitorResource, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := client.requestWithContext(ctx, http.MethodPatch, compositeMonitorPath(id), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return decodeCompositeMonitor(resp)
}

func (client *Client) DeleteCompositeMonitor(ctx context.Context, id string) error {
	resp, err := client.requestWithContext(ctx, http.MethodDelete, compositeMonitorPath(id), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
