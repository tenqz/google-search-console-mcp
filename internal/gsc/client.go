package gsc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

const (
	// webmasterScope is the read-only Search Console OAuth scope.
	webmasterScope = "https://www.googleapis.com/auth/webmasters.readonly"
	// defaultWebmasterBase is the Webmasters v3 REST root.
	defaultWebmasterBase = "https://www.googleapis.com/webmasters/v3"
	// defaultInspectURL is the URL Inspection endpoint.
	defaultInspectURL = "https://searchconsole.googleapis.com/v1/urlInspection/index:inspect"
	// defaultTokenURL is used when the service-account JSON omits token_uri.
	defaultTokenURL = "https://oauth2.googleapis.com/token"
)

// serviceAccountFile is the subset of a Google service-account JSON key we need.
type serviceAccountFile struct {
	// ClientEmail is the service account identity added in Search Console.
	ClientEmail string `json:"client_email"`
	// PrivateKey is the PEM-encoded RSA private key.
	PrivateKey string `json:"private_key"`
	// PrivateKeyID is the key id placed in the JWT header.
	PrivateKeyID string `json:"private_key_id"`
	// TokenURI is the OAuth token endpoint.
	TokenURI string `json:"token_uri"`
}

// Client implements Console against Search Console REST endpoints.
type Client struct {
	http          *http.Client
	webmasterBase string
	inspectURL    string
	options       Options
	slots         chan struct{}
}

// NewClientFromJSON builds a Client from a service-account JSON key.
func NewClientFromJSON(ctx context.Context, credentialsJSON []byte, options ...Options) (*Client, error) {
	opts := Options{}.defaults()
	if len(options) > 0 {
		opts = options[0].defaults()
	}
	httpClient, err := httpClientFromJSON(ctx, credentialsJSON, opts.RequestTimeout)
	if err != nil {
		return nil, err
	}
	return NewClient(httpClient, opts), nil
}

// NewClientFromFile builds a Client from a service-account JSON file path.
func NewClientFromFile(ctx context.Context, credentialsFile string, options ...Options) (*Client, error) {
	raw, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read google credentials: %w", err)
	}
	return NewClientFromJSON(ctx, raw, options...)
}

// ListSites returns properties visible to the service account.
func (c *Client) ListSites(ctx context.Context) ([]Site, error) {
	var payload struct {
		SiteEntry []struct {
			SiteURL         string `json:"siteUrl"`
			PermissionLevel string `json:"permissionLevel"`
		} `json:"siteEntry"`
	}
	if err := c.getJSON(ctx, c.webmasterBase+"/sites", &payload); err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	sites := make([]Site, 0, len(payload.SiteEntry))
	for _, entry := range payload.SiteEntry {
		sites = append(sites, Site{URL: entry.SiteURL, PermissionLevel: entry.PermissionLevel})
	}
	return sites, nil
}

// QueryAnalytics runs a Search Analytics query and maps rows to domain types.
func (c *Client) QueryAnalytics(ctx context.Context, query AnalyticsQuery) (AnalyticsResult, error) {
	body := analyticsRequest{
		StartDate:  query.StartDate,
		EndDate:    query.EndDate,
		Dimensions: query.Dimensions,
		RowLimit:   query.RowLimit,
		StartRow:   query.StartRow,
		SearchType: query.SearchType,
		DataState:  query.DataState,
	}
	if len(query.Filters) > 0 {
		filters := make([]apiDimensionFilter, 0, len(query.Filters))
		for _, filter := range query.Filters {
			filters = append(filters, apiDimensionFilter(filter))
		}
		body.DimensionFilterGroups = []apiDimensionFilterGroup{{Filters: filters}}
	}

	endpoint := c.webmasterBase + "/sites/" + url.PathEscape(query.SiteURL) + "/searchAnalytics/query"
	var payload analyticsResponse
	if err := c.postJSON(ctx, endpoint, body, &payload); err != nil {
		return AnalyticsResult{}, fmt.Errorf("query search analytics: %w", err)
	}
	return mapAnalyticsResult(payload), nil
}

// InspectURL inspects one page against a Search Console property.
func (c *Client) InspectURL(ctx context.Context, query InspectQuery) (InspectResult, error) {
	var payload inspectResponse
	if err := c.postJSON(ctx, c.inspectURL, inspectRequest(query), &payload); err != nil {
		return InspectResult{}, fmt.Errorf("inspect url: %w", err)
	}
	return mapInspectResult(payload), nil
}

// ListSitemaps returns sitemaps Google knows for the property.
func (c *Client) ListSitemaps(ctx context.Context, siteURL string) ([]Sitemap, error) {
	var payload struct {
		Sitemap []sitemapEntry `json:"sitemap"`
	}
	endpoint := c.webmasterBase + "/sites/" + url.PathEscape(siteURL) + "/sitemaps"
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("list sitemaps: %w", err)
	}
	out := make([]Sitemap, 0, len(payload.Sitemap))
	for _, entry := range payload.Sitemap {
		out = append(out, Sitemap(entry))
	}
	return out, nil
}

type analyticsRequest struct {
	StartDate             string                    `json:"startDate"`
	EndDate               string                    `json:"endDate"`
	Dimensions            []string                  `json:"dimensions,omitempty"`
	RowLimit              int64                     `json:"rowLimit,omitempty"`
	StartRow              int64                     `json:"startRow,omitempty"`
	SearchType            string                    `json:"searchType,omitempty"`
	DataState             string                    `json:"dataState,omitempty"`
	DimensionFilterGroups []apiDimensionFilterGroup `json:"dimensionFilterGroups,omitempty"`
}

type apiDimensionFilterGroup struct {
	Filters []apiDimensionFilter `json:"filters"`
}

type apiDimensionFilter struct {
	Dimension  string `json:"dimension"`
	Operator   string `json:"operator"`
	Expression string `json:"expression"`
}

type analyticsResponse struct {
	Rows                    []analyticsRowJSON `json:"rows"`
	ResponseAggregationType string             `json:"responseAggregationType"`
}

type analyticsRowJSON struct {
	Keys        []string `json:"keys"`
	Clicks      float64  `json:"clicks"`
	Impressions float64  `json:"impressions"`
	CTR         float64  `json:"ctr"`
	Position    float64  `json:"position"`
}

type inspectRequest struct {
	SiteURL       string `json:"siteUrl"`
	InspectionURL string `json:"inspectionUrl"`
	LanguageCode  string `json:"languageCode,omitempty"`
}

type inspectResponse struct {
	InspectionResult *inspectResultJSON `json:"inspectionResult"`
}

type inspectResultJSON struct {
	InspectionResultLink string `json:"inspectionResultLink"`
	IndexStatusResult    *struct {
		CoverageState   string   `json:"coverageState"`
		Verdict         string   `json:"verdict"`
		IndexingState   string   `json:"indexingState"`
		PageFetchState  string   `json:"pageFetchState"`
		RobotsTxtState  string   `json:"robotsTxtState"`
		CrawledAs       string   `json:"crawledAs"`
		LastCrawlTime   string   `json:"lastCrawlTime"`
		GoogleCanonical string   `json:"googleCanonical"`
		UserCanonical   string   `json:"userCanonical"`
		ReferringURLs   []string `json:"referringUrls"`
		Sitemap         []string `json:"sitemap"`
	} `json:"indexStatusResult"`
	MobileUsabilityResult *struct {
		Verdict string `json:"verdict"`
	} `json:"mobileUsabilityResult"`
}

type sitemapEntry struct {
	Path            string `json:"path"`
	Type            string `json:"type"`
	LastSubmitted   string `json:"lastSubmitted"`
	LastDownloaded  string `json:"lastDownloaded"`
	IsPending       bool   `json:"isPending"`
	IsSitemapsIndex bool   `json:"isSitemapsIndex"`
	Errors          int64  `json:"errors,string"`
	Warnings        int64  `json:"warnings,string"`
}

func mapAnalyticsResult(resp analyticsResponse) AnalyticsResult {
	rows := make([]AnalyticsRow, 0, len(resp.Rows))
	for _, row := range resp.Rows {
		rows = append(rows, AnalyticsRow(row))
	}
	return AnalyticsResult{
		Rows:                    rows,
		RowCount:                len(rows),
		ResponseAggregationType: resp.ResponseAggregationType,
	}
}

func mapInspectResult(resp inspectResponse) InspectResult {
	if resp.InspectionResult == nil {
		return InspectResult{}
	}
	out := InspectResult{InspectionResultLink: resp.InspectionResult.InspectionResultLink}
	if status := resp.InspectionResult.IndexStatusResult; status != nil {
		out.CoverageState = status.CoverageState
		out.Verdict = status.Verdict
		out.IndexingState = status.IndexingState
		out.PageFetchState = status.PageFetchState
		out.RobotsTxtState = status.RobotsTxtState
		out.CrawledAs = status.CrawledAs
		out.LastCrawlTime = status.LastCrawlTime
		out.GoogleCanonical = status.GoogleCanonical
		out.UserCanonical = status.UserCanonical
		out.ReferringURLs = status.ReferringURLs
		out.Sitemaps = status.Sitemap
	}
	if mobile := resp.InspectionResult.MobileUsabilityResult; mobile != nil {
		out.MobileUsabilityVerdict = mobile.Verdict
	}
	return out
}
