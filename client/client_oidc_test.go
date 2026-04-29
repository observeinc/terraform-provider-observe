package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchOIDCToken(t *testing.T) {
	t.Run("from config OIDCToken", func(t *testing.T) {
		cfg := &Config{
			OAuth2: &OAuth2Config{
				OIDCToken: "token-from-config",
			},
		}
		src := &oidcTokenSource{cfg: cfg, httpClient: http.DefaultClient}
		token, err := src.fetchOIDCToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "token-from-config" {
			t.Errorf("expected 'token-from-config', got %q", token)
		}
	})

	t.Run("from config OIDCTokenFilePath", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "token.txt")
		if err := os.WriteFile(tokenPath, []byte("token-from-file\n"), 0644); err != nil {
			t.Fatal(err)
		}
		cfg := &Config{
			OAuth2: &OAuth2Config{
				OIDCTokenFilePath: tokenPath,
			},
		}
		src := &oidcTokenSource{cfg: cfg, httpClient: http.DefaultClient}
		token, err := src.fetchOIDCToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "token-from-file" {
			t.Errorf("expected 'token-from-file', got %q", token)
		}
	})

	t.Run("from GitHub Actions", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-req-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"value": "token-from-gha"}`)
		}))
		defer srv.Close()

		t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL)
		t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "test-req-token")

		cfg := &Config{
			OAuth2: &OAuth2Config{},
		}
		src := &oidcTokenSource{cfg: cfg, httpClient: http.DefaultClient}
		token, err := src.fetchOIDCToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "token-from-gha" {
			t.Errorf("expected 'token-from-gha', got %q", token)
		}
	})

	t.Run("from Terraform Cloud", func(t *testing.T) {
		// Ensure GHA env vars are unset
		t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
		t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")
		t.Setenv("TFC_WORKLOAD_IDENTITY_TOKEN", "token-from-tfc")

		cfg := &Config{
			OAuth2: &OAuth2Config{},
		}
		src := &oidcTokenSource{cfg: cfg, httpClient: http.DefaultClient}
		token, err := src.fetchOIDCToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "token-from-tfc" {
			t.Errorf("expected 'token-from-tfc', got %q", token)
		}
	})
}

func TestOIDCTokenExchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if r.FormValue("grant_type") != "client_credentials" {
			t.Errorf("expected grant_type=client_credentials, got %q", r.FormValue("grant_type"))
		}
		if r.FormValue("client_assertion_type") != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
			t.Errorf("expected client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer, got %q", r.FormValue("client_assertion_type"))
		}
		if r.FormValue("client_assertion") != "test-oidc-token" {
			t.Errorf("expected client_assertion=test-oidc-token, got %q", r.FormValue("client_assertion"))
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("expected client_id=test-client-id, got %q", r.FormValue("client_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"access_token": "test-access-token", "token_type": "bearer", "expires_in": 3600}`)
	}))
	defer srv.Close()

	cfg := &Config{
		OAuth2: &OAuth2Config{
			ClientID:  "test-client-id",
			TokenURL:  srv.URL,
			OIDCToken: "test-oidc-token",
		},
	}
	src := &oidcTokenSource{cfg: cfg, httpClient: http.DefaultClient}
	token, err := src.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("expected 'test-access-token', got %q", token.AccessToken)
	}
}
