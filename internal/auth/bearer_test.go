package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tenqz/search-console/internal/auth"
)

// TestBearerRejectsMissingToken documents that anonymous MCP calls are denied.
func TestBearerRejectsMissingToken(t *testing.T) {
	handler := auth.NewBearer("secret", okHandler())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// TestBearerAcceptsAuthorizationHeader documents the standard Bearer header agents should send.
func TestBearerAcceptsAuthorizationHeader(t *testing.T) {
	handler := auth.NewBearer("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// TestBearerAcceptsAPIKeyHeader documents X-API-Key as an alternative for MCP clients.
func TestBearerAcceptsAPIKeyHeader(t *testing.T) {
	handler := auth.NewBearer("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("X-API-Key", "secret")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// TestBearerRejectsWrongToken documents that a mismatched secret is still unauthorized.
func TestBearerRejectsWrongToken(t *testing.T) {
	handler := auth.NewBearer("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer other")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
