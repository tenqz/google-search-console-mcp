package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

// Bearer protects next with a shared secret.
// Accepted headers: Authorization: Bearer <token> or X-API-Key: <token>.
// Health checks must be registered outside this middleware.
type Bearer struct {
	token string
	next  http.Handler
}

// NewBearer wraps next so only callers with token may proceed.
func NewBearer(token string, next http.Handler) *Bearer {
	return &Bearer{token: token, next: next}
}

// ServeHTTP rejects missing or mismatched credentials with 401.
func (b *Bearer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !tokenMatches(b.token, extractToken(r)) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="mcp"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b.next.ServeHTTP(w, r)
}

// extractToken reads the shared secret from well-known HTTP headers.
func extractToken(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

// tokenMatches compares secrets in constant time after hashing.
func tokenMatches(expected, got string) bool {
	if expected == "" || got == "" {
		return false
	}
	sumExpected := sha256.Sum256([]byte(expected))
	sumGot := sha256.Sum256([]byte(got))
	return subtle.ConstantTimeCompare(sumExpected[:], sumGot[:]) == 1
}
