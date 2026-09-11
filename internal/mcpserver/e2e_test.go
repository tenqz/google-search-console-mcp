package mcpserver_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tenqz/google-search-console-mcp/internal/config"
	"github.com/tenqz/google-search-console-mcp/internal/gsc"
	"github.com/tenqz/google-search-console-mcp/internal/httpserver"
	"github.com/tenqz/google-search-console-mcp/internal/mcpserver"
	"golang.org/x/oauth2"
)

type auditRoundTrip func(*http.Request) (*http.Response, error)

func (f auditRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestEndToEnd verifies the real HTTP/auth/MCP/GSC stack against an in-memory Google transport.
// All Google and token requests are intercepted; no real account or external network is used.
func TestEndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var apiCalls, tokenCalls atomic.Int32
	upstream := &http.Client{Transport: auditRoundTrip(func(r *http.Request) (*http.Response, error) {
		status := http.StatusOK
		var payload string
		if r.URL.Host == "oauth.audit.invalid" {
			tokenCalls.Add(1)
			if r.Method != http.MethodPost {
				return nil, fmt.Errorf("token method %s", r.Method)
			}
			payload = `{"access_token":"audit-google-token","token_type":"Bearer","expires_in":3600}`
		} else {
			if r.URL.Host != "www.googleapis.com" && r.URL.Host != "searchconsole.googleapis.com" {
				return nil, fmt.Errorf("unexpected network destination %s", r.URL.Host)
			}
			apiCalls.Add(1)
			if r.Header.Get("Authorization") != "Bearer audit-google-token" {
				return nil, fmt.Errorf("Google auth missing")
			}
			switch {
			case strings.Contains(r.URL.Path, "forbidden.example"):
				status, payload = http.StatusForbidden, `{"error":{"code":403,"message":"audit permission denied"}}`
			case r.URL.Path == "/webmasters/v3/sites":
				payload = `{"siteEntry":[{"siteUrl":"sc-domain:example.com","permissionLevel":"siteFullUser"}]}`
			case strings.HasSuffix(r.URL.Path, "/searchAnalytics/query"):
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					return nil, err
				}
				if r.Method != http.MethodPost || body["startDate"] != "2026-08-01" || body["endDate"] != "2026-08-28" {
					return nil, fmt.Errorf("wrong analytics request: %v", body)
				}
				if !strings.Contains(r.URL.EscapedPath(), "https:%2F%2Fexample.com%2F") {
					return nil, fmt.Errorf("property escaping: %s", r.URL.EscapedPath())
				}
				payload = `{"rows":[{"keys":["example query"],"clicks":12,"impressions":120,"ctr":0.1,"position":4.5}],"responseAggregationType":"byProperty"}`
			case r.URL.Path == "/v1/urlInspection/index:inspect":
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					return nil, err
				}
				if body["inspectionUrl"] != "https://example.com/page" {
					return nil, fmt.Errorf("inspection URL lost")
				}
				payload = `{"inspectionResult":{"indexStatusResult":{"verdict":"PASS","coverageState":"Submitted and indexed","googleCanonical":"https://example.com/page"}}}`
			case strings.HasSuffix(r.URL.Path, "/sitemaps"):
				payload = `{"sitemap":[{"path":"https://example.com/sitemap.xml","errors":"0","warnings":"2"}]}`
			default:
				return nil, fmt.Errorf("unexpected Google path: %s", r.URL.Path)
			}
		}
		return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
	})}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{"client_email": "audit@example.invalid", "private_key": string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})), "token_uri": "https://oauth.audit.invalid/token"})
	if err != nil {
		t.Fatal(err)
	}
	console, err := gsc.NewClientFromJSON(context.WithValue(ctx, oauth2.HTTPClient, upstream), raw)
	if err != nil {
		t.Fatal(err)
	}
	server := mcpserver.New(console)
	transport := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	ts := httptest.NewServer(httpserver.New(config.Config{MCPPath: "/mcp", AuthToken: "audit-agent-token"}, transport))
	defer ts.Close()

	unauthorized, err := ts.Client().Get(ts.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	_ = unauthorized.Body.Close()
	if unauthorized.StatusCode != 401 {
		t.Fatalf("unauthorized status=%d", unauthorized.StatusCode)
	}
	t.Log("missing bearer rejected with 401")

	agentHTTP := &http.Client{Transport: auditRoundTrip(func(r *http.Request) (*http.Response, error) {
		clone := r.Clone(r.Context())
		clone.Header.Set("Authorization", "Bearer audit-agent-token")
		return ts.Client().Transport.RoundTrip(clone)
	})}
	agent := mcp.NewClient(&mcp.Implementation{Name: "architecture-audit", Version: "0.1.0"}, nil)
	session, err := agent.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp", HTTPClient: agentHTTP, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	catalog, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tools) != 4 {
		t.Fatalf("catalog=%d", len(catalog.Tools))
	}
	for _, tool := range catalog.Tools {
		t.Logf("catalog %s outputSchema=%t annotations=%t", tool.Name, tool.OutputSchema != nil, tool.Annotations != nil)
	}

	cases := []struct {
		name     string
		args     map[string]any
		expected []string
	}{
		{"list_sites", map[string]any{}, []string{`"siteUrl": "sc-domain:example.com"`, `"count": 1`}},
		{"query_analytics", map[string]any{"siteUrl": "https://example.com/", "startDate": "2026-08-01", "endDate": "2026-08-28", "dimensions": []string{"query"}}, []string{`"clicks": 12`, `"impressions": 120`, `"ctr": 0.1`, `"position": 4.5`}},
		{"inspect_url", map[string]any{"siteUrl": "https://example.com/", "inspectionUrl": "https://example.com/page"}, []string{`"verdict": "PASS"`, `"googleCanonical": "https://example.com/page"`}},
		{"list_sitemaps", map[string]any{"siteUrl": "https://example.com/"}, []string{`"warnings": 2`, `"count": 1`}},
	}
	for _, tc := range cases {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if result.IsError {
			t.Fatalf("%s tool error: %v", tc.name, result.Content)
		}
		body := text(t, result)
		for _, want := range tc.expected {
			if !strings.Contains(body, want) {
				t.Fatalf("%s missing %s in %s", tc.name, want, body)
			}
		}
		if result.StructuredContent == nil {
			t.Fatalf("%s missing structured content", tc.name)
		}
		t.Logf("%s: HTTP MCP -> real GSC client -> intercepted Google -> expected content + structuredContent", tc.name)
	}
	before := apiCalls.Load()
	invalid, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "query_analytics", Arguments: map[string]any{"siteUrl": ""}})
	if err != nil || !invalid.IsError || apiCalls.Load() != before {
		t.Fatalf("invalid input not contained before Google: %v", err)
	}
	t.Log("invalid input rejected before Google")
	forbidden, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_sitemaps", Arguments: map[string]any{"siteUrl": "sc-domain:forbidden.example"}})
	if err != nil || !forbidden.IsError || !strings.Contains(text(t, forbidden), "403") {
		t.Fatalf("Google failure not returned as tool error: %v", err)
	}
	if tokenCalls.Load() != 1 {
		t.Fatalf("token requests=%d", tokenCalls.Load())
	}
	t.Logf("Google 403 propagated as IsError; OAuth token reused; API calls=%d", apiCalls.Load())
}
