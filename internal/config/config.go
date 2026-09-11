package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds process-wide settings loaded from the environment.
// It is the only place that reads OS environment variables.
type Config struct {
	RequestTimeout time.Duration
	MaxConcurrent  int
	MaxAttempts    int
	MaxBodyBytes   int64
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

	if cfg.AllowInsecure && strings.TrimSpace(os.Getenv("HTTP_ADDR")) == "" {
		cfg.Addr = "127.0.0.1:8080"
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_ADDR must be host:port")
	}
	if cfg.AllowInsecure {
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return Config{}, fmt.Errorf("MCP_ALLOW_INSECURE requires a loopback HTTP_ADDR")
		}
	}
	cfg.RequestTimeout, err = time.ParseDuration(envOr("GOOGLE_REQUEST_TIMEOUT", "30s"))
	if err != nil || cfg.RequestTimeout < time.Second || cfg.RequestTimeout > 5*time.Minute {
		return Config{}, fmt.Errorf("GOOGLE_REQUEST_TIMEOUT must be between 1s and 5m")
	}
	cfg.MaxConcurrent, err = boundedInt("GOOGLE_MAX_CONCURRENT", 8, 1, 128)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxAttempts, err = boundedInt("GOOGLE_MAX_ATTEMPTS", 3, 1, 5)
	if err != nil {
		return Config{}, err
	}
	bodyLimit, err := boundedInt("MCP_MAX_BODY_BYTES", 1<<20, 1024, 16<<20)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxBodyBytes = int64(bodyLimit)
	if !regexp.MustCompile(`^/[a-zA-Z0-9_-]+(?:/[a-zA-Z0-9_-]+)*$`).MatchString(cfg.MCPPath) || cfg.MCPPath == "/health" {
		return Config{}, fmt.Errorf("MCP_PATH must be a literal path distinct from /health")
	}
	if !strings.HasPrefix(cfg.MCPPath, "/") {
		return Config{}, fmt.Errorf("MCP_PATH must start with /")
	}
	if !cfg.AllowInsecure && (strings.TrimSpace(cfg.AuthToken) == "" || cfg.AuthToken == "replace-me-with-a-long-random-token") {
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

func boundedInt(name string, fallback, minValue, maxValue int) (int, error) {
	value, err := strconv.Atoi(envOr(name, strconv.Itoa(fallback)))
	if err != nil || value < minValue || value > maxValue {
		return 0, fmt.Errorf("%s must be between %d and %d", name, minValue, maxValue)
	}
	return value, nil
}
