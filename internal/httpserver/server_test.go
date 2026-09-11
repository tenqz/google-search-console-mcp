package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tenqz/search-console/internal/config"
	"github.com/tenqz/search-console/internal/httpserver"
)

// TestHealthIsPublic documents that Docker healthchecks do not need a bearer token.
func TestHealthIsPublic(t *testing.T) {
	handler := httpserver.New(config.Config{MCPPath: "/mcp", AuthToken: "secret"}, okHandler())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// TestMCPRequiresBearer documents that the MCP path is protected when a token is configured.
func TestMCPRequiresBearer(t *testing.T) {
	handler := httpserver.New(config.Config{MCPPath: "/mcp", AuthToken: "secret"}, okHandler())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// TestMCPAllowInsecure documents the local-debug bypass for the MCP path.
func TestMCPAllowInsecure(t *testing.T) {
	handler := httpserver.New(config.Config{MCPPath: "/mcp", AllowInsecure: true}, okHandler())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
