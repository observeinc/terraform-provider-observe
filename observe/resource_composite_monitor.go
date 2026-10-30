package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

// Composite monitors are written through the dedicated /v1/composite-monitors
// REST API; there is no GraphQL API to create or update them. The `actions` blocks
// share their schema and input parsing with observe_monitor_v2, and are
// converted from the GraphQL input types into the REST ones.

func resourceCompositeMonitor() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("composite_monitor", "description"),
		CreateContext: resourceCompositeMonitorCreate,
		ReadContext:   resourceCompositeMonitorRead,
		UpdateContext: resourceCompositeMonitorUpdate,
		DeleteContext: resourceCompositeMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("composite_monitor", "schema", "name"),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions.Get("composite_monitor", "schema", "description"),
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions.Get("composite_monitor", "schema", "disabled"),
			},
			"condition": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions.Get("composite_monitor", "schema", "condition"),
			},
			"constituents": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    2,
				MaxItems:    10,
				Description: descriptions.Get("composite_monitor", "schema", "constituents", "description"),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alias": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions.Get("composite_monitor", "schema", "constituents", "alias"),
						},
						"monitor": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateOID(oid.TypeMonitorV2),
							Description:      descriptions.Get("composite_monitor", "schema", "constituents", "monitor"),
						},
						"notify_independently": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: descriptions.Get("composite_monitor", "schema", "constituents", "notify_independently"),
						},
					},
				},
			},
			"actions": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        monitorV2ActionRuleResource(),
				Description: descriptions.Get("composite_monitor", "schema", "actions", "description"),
			},
			"custom_variables": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: descriptions.Get("composite_monitor", "schema", "custom_variables"),
			},
			"groupings": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        monitorV2ColumnResource(),
				Description: descriptions.Get("composite_monitor", "schema", "groupings"),
			},
			"evaluation_interval": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("composite_monitor", "schema", "evaluation_interval"),
			},
			"oid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: descriptions.Get("common", "schema", "oid"),
			},
		},
	}
}

func resourceCompositeMonitorCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	req, diags := newCompositeMonitorCreateRequest(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.CreateCompositeMonitor(ctx, req)
	if err != nil {
		return compositeMonitorErrorDiag("failed to create composite monitor", err)
	}
	data.SetId(result.Id)
	diags = append(diags, compositeMonitorWarningDiags(result.Warnings)...)
	return append(diags, resourceCompositeMonitorRead(ctx, data, meta)...)
}

func resourceCompositeMonitorRead(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	result, err := client.GetCompositeMonitor(ctx, data.Id())
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return compositeMonitorErrorDiag("failed to retrieve composite monitor", err)
	}
	return compositeMonitorToResourceData(result, data)
}

func resourceCompositeMonitorUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	req, diags := newCompositeMonitorUpdateRequest(data)
	if diags.HasError() {
		return diags
	}

	result, err := client.UpdateCompositeMonitor(ctx, data.Id(), req)
	if err != nil {
		return compositeMonitorErrorDiag("failed to update composite monitor", err)
	}
	diags = append(diags, compositeMonitorWarningDiags(result.Warnings)...)
	return append(diags, resourceCompositeMonitorRead(ctx, data, meta)...)
}

func resourceCompositeMonitorDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)
	if err := client.DeleteCompositeMonitor(ctx, data.Id()); err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			return nil
		}
		return compositeMonitorErrorDiag("failed to delete composite monitor", err)
	}
	return nil
}

func compositeMonitorErrorDiag(summary string, err error) diag.Diagnostics {
	detail := err.Error()
	if rest.HasStatusCode(err, http.StatusForbidden) {
		detail += "\n\nComposite monitors require the composite monitors API to be enabled for your " +
			"Observe customer (MonitorV2.compositeRestApi); contact Observe support if it is not."
	}
	return diag.Diagnostics{{Severity: diag.Error, Summary: summary, Detail: detail}}
}

func compositeMonitorWarningDiags(warnings []rest.CompositeMonitorWarning) (diags diag.Diagnostics) {
	for _, w := range warnings {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("composite monitor: %s", w.Code),
			Detail:   w.Message,
		})
	}
	return diags
}

func newCompositeMonitorCreateRequest(data *schema.ResourceData) (*rest.CompositeMonitorCreateRequest, diag.Diagnostics) {
	actions, diags := newCompositeMonitorActionRuleInputs(data)
	if diags.HasError() {
		return nil, diags
	}
	req := &rest.CompositeMonitorCreateRequest{
		Label:        data.Get("name").(string),
		Disabled:     boolPtr(data.Get("disabled").(bool)),
		Constituents: newCompositeMonitorConstituentInputs(data),
		Condition:    data.Get("condition").(string),
		Actions:      actions,
	}
	if v, ok := data.GetOk("description"); ok {
		req.Description = stringPtr(v.(string))
	}
	if v, ok := data.GetOk("custom_variables"); ok {
		req.CustomVariables = make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			req.CustomVariables[k] = val.(string)
		}
	}
	return req, diags
}

// newCompositeMonitorUpdateRequest builds a merge-patch body with only the changed fields.
func newCompositeMonitorUpdateRequest(data *schema.ResourceData) (*rest.CompositeMonitorUpdateRequest, diag.Diagnostics) {
	req := &rest.CompositeMonitorUpdateRequest{}
	if data.HasChange("name") {
		req.Label = stringPtr(data.Get("name").(string))
	}
	if data.HasChange("description") {
		var description *string
		if v := data.Get("description").(string); v != "" {
			description = &v
		}
		req.Description = &description
	}
	if data.HasChange("disabled") {
		req.Disabled = boolPtr(data.Get("disabled").(bool))
	}
	if data.HasChange("condition") {
		req.Condition = stringPtr(data.Get("condition").(string))
	}
	if data.HasChange("constituents") {
		req.Constituents = newCompositeMonitorConstituentInputs(data)
	}
	if data.HasChange("actions") {
		actions, diags := newCompositeMonitorActionRuleInputs(data)
		if diags.HasError() {
			return nil, diags
		}
		if actions == nil {
			actions = []rest.MonitorActionRuleInput{}
		}
		req.Actions = &actions
	}
	if data.HasChange("custom_variables") {
		oldVars, newVars := data.GetChange("custom_variables")
		req.CustomVariables = make(map[string]*string)
		for k := range oldVars.(map[string]interface{}) {
			req.CustomVariables[k] = nil
		}
		for k, v := range newVars.(map[string]interface{}) {
			req.CustomVariables[k] = stringPtr(v.(string))
		}
	}
	return req, nil
}

func newCompositeMonitorConstituentInputs(data *schema.ResourceData) []rest.CompositeMonitorConstituentInput {
	var constituents []rest.CompositeMonitorConstituentInput
	for i := range data.Get("constituents").([]interface{}) {
		path := fmt.Sprintf("constituents.%d.", i)
		monitorOID, _ := oid.NewOID(data.Get(path + "monitor").(string))
		constituents = append(constituents, rest.CompositeMonitorConstituentInput{
			Alias:               data.Get(path + "alias").(string),
			Monitor:             rest.IdRef{Id: monitorOID.Id},
			NotifyIndependently: boolPtr(data.Get(path + "notify_independently").(bool)),
		})
	}
	return constituents
}

// newCompositeMonitorActionRuleInputs parses `actions` with the observe_monitor_v2
// parser and converts each rule to its REST form. Like observe_monitor_v2, inline
// actions are always sent without an id, so every save of the actions creates
// them anew and the server deletes the inline actions no rule refers to anymore.
func newCompositeMonitorActionRuleInputs(data *schema.ResourceData) ([]rest.MonitorActionRuleInput, diag.Diagnostics) {
	gqlRules, diags := newMonitorV2ActionAndRelationInputs(data)
	if diags.HasError() {
		return nil, diags
	}

	var rules []rest.MonitorActionRuleInput
	for i, gqlRule := range gqlRules {
		rule, err := compositeMonitorActionRuleFromGQL(gqlRule)
		if err != nil {
			return nil, diag.Errorf("%s. Path = actions.%d", err, i)
		}
		rules = append(rules, *rule)
	}
	return rules, nil
}

func compositeMonitorActionRuleFromGQL(in gql.MonitorV2ActionAndRelationInput) (*rest.MonitorActionRuleInput, error) {
	out := &rest.MonitorActionRuleInput{
		SendEndNotifications: in.SendEndNotifications,
	}
	if in.ActionID != nil {
		out.Action = &rest.IdRef{Id: *in.ActionID}
	}
	if in.Action != nil {
		inline, err := compositeMonitorInlineActionFromGQL(in.Action)
		if err != nil {
			return nil, err
		}
		out.InlineAction = inline
	}
	for _, level := range in.Levels {
		out.Levels = append(out.Levels, string(level))
	}
	if in.SendRemindersInterval != nil {
		seconds := time.Duration(*in.SendRemindersInterval).Seconds()
		out.SendRemindersIntervalSeconds = &seconds
	}
	if in.Conditions != nil {
		out.Conditions = compositeMonitorComparisonExpressionFromGQL(in.Conditions)
	}
	return out, nil
}

func compositeMonitorInlineActionFromGQL(in *gql.MonitorV2ActionInput) (*rest.MonitorActionInlineInput, error) {
	if in.Description != nil {
		return nil, fmt.Errorf("inline actions of a composite monitor do not support 'description'")
	}
	out := &rest.MonitorActionInlineInput{Type: string(in.Type)}
	if e := in.Email; e != nil {
		email := &rest.MonitorActionEmailInput{
			Addresses: e.Addresses,
			Body:      e.Body,
		}
		if e.Subject != "" {
			email.Subject = stringPtr(e.Subject)
		}
		for _, uid := range e.Users {
			email.Users = append(email.Users, rest.IdRef{Id: uid.String()})
		}
		if e.Fragments != nil {
			email.Fragments = json.RawMessage(e.Fragments.String())
		}
		out.Email = email
	}
	if w := in.Webhook; w != nil {
		webhook := &rest.MonitorActionWebhookInput{
			Url:    w.Url,
			Method: stringPtr(string(w.Method)),
			Body:   stringPtr(w.Body),
		}
		for _, h := range w.Headers {
			webhook.Headers = append(webhook.Headers, rest.MonitorActionWebhookHeader{Header: h.Header, Value: h.Value, Redact: h.Redact})
		}
		if w.Fragments != nil {
			webhook.Fragments = json.RawMessage(w.Fragments.String())
		}
		out.Webhook = webhook
	}
	return out, nil
}

func compositeMonitorComparisonExpressionFromGQL(in *gql.MonitorV2ComparisonExpressionInput) *rest.MonitorComparisonExpressionInput {
	out := &rest.MonitorComparisonExpressionInput{Operator: string(in.Operator)}
	for _, term := range in.CompareTerms {
		restTerm := rest.MonitorComparisonTermInput{
			Comparison: compositeMonitorComparisonFromGQL(term.Comparison),
		}
		switch col := term.Column; {
		case col.ColumnPath != nil:
			restTerm.Column = stringPtr(col.ColumnPath.Name)
			restTerm.Path = col.ColumnPath.Path
		case col.LinkColumn != nil:
			restTerm.Link = stringPtr(col.LinkColumn.Name)
		case col.CorrelationTag != nil:
			restTerm.Tag = stringPtr(col.CorrelationTag.Tag)
		}
		out.CompareTerms = append(out.CompareTerms, restTerm)
	}
	for i := range in.SubExpressions {
		out.SubExpressions = append(out.SubExpressions, *compositeMonitorComparisonExpressionFromGQL(&in.SubExpressions[i]))
	}
	return out
}

// compositeMonitorComparisonFromGQL renders the comparison's value as the REST
// API's string plus value type. Durations are counts of nanoseconds.
func compositeMonitorComparisonFromGQL(in gql.MonitorV2ComparisonInput) rest.MonitorComparisonInput {
	out := rest.MonitorComparisonInput{CompareFn: string(in.CompareFn)}
	set := func(value, valueType string) {
		out.CompareValue = &value
		out.ValueType = &valueType
	}
	switch v := in.CompareValue; {
	case v.Bool != nil:
		set(strconv.FormatBool(*v.Bool), "Bool")
	case v.Int64 != nil:
		set(strconv.FormatInt(int64(*v.Int64), 10), "Int64")
	case v.Float64 != nil:
		set(strconv.FormatFloat(*v.Float64, 'g', -1, 64), "Float64")
	case v.String != nil:
		set(*v.String, "String")
	case v.Timestamp != nil:
		set(time.Time(*v.Timestamp).UTC().Format(time.RFC3339), "Timestamp")
	case v.Duration != nil:
		set(strconv.FormatInt(int64(*v.Duration), 10), "Duration")
	}
	return out
}

func compositeMonitorToResourceData(m *rest.CompositeMonitorResource, data *schema.ResourceData) (diags diag.Diagnostics) {
	setResourceData := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	data.SetId(m.Id)
	setResourceData("oid", m.Oid().String())
	setResourceData("name", m.Label)
	description := ""
	if m.Description != nil {
		description = *m.Description
	}
	setResourceData("description", description)
	setResourceData("disabled", m.Disabled)
	setResourceData("condition", m.Condition)

	constituents := make([]interface{}, 0, len(m.Constituents))
	for _, c := range m.Constituents {
		constituents = append(constituents, map[string]interface{}{
			"alias":                c.Alias,
			"monitor":              oid.MonitorV2Oid(c.Monitor.Id).String(),
			"notify_independently": c.NotifyIndependently,
		})
	}
	setResourceData("constituents", constituents)

	actions := make([]interface{}, 0, len(m.Actions))
	for _, a := range m.Actions {
		rule, err := compositeMonitorFlattenActionRule(a)
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			continue
		}
		actions = append(actions, rule)
	}
	setResourceData("actions", actions)

	setResourceData("custom_variables", m.CustomVariables)

	groupings := make([]interface{}, 0, len(m.Groupings))
	for _, g := range m.Groupings {
		groupings = append(groupings, compositeMonitorFlattenAlertColumn(g)...)
	}
	setResourceData("groupings", groupings)

	evaluationInterval := ""
	if m.EvaluationIntervalSeconds != nil {
		evaluationInterval = (time.Duration(*m.EvaluationIntervalSeconds) * time.Second).String()
	}
	setResourceData("evaluation_interval", evaluationInterval)

	return diags
}

func compositeMonitorFlattenAlertColumn(c rest.AlertColumn) []interface{} {
	column := map[string]interface{}{}
	if c.LinkColumn != nil {
		column["link_column"] = []interface{}{map[string]interface{}{"name": c.LinkColumn.Name}}
	}
	if c.ColumnPath != nil {
		column["column_path"] = []interface{}{map[string]interface{}{"name": c.ColumnPath.Name, "path": derefString(c.ColumnPath.Path)}}
	}
	if c.CorrelationTag != nil {
		column["correlation_tag"] = []interface{}{map[string]interface{}{"tag": c.CorrelationTag.Tag}}
	}
	return []interface{}{column}
}

// compositeMonitorFlattenActionRule mirrors monitorV2FlattenActionRule for the
// REST shape. Inline actions are read from the expanded action record.
func compositeMonitorFlattenActionRule(a rest.MonitorActionRule) (map[string]interface{}, error) {
	rule := map[string]interface{}{}

	if record := a.Action.Record; record != nil && record.Inline {
		act := map[string]interface{}{"type": toSnake(record.Type)}
		if e := record.Email; e != nil {
			users := make([]string, 0, len(e.Users))
			for _, u := range e.Users {
				users = append(users, oid.OID{Id: u.Id, Type: oid.TypeUser}.String())
			}
			email := map[string]interface{}{
				"subject":   derefString(e.Subject),
				"body":      derefString(e.Body),
				"addresses": e.Addresses,
				"users":     users,
			}
			if f := compositeMonitorFlattenFragments(e.Fragments); f != "" {
				email["fragments"] = f
			}
			act["email"] = []interface{}{email}
		}
		if w := record.Webhook; w != nil {
			headers := make([]interface{}, 0, len(w.Headers))
			for _, h := range w.Headers {
				headers = append(headers, map[string]interface{}{"header": h.Header, "value": h.Value})
			}
			webhook := map[string]interface{}{
				"url":     derefString(w.Url),
				"method":  toSnake(derefString(w.Method)),
				"body":    derefString(w.Body),
				"headers": headers,
			}
			if f := compositeMonitorFlattenFragments(w.Fragments); f != "" {
				webhook["fragments"] = f
			}
			act["webhook"] = []interface{}{webhook}
		}
		rule["action"] = []interface{}{act}
	} else {
		rule["oid"] = oid.MonitorV2ActionOid(a.Action.Id).String()
	}

	levels := make([]interface{}, 0, len(a.Levels))
	for _, level := range a.Levels {
		levels = append(levels, toSnake(level))
	}
	rule["levels"] = levels
	rule["send_end_notifications"] = a.SendEndNotifications
	if a.SendRemindersIntervalSeconds != nil {
		rule["send_reminders_interval"] = time.Duration(*a.SendRemindersIntervalSeconds * float64(time.Second)).String()
	}
	if a.Conditions != nil && len(a.Conditions.CompareTerms) != 0 {
		conditions, err := compositeMonitorFlattenComparisonExpression(a.Conditions)
		if err != nil {
			return nil, err
		}
		rule["conditions"] = []interface{}{conditions}
	}
	return rule, nil
}

func compositeMonitorFlattenFragments(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	return string(raw)
}

// compositeMonitorFlattenComparisonExpression supports a single level of terms,
// like observe_monitor_v2.
func compositeMonitorFlattenComparisonExpression(expr *rest.MonitorComparisonExpression) (map[string]interface{}, error) {
	terms := make([]interface{}, 0, len(expr.CompareTerms))
	for _, term := range expr.CompareTerms {
		comparison, err := compositeMonitorFlattenComparison(term.Comparison)
		if err != nil {
			return nil, err
		}
		column := rest.AlertColumn{}
		switch {
		case term.Column != nil:
			column.ColumnPath = &rest.AlertColumnPath{Name: *term.Column, Path: term.Path}
		case term.Link != nil:
			column.LinkColumn = &rest.AlertLinkColumn{Name: *term.Link}
		case term.Tag != nil:
			column.CorrelationTag = &rest.AlertCorrelationTag{Tag: *term.Tag}
		}
		terms = append(terms, map[string]interface{}{
			"comparison": []interface{}{comparison},
			"column":     compositeMonitorFlattenAlertColumn(column),
		})
	}
	return map[string]interface{}{
		"compare_terms": terms,
		"operator":      toSnake(expr.Operator),
	}, nil
}

func compositeMonitorFlattenComparison(c rest.MonitorComparison) (map[string]interface{}, error) {
	comparison := map[string]interface{}{
		"compare_fn":      toSnake(c.CompareFn),
		"value_bool":      []interface{}{},
		"value_int64":     []interface{}{},
		"value_float64":   []interface{}{},
		"value_string":    []interface{}{},
		"value_timestamp": []interface{}{},
		"value_duration":  []interface{}{},
	}
	if c.CompareValue == nil || c.ValueType == nil {
		return comparison, nil
	}
	value := *c.CompareValue
	var err error
	switch *c.ValueType {
	case "Bool":
		var b bool
		b, err = strconv.ParseBool(value)
		comparison["value_bool"] = []interface{}{b}
	case "Int64":
		var i int64
		i, err = strconv.ParseInt(value, 10, 64)
		comparison["value_int64"] = []interface{}{int(i)}
	case "Float64":
		var f float64
		f, err = strconv.ParseFloat(value, 64)
		comparison["value_float64"] = []interface{}{f}
	case "String":
		comparison["value_string"] = []interface{}{value}
	case "Timestamp":
		comparison["value_timestamp"] = []interface{}{value}
	case "Duration":
		var ns int64
		ns, err = strconv.ParseInt(value, 10, 64)
		comparison["value_duration"] = []interface{}{time.Duration(ns).String()}
	default:
		err = fmt.Errorf("unsupported comparison value type %q", *c.ValueType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read comparison value %q: %w", value, err)
	}
	return comparison, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
