package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds process-wide settings loaded from the environment.
// It is the only place that reads OS environment variables.
type Config struct {
	// Addr is the TCP host:port the HTTP server binds to.
	Addr string
	// MCPPath is the HTTP path that serves the MCP Streamable transport.
	MCPPath string
	// AuthToken is the shared bearer secret agents must send.
	AuthToken string
	// AllowInsecure disables bearer auth. Use only for local debugging.
	AllowInsecure bool
	// GoogleCredentialsFile is a filesystem path to a service-account JSON key.
	GoogleCredentialsFile string
	// GoogleCredentialsJSON is an inline service-account JSON key.
	GoogleCredentialsJSON string
}

// Load reads configuration from environment variables and applies defaults.
// It returns an error when a production-unsafe combination is detected.
func Load() (Config, error) {
	cfg := Config{
		Addr:          envOr("HTTP_ADDR", ":8080"),
		MCPPath:       envOr("MCP_PATH", "/mcp"),
		AuthToken:     os.Getenv("MCP_AUTH_TOKEN"),
		AllowInsecure: truthy(os.Getenv("MCP_ALLOW_INSECURE")),
		GoogleCredentialsFile: firstNonEmpty(
			os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
			os.Getenv("GOOGLE_CREDENTIALS_FILE"),
		),
		GoogleCredentialsJSON: os.Getenv("GOOGLE_CREDENTIALS_JSON"),
	}

	if !strings.HasPrefix(cfg.MCPPath, "/") {
		return Config{}, fmt.Errorf("MCP_PATH must start with /")
	}
	if !cfg.AllowInsecure && cfg.AuthToken == "" {
		return Config{}, fmt.Errorf("MCP_AUTH_TOKEN is required unless MCP_ALLOW_INSECURE=true")
	}
	if cfg.GoogleCredentialsFile == "" && cfg.GoogleCredentialsJSON == "" {
		return Config{}, fmt.Errorf("set GOOGLE_APPLICATION_CREDENTIALS or GOOGLE_CREDENTIALS_JSON")
	}

	return cfg, nil
}

// envOr returns the environment value or fallback when the variable is empty.
func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// firstNonEmpty returns the first non-empty string from values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// truthy reports whether value looks like an enabled boolean flag.
func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
