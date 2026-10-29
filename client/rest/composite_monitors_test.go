package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const compositeMonitorJSON = `{
	"id": "41000100",
	"label": "both down",
	"description": null,
	"disabled": false,
	"monitorVersion": "1761000000000000000",
	"constituents": [
		{"alias": "A", "monitor": {"id": "41000001"}, "notifyIndependently": true},
		{"alias": "B", "monitor": {"id": "41000002"}, "notifyIndependently": false}
	],
	"condition": "A && B",
	"groupings": [{"linkColumn": null, "columnPath": {"name": "host", "path": null}, "correlationTag": null}],
	"evaluationIntervalSeconds": 60,
	"actions": [{
		"action": {"id": "41000200", "record": {"label": null, "type": "Webhook", "inline": true, "email": null,
			"webhook": {"url": "https://example.com/hook", "method": "Post", "headers": [], "body": "{}", "fragments": null}}},
		"levels": ["Critical"],
		"conditions": null,
		"sendEndNotifications": true,
		"sendRemindersIntervalSeconds": 600
	}],
	"customVariables": {"title": "down"},
	"warnings": [{"code": "FiresWhenAllQuiet", "message": "fires when quiet", "monitors": []}],
	"createdBy": {"id": "1"}, "createdAt": "2026-10-29T00:00:00Z",
	"updatedBy": {"id": "1"}, "updatedAt": "2026-10-29T00:00:00Z"
}`

func TestCompositeMonitorClient(t *testing.T) {
	var gotPatch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/composite-monitors":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create: %v", err)
			}
			if body["label"] != "both down" || body["condition"] != "A && B" {
				t.Errorf("unexpected create body %#v", body)
			}
			if _, ok := body["description"]; ok {
				t.Errorf("create sent description although none was set: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(compositeMonitorJSON))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/composite-monitors/41000100":
			if r.URL.Query().Get("expand") != "true" {
				t.Errorf("get without expand=true: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(compositeMonitorJSON))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/composite-monitors/404":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "not found"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/composite-monitors/41000100":
			b, _ := io.ReadAll(r.Body)
			gotPatch = string(b)
			_, _ = w.Write([]byte(compositeMonitorJSON))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/composite-monitors/41000100":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	client := New(server.URL, server.Client())

	created, err := client.CreateCompositeMonitor(ctx, &CompositeMonitorCreateRequest{
		Label: "both down",
		Constituents: []CompositeMonitorConstituentInput{
			{Alias: "A", Monitor: IdRef{Id: "41000001"}},
			{Alias: "B", Monitor: IdRef{Id: "41000002"}},
		},
		Condition: "A && B",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Id != "41000100" || len(created.Warnings) != 1 || created.Warnings[0].Code != "FiresWhenAllQuiet" {
		t.Errorf("unexpected create result %#v", created)
	}
	if got := created.Oid().String(); got != "o:::monitorv2:41000100" {
		t.Errorf("oid = %q", got)
	}

	got, err := client.GetCompositeMonitor(ctx, "41000100")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Constituents) != 2 || got.Constituents[1].NotifyIndependently {
		t.Errorf("unexpected constituents %#v", got.Constituents)
	}
	record := got.Actions[0].Action.Record
	if record == nil || !record.Inline || record.Webhook == nil || *record.Webhook.Url != "https://example.com/hook" {
		t.Errorf("unexpected action record %#v", record)
	}
	if *got.EvaluationIntervalSeconds != 60 || *got.Actions[0].SendRemindersIntervalSeconds != 600 {
		t.Errorf("unexpected intervals in %#v", got)
	}

	if _, err := client.GetCompositeMonitor(ctx, "404"); !HasStatusCode(err, http.StatusNotFound) {
		t.Errorf("get missing: err = %v, want 404", err)
	}

	var clearedDescription *string
	clearedActions := []MonitorActionRuleInput{}
	newTitle := "still down"
	if _, err := client.UpdateCompositeMonitor(ctx, "41000100", &CompositeMonitorUpdateRequest{
		Description:     &clearedDescription,
		Actions:         &clearedActions,
		CustomVariables: map[string]*string{"old": nil, "title": &newTitle},
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if want := `{"description":null,"actions":[],"customVariables":{"old":null,"title":"still down"}}`; gotPatch != want {
		t.Errorf("patch body = %s, want %s", gotPatch, want)
	}

	if err := client.DeleteCompositeMonitor(ctx, "41000100"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
