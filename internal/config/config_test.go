package config_test

import (
	"testing"

	"github.com/tenqz/google-search-console-mcp/internal/config"
)

// TestLoadRequiresAuthToken documents that a public MCP endpoint must not start without a shared secret.
func TestLoadRequiresAuthToken(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("MCP_ALLOW_INSECURE", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/sa.json")

	_, err := config.Load()

	if err == nil {
		t.Fatal("expected an error when MCP_AUTH_TOKEN is empty")
	}
}

// TestLoadAllowsInsecureWithoutToken documents the local-debug escape hatch.
func TestLoadAllowsInsecureWithoutToken(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("MCP_ALLOW_INSECURE", "true")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/sa.json")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.AllowInsecure {
		t.Fatal("expected AllowInsecure to be true")
	}
}

// TestLoadRequiresGoogleCredentials documents that Google auth is mandatory.
func TestLoadRequiresGoogleCredentials(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "secret")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("GOOGLE_CREDENTIALS_JSON", "")
	t.Setenv("GOOGLE_CREDENTIALS_FILE", "")

	_, err := config.Load()

	if err == nil {
		t.Fatal("expected an error when Google credentials are missing")
	}
}

// TestLoadAcceptsInlineJSON documents GOOGLE_CREDENTIALS_JSON as an alternative to a file.
func TestLoadAcceptsInlineJSON(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "secret")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("GOOGLE_CREDENTIALS_FILE", "")
	t.Setenv("GOOGLE_CREDENTIALS_JSON", `{"type":"service_account"}`)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GoogleCredentialsJSON == "" {
		t.Fatal("expected GoogleCredentialsJSON to be set")
	}
}

// TestLoadRejectsMCPPathWithoutSlash documents that MCP_PATH must be an absolute HTTP path.
func TestLoadRejectsMCPPathWithoutSlash(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "secret")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/sa.json")
	t.Setenv("MCP_PATH", "mcp")

	_, err := config.Load()

	if err == nil {
		t.Fatal("expected an error for MCP_PATH without a leading slash")
	}
}

// TestLoadRejectsUnsafeAndInvalidSettings protects public binds, route patterns and resource budgets.
func TestLoadRejectsUnsafeAndInvalidSettings(t *testing.T) {
	for _, tc := range []struct{ name, value string }{{"HTTP_ADDR", ":8080"}, {"MCP_PATH", "/health"}, {"MCP_PATH", "/{wildcard}"}, {"GOOGLE_REQUEST_TIMEOUT", "0s"}, {"GOOGLE_MAX_CONCURRENT", "0"}, {"GOOGLE_MAX_ATTEMPTS", "20"}, {"MCP_MAX_BODY_BYTES", "0"}} {
		t.Run(tc.name+tc.value, func(t *testing.T) {
			t.Setenv("MCP_AUTH_TOKEN", "")
			t.Setenv("MCP_ALLOW_INSECURE", "true")
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/test.json")
			t.Setenv(tc.name, tc.value)
			if _, err := config.Load(); err == nil {
				t.Fatal("unsafe settings accepted")
			}
		})
	}
}
