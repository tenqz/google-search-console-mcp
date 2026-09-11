package gsc

import "context"

// Console is the Search Console capability used by MCP tools.
// Implementations talk to Google; tests provide an in-memory fake.
type Console interface {
	// ListSites returns properties the credentials can access.
	ListSites(ctx context.Context) ([]Site, error)
	// QueryAnalytics returns Search performance rows for a property.
	QueryAnalytics(ctx context.Context, query AnalyticsQuery) (AnalyticsResult, error)
	// InspectURL returns the Google index status for a single page.
	InspectURL(ctx context.Context, query InspectQuery) (InspectResult, error)
	// ListSitemaps returns sitemaps known for a property.
	ListSitemaps(ctx context.Context, siteURL string) ([]Sitemap, error)
}

// Site is a Search Console property the service account can see.
type Site struct {
	// URL is the property identifier, e.g. https://example.com/ or sc-domain:example.com.
	URL string `json:"siteUrl"`
	// PermissionLevel is the access level granted to the credentials.
	PermissionLevel string `json:"permissionLevel"`
}

// DimensionFilter restricts analytics rows by a single dimension.
type DimensionFilter struct {
	// Dimension is the dimension name: query, page, country, device, searchAppearance.
	Dimension string `json:"dimension"`
	// Operator is the comparison: equals, contains, notContains, includingRegex, excludingRegex.
	Operator string `json:"operator"`
	// Expression is the value compared against the dimension.
	Expression string `json:"expression"`
}

// AnalyticsQuery is a Search Analytics request in domain terms.
type AnalyticsQuery struct {
	// SiteURL is the property URL as shown in Search Console.
	SiteURL string `json:"siteUrl"`
	// StartDate is the inclusive range start in YYYY-MM-DD (PST).
	StartDate string `json:"startDate"`
	// EndDate is the inclusive range end in YYYY-MM-DD (PST).
	EndDate string `json:"endDate"`
	// Dimensions groups rows, e.g. query, page, date, country, device.
	Dimensions []string `json:"dimensions,omitempty"`
	// Filters are optional dimension filters applied together.
	Filters []DimensionFilter `json:"filters,omitempty"`
	// RowLimit is the maximum number of rows (1–25000). Zero means Google default.
	RowLimit int64 `json:"rowLimit,omitempty"`
	// StartRow is the zero-based offset for pagination.
	StartRow int64 `json:"startRow,omitempty"`
	// SearchType filters the search surface: WEB, IMAGE, VIDEO, NEWS, DISCOVER, GOOGLE_NEWS.
	SearchType string `json:"searchType,omitempty"`
	// DataState selects FINAL or ALL (includes fresh/unfinalized data).
	DataState string `json:"dataState,omitempty"`
}

// AnalyticsRow is one aggregated Search performance line.
type AnalyticsRow struct {
	// Keys holds dimension values in the same order as the request dimensions.
	Keys []string `json:"keys,omitempty"`
	// Clicks is the number of search clicks.
	Clicks float64 `json:"clicks"`
	// Impressions is the number of search impressions.
	Impressions float64 `json:"impressions"`
	// CTR is clicks divided by impressions.
	CTR float64 `json:"ctr"`
	// Position is the average ranking position.
	Position float64 `json:"position"`
}

// AnalyticsResult is a Search Analytics response plus response metadata.
type AnalyticsResult struct {
	// Rows are the matching performance lines. Empty when there is no data.
	Rows []AnalyticsRow `json:"rows"`
	// RowCount is the number of rows returned.
	RowCount int `json:"rowCount"`
	// ResponseAggregationType is how Google aggregated the data, when present.
	ResponseAggregationType string `json:"responseAggregationType,omitempty"`
}

// InspectQuery asks Google how a URL is indexed for a property.
type InspectQuery struct {
	// SiteURL is the property URL exactly as registered in Search Console.
	SiteURL string `json:"siteUrl"`
	// InspectionURL is the fully-qualified page URL to inspect.
	InspectionURL string `json:"inspectionUrl"`
	// LanguageCode is an optional BCP-47 code for translated issue messages.
	LanguageCode string `json:"languageCode,omitempty"`
}

// InspectResult is a compact view of URL Inspection that agents can read.
type InspectResult struct {
	// InspectionResultLink opens the same inspection in the Search Console UI.
	InspectionResultLink string `json:"inspectionResultLink,omitempty"`
	// CoverageState is the high-level indexing coverage label.
	CoverageState string `json:"coverageState,omitempty"`
	// Verdict is PASS, NEUTRAL, FAIL, or unspecified.
	Verdict string `json:"verdict,omitempty"`
	// IndexingState reports whether indexing is allowed or blocked.
	IndexingState string `json:"indexingState,omitempty"`
	// PageFetchState reports whether Google could fetch the page.
	PageFetchState string `json:"pageFetchState,omitempty"`
	// RobotsTxtState reports robots.txt allow/disallow for Googlebot.
	RobotsTxtState string `json:"robotsTxtState,omitempty"`
	// CrawledAs is DESKTOP or MOBILE.
	CrawledAs string `json:"crawledAs,omitempty"`
	// LastCrawlTime is the last successful primary crawl, RFC 3339 when present.
	LastCrawlTime string `json:"lastCrawlTime,omitempty"`
	// GoogleCanonical is the canonical URL Google selected.
	GoogleCanonical string `json:"googleCanonical,omitempty"`
	// UserCanonical is the canonical URL declared by the page.
	UserCanonical string `json:"userCanonical,omitempty"`
	// ReferringURLs are known URLs that link to the inspected URL.
	ReferringURLs []string `json:"referringUrls,omitempty"`
	// Sitemaps lists sitemaps Google associated with the URL.
	Sitemaps []string `json:"sitemaps,omitempty"`
	// MobileUsabilityVerdict is the mobile usability verdict when present.
	MobileUsabilityVerdict string `json:"mobileUsabilityVerdict,omitempty"`
}

// Sitemap is a sitemap URL known to Search Console for a property.
type Sitemap struct {
	// Path is the sitemap URL.
	Path string `json:"path"`
	// Type is the sitemap type, e.g. SITEMAP or RSS_FEED.
	Type string `json:"type,omitempty"`
	// LastSubmitted is when the sitemap was last submitted.
	LastSubmitted string `json:"lastSubmitted,omitempty"`
	// LastDownloaded is when Google last downloaded the sitemap.
	LastDownloaded string `json:"lastDownloaded,omitempty"`
	// IsPending is true while Google has not finished processing it.
	IsPending bool `json:"isPending,omitempty"`
	// IsSitemapsIndex is true when the file is a sitemap index.
	IsSitemapsIndex bool `json:"isSitemapsIndex,omitempty"`
	// Errors is the number of sitemap-level errors.
	Errors int64 `json:"errors,omitempty"`
	// Warnings is the number of sitemap-level warnings.
	Warnings int64 `json:"warnings,omitempty"`
}
