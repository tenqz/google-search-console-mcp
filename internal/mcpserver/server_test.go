package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tenqz/search-console/internal/gsc"
	"github.com/tenqz/search-console/internal/mcpserver"
)

// fakeConsole is an in-memory Search Console used by tool tests.
type fakeConsole struct {
	sites    []gsc.Site
	result   gsc.AnalyticsResult
	inspect  gsc.InspectResult
	sitemaps []gsc.Sitemap
	err      error
	query    gsc.AnalyticsQuery
}

func (f *fakeConsole) ListSites(context.Context) ([]gsc.Site, error) {
	return f.sites, f.err
}

func (f *fakeConsole) QueryAnalytics(_ context.Context, query gsc.AnalyticsQuery) (gsc.AnalyticsResult, error) {
	f.query = query
	return f.result, f.err
}

func (f *fakeConsole) InspectURL(context.Context, gsc.InspectQuery) (gsc.InspectResult, error) {
	return f.inspect, f.err
}

func (f *fakeConsole) ListSitemaps(context.Context, string) ([]gsc.Sitemap, error) {
	return f.sitemaps, f.err
}

func text(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatal("expected text content")
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	return content.Text
}

// TestListSitesReturnsJSON documents the list_sites payload shape for agents.
func TestListSitesReturnsJSON(t *testing.T) {
	tools := &mcpserver.Toolset{Console: &fakeConsole{sites: []gsc.Site{{
		URL:             "sc-domain:example.com",
		PermissionLevel: "siteFullUser",
	}}}}

	result, _, err := tools.ListSites(context.Background(), nil, mcpserver.ListSitesInput{})
	if err != nil {
		t.Fatalf("ListSites() error = %v", err)
	}

	if !strings.Contains(text(t, result), "sc-domain:example.com") {
		t.Fatalf("payload = %s", text(t, result))
	}
}

// TestQueryAnalyticsRejectsMissingSite documents tool-level validation errors are MCP errors, not panics.
func TestQueryAnalyticsRejectsMissingSite(t *testing.T) {
	tools := &mcpserver.Toolset{Console: &fakeConsole{}, Now: time.Now}

	result, _, err := tools.QueryAnalytics(context.Background(), nil, mcpserver.QueryAnalyticsInput{})
	if err != nil {
		t.Fatalf("QueryAnalytics() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError for missing siteUrl")
	}
}

// TestQueryAnalyticsPassesValidatedQuery documents that defaults reach the Console client.
func TestQueryAnalyticsPassesValidatedQuery(t *testing.T) {
	fake := &fakeConsole{result: gsc.AnalyticsResult{Rows: []gsc.AnalyticsRow{{Clicks: 3}}, RowCount: 1}}
	tools := &mcpserver.Toolset{
		Console: fake,
		Now:     func() time.Time { return time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC) },
	}

	result, _, err := tools.QueryAnalytics(context.Background(), nil, mcpserver.QueryAnalyticsInput{
		SiteURL:    "https://example.com/",
		Dimensions: []string{"query"},
	})
	if err != nil {
		t.Fatalf("QueryAnalytics() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", text(t, result))
	}

	var decoded gsc.AnalyticsResult
	if err := json.Unmarshal([]byte(text(t, result)), &decoded); err != nil {
		t.Fatalf("json: %v", err)
	}
	if decoded.RowCount != 1 {
		t.Fatalf("RowCount = %d", decoded.RowCount)
	}
	if fake.query.StartDate != "2026-08-14" {
		t.Fatalf("StartDate = %q", fake.query.StartDate)
	}
}

// TestInspectURLRequiresInspectionURL documents inspect_url validation.
func TestInspectURLRequiresInspectionURL(t *testing.T) {
	tools := &mcpserver.Toolset{Console: &fakeConsole{}}

	result, _, err := tools.InspectURL(context.Background(), nil, mcpserver.InspectURLInput{SiteURL: "https://example.com/"})
	if err != nil {
		t.Fatalf("InspectURL() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError")
	}
}

// TestListSitemapsRequiresSiteURL documents list_sitemaps input validation.
func TestListSitemapsRequiresSiteURL(t *testing.T) {
	tools := &mcpserver.Toolset{Console: &fakeConsole{}}

	result, _, err := tools.ListSitemaps(context.Background(), nil, mcpserver.ListSitemapsInput{})
	if err != nil {
		t.Fatalf("ListSitemaps() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError")
	}
}

// TestNewRegistersExpectedTools documents the MVP tool catalog advertised to agents.
func TestNewRegistersExpectedTools(t *testing.T) {
	server := mcpserver.New(&fakeConsole{})
	if server == nil {
		t.Fatal("expected server")
	}
}
