package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/jsonschema-go/jsonschema"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tenqz/google-search-console-mcp/internal/gsc"
)

const (
	// ServerName is the MCP implementation name advertised to agents.
	ServerName = "google-search-console"
	// ServerVersion is the semantic version of this MCP server.
	ServerVersion = "0.1.0"
)

// ListSitesInput is the MCP argument object for list_sites.
// The tool has no required fields; an empty object is valid.
type ListSitesInput struct{}

// QueryAnalyticsInput is the MCP argument object for query_analytics.
type QueryAnalyticsInput struct {
	// SiteURL is the property identifier from Search Console.
	SiteURL string `json:"siteUrl" jsonschema:"Search Console property URL, e.g. https://example.com/ or sc-domain:example.com"`
	// StartDate is inclusive YYYY-MM-DD. Defaults to 27 days before endDate.
	StartDate string `json:"startDate,omitempty" jsonschema:"Inclusive Pacific Time start YYYY-MM-DD; defaults to 27 days before endDate"`
	// EndDate is inclusive YYYY-MM-DD. Defaults to yesterday.
	EndDate string `json:"endDate,omitempty" jsonschema:"Inclusive Pacific Time end YYYY-MM-DD; defaults to yesterday"`
	// Dimensions groups rows: query, page, date, country, device, searchAppearance.
	Dimensions []string `json:"dimensions,omitempty" jsonschema:"Group by query, page, date, country, device, searchAppearance or hour; no duplicates"`
	// Filters optional dimension filters applied together.
	Filters []gsc.DimensionFilter `json:"filters,omitempty" jsonschema:"Dimension filters"`
	// RowLimit maximum rows, 1-25000. Google default is 1000.
	RowLimit int64 `json:"rowLimit,omitempty" jsonschema:"Maximum rows to return (1-25000)"`
	// StartRow zero-based offset for pagination.
	StartRow int64 `json:"startRow,omitempty" jsonschema:"Zero-based row offset"`
	// SearchType WEB, IMAGE, VIDEO, NEWS, DISCOVER, or GOOGLE_NEWS.
	SearchType string `json:"searchType,omitempty" jsonschema:"Search type: WEB, IMAGE, VIDEO, NEWS, DISCOVER, GOOGLE_NEWS"`
	// DataState FINAL or ALL.
	DataState       string `json:"dataState,omitempty" jsonschema:"final, all, or hourly_all; hourly_all requires the hour dimension"`
	AggregationType string `json:"aggregationType,omitempty" jsonschema:"auto, byPage, byProperty, or byNewsShowcasePanel"`
}

// InspectURLInput is the MCP argument object for inspect_url.
type InspectURLInput struct {
	// SiteURL is the property URL exactly as registered in Search Console.
	SiteURL string `json:"siteUrl" jsonschema:"Search Console property URL, including trailing slash for URL-prefix properties"`
	// InspectionURL is the fully-qualified page to inspect.
	InspectionURL string `json:"inspectionUrl" jsonschema:"Fully-qualified URL to inspect"`
	// LanguageCode is optional BCP-47 for translated messages.
	LanguageCode string `json:"languageCode,omitempty" jsonschema:"Optional BCP-47 language code, e.g. en-US"`
}

// ListSitemapsInput is the MCP argument object for list_sitemaps.
type ListSitemapsInput struct {
	// SiteURL is the property URL from Search Console.
	SiteURL string `json:"siteUrl" jsonschema:"Search Console property URL"`
}

// New builds an MCP server with Search Console tools registered.
func New(console gsc.Console) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, nil)

	tools := &Toolset{Console: console, Now: time.Now}
	destructive := false
	annotations := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, DestructiveHint: &destructive}
	mcp.AddTool(server, &mcp.Tool{
		Name:         "list_sites",
		OutputSchema: schemaFor[SitesOutput](),
		Annotations:  annotations,
		Description:  "List Google Search Console properties the service account can access, including permission level.",
	}, tools.ListSites)
	mcp.AddTool(server, &mcp.Tool{
		Name:         "query_analytics",
		OutputSchema: schemaFor[gsc.AnalyticsResult](),
		Annotations:  annotations,
		Description:  "Query Google Search performance (clicks, impressions, CTR, position) for a Search Console property. Use list_sites first if siteUrl is unknown.",
	}, tools.QueryAnalytics)
	mcp.AddTool(server, &mcp.Tool{
		Name:         "inspect_url",
		OutputSchema: schemaFor[gsc.InspectResult](),
		Annotations:  annotations,
		Description:  "Inspect how Google indexes a URL for a Search Console property (coverage, fetch, robots, canonicals).",
	}, tools.InspectURL)
	mcp.AddTool(server, &mcp.Tool{
		Name:         "list_sitemaps",
		OutputSchema: schemaFor[SitemapsOutput](),
		Annotations:  annotations,
		Description:  "List sitemaps known to Google Search Console for a property.",
	}, tools.ListSitemaps)

	return server
}

// Toolset holds dependencies for MCP tool handlers.
type Toolset struct {
	// Console talks to Search Console.
	Console gsc.Console
	// Now returns the current time; tests inject a fixed clock.
	Now func() time.Time
}

// ListSites returns every Search Console property visible to the credentials.
func (t *Toolset) ListSites(ctx context.Context, _ *mcp.CallToolRequest, _ ListSitesInput) (*mcp.CallToolResult, any, error) {
	sites, err := t.Console.ListSites(ctx)
	if err != nil {
		return toolFailure(err)
	}
	return jsonResult(SitesOutput{Sites: sites, Count: len(sites)})
}

// QueryAnalytics returns Search performance rows for the given property and range.
func (t *Toolset) QueryAnalytics(ctx context.Context, _ *mcp.CallToolRequest, in QueryAnalyticsInput) (*mcp.CallToolResult, any, error) {
	query, err := gsc.ValidateAnalyticsQuery(gsc.AnalyticsQuery{
		SiteURL:         in.SiteURL,
		StartDate:       in.StartDate,
		EndDate:         in.EndDate,
		Dimensions:      in.Dimensions,
		Filters:         in.Filters,
		RowLimit:        in.RowLimit,
		StartRow:        in.StartRow,
		SearchType:      in.SearchType,
		DataState:       in.DataState,
		AggregationType: in.AggregationType,
	}, t.Now())
	if err != nil {
		return toolError("%v", err)
	}
	result, err := t.Console.QueryAnalytics(ctx, query)
	if err != nil {
		return toolFailure(err)
	}
	return jsonResult(result)
}

// InspectURL returns Google index status for a single page under a property.
func (t *Toolset) InspectURL(ctx context.Context, _ *mcp.CallToolRequest, in InspectURLInput) (*mcp.CallToolResult, any, error) {
	query, err := gsc.ValidateInspectQuery(gsc.InspectQuery{
		SiteURL:       in.SiteURL,
		InspectionURL: in.InspectionURL,
		LanguageCode:  in.LanguageCode,
	})
	if err != nil {
		return toolError("%v", err)
	}
	result, err := t.Console.InspectURL(ctx, query)
	if err != nil {
		return toolFailure(err)
	}
	return jsonResult(result)
}

// ListSitemaps returns sitemaps Google knows for the property.
func (t *Toolset) ListSitemaps(ctx context.Context, _ *mcp.CallToolRequest, in ListSitemapsInput) (*mcp.CallToolResult, any, error) {
	siteURL := strings.TrimSpace(in.SiteURL)
	if siteURL == "" {
		return toolError("siteUrl is required")
	}
	sitemaps, err := t.Console.ListSitemaps(ctx, siteURL)
	if err != nil {
		return toolFailure(err)
	}
	return jsonResult(SitemapsOutput{Sitemaps: sitemaps, Count: len(sitemaps)})
}

// jsonResult encodes value as JSON text content for the MCP client.
func jsonResult(value any) (*mcp.CallToolResult, any, error) {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("encode tool result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, value, nil
}

// SitesOutput is the stable result of list_sites.
type SitesOutput struct {
	Sites []gsc.Site `json:"sites"`
	Count int        `json:"count"`
}

// SitemapsOutput is the stable result of list_sitemaps.
type SitemapsOutput struct {
	Sitemaps []gsc.Sitemap `json:"sitemaps"`
	Count    int           `json:"count"`
}

func schemaFor[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Errorf("MCP output schema: %w", err))
	}
	return schema
}

// toolError classifies local argument errors separately from Google failures.
func toolError(format string, args ...any) (*mcp.CallToolResult, any, error) {
	return toolFailure(fmt.Errorf(format, args...))
}

func toolFailure(err error) (*mcp.CallToolResult, any, error) {
	payload := map[string]any{"code": "invalid_argument", "message": err.Error(), "retryable": false}
	var failure *gsc.RequestError
	if errors.As(err, &failure) {
		payload["code"], payload["retryable"] = failure.Kind, failure.Retryable
		if failure.Status != 0 {
			payload["httpStatus"] = failure.Status
		}
		if failure.RetryAfterSeconds > 0 {
			payload["retryAfterSeconds"] = failure.RetryAfterSeconds
		}
	}
	wrapped := map[string]any{"error": payload}
	raw, _ := json.Marshal(wrapped)
	// An absent typed output lets execution errors bypass the success output schema.
	return &mcp.CallToolResult{IsError: true, StructuredContent: wrapped, Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil, nil
}
