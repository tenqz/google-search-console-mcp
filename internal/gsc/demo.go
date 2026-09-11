package gsc

import "context"

const demoSite = "sc-domain:example.invalid"

// NewDemo provides deterministic fixtures without Google credentials or network access.
func NewDemo() Console { return demoConsole{} }

type demoConsole struct{}

func (demoConsole) DemoMode() bool { return true }

func (demoConsole) ListSites(ctx context.Context) ([]Site, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []Site{{URL: demoSite, PermissionLevel: "siteFullUser"}}, nil
}

func (demoConsole) QueryAnalytics(ctx context.Context, q AnalyticsQuery) (AnalyticsResult, error) {
	if err := demoAccess(ctx, q.SiteURL); err != nil {
		return AnalyticsResult{}, err
	}
	result := AnalyticsResult{StartDate: q.StartDate, EndDate: q.EndDate, DataState: q.DataState, Dimensions: append([]string{}, q.Dimensions...), Rows: []AnalyticsRow{}, ResponseAggregationType: "auto", DataNotice: "DEMO: deterministic sample data; not a Google API response."}
	if q.StartRow > 0 {
		return result, nil
	}
	values := map[string]string{"query": "example search", "page": "https://example.invalid/guide", "country": "usa", "device": "DESKTOP", "date": q.StartDate, "hour": q.StartDate + "T00:00:00", "searchAppearance": "AMP_BLUE_LINK"}
	keys := []string{}
	for _, dimension := range q.Dimensions {
		keys = append(keys, values[dimension])
	}
	result.Rows = []AnalyticsRow{{Keys: keys, Clicks: 120, Impressions: 2400, CTR: 0.05, Position: 7.2}}
	result.RowCount = 1
	return result, nil
}

func (demoConsole) InspectURL(ctx context.Context, q InspectQuery) (InspectResult, error) {
	if err := demoAccess(ctx, q.SiteURL); err != nil {
		return InspectResult{}, err
	}
	return InspectResult{CoverageState: "Submitted and indexed (demo)", Verdict: "PASS", IndexingState: "INDEXING_ALLOWED", PageFetchState: "SUCCESSFUL", RobotsTxtState: "ALLOWED", GoogleCanonical: q.InspectionURL, UserCanonical: q.InspectionURL}, nil
}

func (demoConsole) ListSitemaps(ctx context.Context, site string) ([]Sitemap, error) {
	if err := demoAccess(ctx, site); err != nil {
		return nil, err
	}
	return []Sitemap{{Path: "https://example.invalid/sitemap.xml", Type: "SITEMAP"}}, nil
}

func demoAccess(ctx context.Context, site string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if site != demoSite {
		return &RequestError{Kind: "permission_denied", Status: 403}
	}
	return nil
}
