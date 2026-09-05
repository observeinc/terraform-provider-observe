package observe

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestEscapeHCLTemplateMarkers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "interpolation marker",
			in:   `filter contains(message, "${")`,
			want: `filter contains(message, "$${")`,
		},
		{
			name: "directive marker",
			in:   `pct %{if true} y`,
			want: `pct %%{if true} y`,
		},
		{
			name: "already escaped interpolation",
			in:   `a $${b} c`,
			want: `a $$${b} c`,
		},
		{
			name: "triple-dollar already escaped",
			in:   `a $$${b} c`,
			want: `a $$$${b} c`,
		},
		{
			name: "lone dollar signs untouched",
			in:   `lone $$ dollars $5`,
			want: `lone $$ dollars $5`,
		},
		{
			name: "multi-line value",
			in:   "line 1 ${x}\nline 2 %{if cond}y%{endif}\nline 3",
			want: "line 1 $${x}\nline 2 %%{if cond}y%%{endif}\nline 3",
		},
		{
			name: "JSON blob with embedded markers",
			in:   `{"query":"filter contains(msg, \"${app.name}\")"}`,
			want: `{"query":"filter contains(msg, \"$${app.name}\")"}`,
		},
		{
			name: "no markers unchanged",
			in:   "just plain text with $ and % signs",
			want: "just plain text with $ and % signs",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "both markers in one string",
			in:   `${interp} and %{directive}`,
			want: `$${interp} and %%{directive}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeHCLTemplateMarkers(tt.in)
			if got != tt.want {
				t.Errorf("escapeHCLTemplateMarkers(%q)\n  got  = %q\n  want = %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEscapeExportedStrings_Dashboard(t *testing.T) {
	// Use the real dashboard data source schema to test the walker against
	// top-level TypeString attributes.
	s := dataSourceDashboard().Schema

	data := schema.TestResourceDataRaw(t, s, map[string]interface{}{
		"id":               "12345",
		"name":             "My ${env} Dashboard",
		"description":      "Tracks %{if prod}important%{endif} things",
		"stages":           `{"pipeline":"filter ${app}"}`,
		"layout":           `{}`,
		"parameters":       ``,
		"parameter_values": `{"val":"${x}"}`,
		"icon_url":         "https://example.com/icon.png",
	})

	if err := escapeExportedStrings(data, s); err != nil {
		t.Fatalf("escapeExportedStrings: %v", err)
	}

	checks := map[string]string{
		"name":             "My $${env} Dashboard",
		"description":      "Tracks %%{if prod}important%%{endif} things",
		"stages":           `{"pipeline":"filter $${app}"}`,
		"parameter_values": `{"val":"$${x}"}`,
		"icon_url":         "https://example.com/icon.png", // unchanged
		"layout":           `{}`,                           // unchanged
	}
	for key, want := range checks {
		got := data.Get(key).(string)
		if got != want {
			t.Errorf("data.Get(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestEscapeExportedStrings_NestedBlock(t *testing.T) {
	// Verify the walker recurses into nested blocks (using the monitor_v2_action
	// schema which has nested email/webhook blocks with TypeString fields).
	s := dataSourceMonitorV2Action().Schema

	data := schema.TestResourceDataRaw(t, s, map[string]interface{}{
		"id":          "67890",
		"workspace":   "ws-1",
		"name":        "action with ${marker}",
		"description": "plain description",
		"email": []interface{}{
			map[string]interface{}{
				"subject":   "Alert: ${service} is down",
				"body":      "Check %{if critical}now%{endif}",
				"fragments": `{"tmpl":"${x}"}`,
				"users":     []interface{}{"user-1"},
				"addresses": []interface{}{"addr-${team}@example.com"},
			},
		},
	})

	if err := escapeExportedStrings(data, s); err != nil {
		t.Fatalf("escapeExportedStrings: %v", err)
	}

	// Top-level string
	if got := data.Get("name").(string); got != "action with $${marker}" {
		t.Errorf("name = %q, want %q", got, "action with $${marker}")
	}
	// Unchanged string
	if got := data.Get("description").(string); got != "plain description" {
		t.Errorf("description = %q, want %q", got, "plain description")
	}
	// Nested block strings
	email := data.Get("email").([]interface{})
	if len(email) != 1 {
		t.Fatalf("expected 1 email block, got %d", len(email))
	}
	em := email[0].(map[string]interface{})
	if got := em["subject"].(string); got != "Alert: $${service} is down" {
		t.Errorf("email.subject = %q, want %q", got, "Alert: $${service} is down")
	}
	if got := em["body"].(string); got != "Check %%{if critical}now%%{endif}" {
		t.Errorf("email.body = %q, want %q", got, "Check %%{if critical}now%%{endif}")
	}
	if got := em["fragments"].(string); got != `{"tmpl":"$${x}"}` {
		t.Errorf("email.fragments = %q, want %q", got, `{"tmpl":"$${x}"}`)
	}
	// String list in nested block
	addrs := em["addresses"].([]interface{})
	if len(addrs) != 1 {
		t.Fatalf("expected 1 address, got %d", len(addrs))
	}
	if got := addrs[0].(string); got != "addr-$${team}@example.com" {
		t.Errorf("email.addresses[0] = %q, want %q", got, "addr-$${team}@example.com")
	}
}

func TestEscapeExportedStrings_NoChange(t *testing.T) {
	// When no markers are present, the walker should not call data.Set at all.
	// We verify by checking the value stays identical (no-op round-trip).
	s := dataSourceDashboard().Schema

	data := schema.TestResourceDataRaw(t, s, map[string]interface{}{
		"id":          "99999",
		"name":        "Simple Dashboard",
		"description": "No special characters here",
		"stages":      `{"pipeline":"filter true"}`,
		"layout":      `{"cards":[]}`,
	})

	if err := escapeExportedStrings(data, s); err != nil {
		t.Fatalf("escapeExportedStrings: %v", err)
	}

	// Values unchanged
	if got := data.Get("name").(string); got != "Simple Dashboard" {
		t.Errorf("name = %q, want %q", got, "Simple Dashboard")
	}
	if got := data.Get("stages").(string); got != `{"pipeline":"filter true"}` {
		t.Errorf("stages = %q, want %q", got, `{"pipeline":"filter true"}`)
	}
}
