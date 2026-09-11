package httpserver

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tenqz/google-search-console-mcp/internal/auth"
	"github.com/tenqz/google-search-console-mcp/internal/config"
)

// New constructs the public HTTP mux: health is open, MCP is optionally authed.
func New(cfg config.Config, mcpHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mcpHandler = boundedRequest(cfg.MaxBodyBytes, mcpHandler)
	mux.HandleFunc("GET /health", health)
	if cfg.AllowInsecure {
		mux.Handle(cfg.MCPPath, mcpHandler)
	} else {
		mux.Handle(cfg.MCPPath, auth.NewBearer(cfg.AuthToken, mcpHandler))
	}
	return mux
}

// NewServer wraps handler in an HTTP server with conservative timeouts.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// health reports that the process is accepting HTTP traffic.
func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// boundedRequest runs after bearer auth and rejects large or cross-origin requests before the SDK reads them.
func boundedRequest(limit int64, next http.Handler) http.Handler {
	if limit == 0 {
		limit = 1 << 20
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host != r.Host || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
		}
		if r.Body != nil {
			raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
			if err != nil {
				var oversized *http.MaxBytesError
				if errors.As(err, &oversized) {
					http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				} else {
					http.Error(w, "invalid request body", http.StatusBadRequest)
				}
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
		}
		next.ServeHTTP(w, r)
	})
}
