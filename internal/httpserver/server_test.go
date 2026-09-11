package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tenqz/google-search-console-mcp/internal/config"
	"github.com/tenqz/google-search-console-mcp/internal/httpserver"
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

// TestMCPBodyAndOriginLimits verifies both chunked bodies and authentication ordering.
func TestMCPBodyAndOriginLimits(t *testing.T) {
	handler := httpserver.New(config.Config{MCPPath: "/mcp", AuthToken: "secret", MaxBodyBytes: 8}, okHandler())
	for _, tc := range []struct {
		body, auth, origin string
		want               int
	}{{"123456789", "Bearer secret", "", 413}, {"123456789", "", "", 401}, {"{}", "bearer secret", "", 200}, {"{}", "Bearer secret", "https://evil.example", 403}, {"{}", "Bearer secret", "http://example.com", 200}} {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/mcp", strings.NewReader(tc.body))
		req.ContentLength = -1
		req.Header.Set("Authorization", tc.auth)
		req.Header.Set("Origin", tc.origin)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%+v: status %d", tc, rec.Code)
		}
	}
}
