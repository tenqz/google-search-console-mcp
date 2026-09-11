package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tenqz/google-search-console-mcp/internal/gsc"
	"github.com/tenqz/google-search-console-mcp/internal/mcpserver"
)

func checkHealth(endpoint string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: HTTP %d", resp.StatusCode)
	}
	return nil
}

type bearerTransport struct{ token string }

func (t bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(clone)
}

// smoke uses the same public MCP protocol as an agent, without requiring a specific desktop app.
func smoke(endpoint, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "gsc-mcp-smoke", Version: mcpserver.ServerVersion}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: &http.Client{Transport: bearerTransport{token}, Timeout: 45 * time.Second}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()
	catalog, err := session.ListTools(ctx, nil)
	if err != nil {
		return err
	}
	if len(catalog.Tools) != 4 {
		return fmt.Errorf("expected four tools, got %d", len(catalog.Tools))
	}
	for _, tool := range catalog.Tools {
		if tool.OutputSchema == nil || tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			return fmt.Errorf("missing output/read-only contract: %s", tool.Name)
		}
	}
	call := func(name string, args any, out any) error {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			return err
		}
		if result.IsError {
			return fmt.Errorf("%s failed: %v", name, result.Content)
		}
		raw, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return err
		}
		if string(raw) == "null" {
			return fmt.Errorf("%s has no structured result", name)
		}
		if out != nil {
			if err := json.Unmarshal(raw, out); err != nil {
				return err
			}
		}
		fmt.Println(name + ": ok")
		return nil
	}
	var sites mcpserver.SitesOutput
	if err := call("list_sites", map[string]any{}, &sites); err != nil {
		return err
	}
	if len(sites.Sites) == 0 {
		return fmt.Errorf("no properties: invite the service account in Search Console")
	}
	site := sites.Sites[0].URL
	var analytics gsc.AnalyticsResult
	if err := call("query_analytics", map[string]any{"siteUrl": site, "rowLimit": 1}, &analytics); err != nil {
		return err
	}
	inspection := site
	if domain, ok := strings.CutPrefix(site, "sc-domain:"); ok {
		inspection = "https://" + domain + "/"
	}
	if parsed, err := url.Parse(inspection); err != nil || parsed.Host == "" {
		return fmt.Errorf("cannot derive inspection URL")
	}
	if err := call("inspect_url", map[string]any{"siteUrl": site, "inspectionUrl": inspection}, nil); err != nil {
		return err
	}
	return call("list_sitemaps", map[string]any{"siteUrl": site}, nil)
}
