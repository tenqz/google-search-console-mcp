package httpserver

import (
	"net/http"
	"time"

	"github.com/tenqz/search-console/internal/auth"
	"github.com/tenqz/search-console/internal/config"
)

// New constructs the public HTTP mux: health is open, MCP is optionally authed.
func New(cfg config.Config, mcpHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
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
