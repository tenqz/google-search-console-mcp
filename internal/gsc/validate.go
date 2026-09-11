package gsc

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
)

var googleLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	} // tzdata is embedded, including in the scratch image.
	return loc
}()

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateAnalyticsQuery checks required fields and fills date defaults.
// Dates default to the last 28 complete days ending yesterday (GSC data lag).
func ValidateAnalyticsQuery(query AnalyticsQuery, now time.Time) (AnalyticsQuery, error) {
	query.SiteURL = strings.TrimSpace(query.SiteURL)
	if query.SiteURL == "" {
		return AnalyticsQuery{}, fmt.Errorf("siteUrl is required")
	}
	if query.StartRow < 0 {
		return AnalyticsQuery{}, fmt.Errorf("startRow must be >= 0")
	}
	if query.RowLimit < 0 || query.RowLimit > 25000 {
		return AnalyticsQuery{}, fmt.Errorf("rowLimit must be between 0 and 25000")
	}

	if query.EndDate == "" {
		query.EndDate = now.In(googleLocation).AddDate(0, 0, -1).Format("2006-01-02")
	}
	if err := validateDate("endDate", query.EndDate); err != nil {
		return AnalyticsQuery{}, err
	}
	if query.StartDate == "" {
		end, _ := time.Parse("2006-01-02", query.EndDate)
		query.StartDate = end.AddDate(0, 0, -27).Format("2006-01-02")
	}
	if query.RowLimit == 0 {
		query.RowLimit = 1000
	}
	// Reserve space for a returned page without overflowing the next offset.
	if query.StartRow > 1<<53-query.RowLimit {
		return AnalyticsQuery{}, fmt.Errorf("startRow is too large")
	}
	searchTypes := map[string]string{"": "web", "web": "web", "image": "image", "video": "video", "news": "news", "discover": "discover", "googlenews": "googleNews", "google_news": "googleNews"}
	searchType, ok := searchTypes[strings.ToLower(query.SearchType)]
	if !ok {
		return AnalyticsQuery{}, fmt.Errorf("invalid searchType")
	}
	query.SearchType = searchType
	query.DataState = strings.ToLower(query.DataState)
	if query.DataState == "" {
		query.DataState = "final"
	}
	if query.DataState != "final" && query.DataState != "all" && query.DataState != "hourly_all" {
		return AnalyticsQuery{}, fmt.Errorf("dataState must be final, all or hourly_all")
	}
	aggregations := map[string]bool{"": true, "auto": true, "byPage": true, "byProperty": true, "byNewsShowcasePanel": true}
	if !aggregations[query.AggregationType] {
		return AnalyticsQuery{}, fmt.Errorf("invalid aggregationType")
	}
	dimensions := map[string]bool{"query": true, "page": true, "date": true, "hour": true, "country": true, "device": true, "searchAppearance": true}
	seen := map[string]bool{}
	for _, dimension := range query.Dimensions {
		if !dimensions[dimension] || seen[dimension] {
			return AnalyticsQuery{}, fmt.Errorf("invalid or duplicate dimension: %s", dimension)
		}
		seen[dimension] = true
	}
	if seen["hour"] != (query.DataState == "hourly_all") {
		return AnalyticsQuery{}, fmt.Errorf("hour dimension and hourly_all dataState must be used together")
	}
	if err := validateDate("startDate", query.StartDate); err != nil {
		return AnalyticsQuery{}, err
	}
	if err := validateDate("endDate", query.EndDate); err != nil {
		return AnalyticsQuery{}, err
	}
	if query.StartDate > query.EndDate {
		return AnalyticsQuery{}, fmt.Errorf("startDate must be on or before endDate")
	}

	if len(query.Filters) > 100 {
		return AnalyticsQuery{}, fmt.Errorf("at most 100 filters are allowed")
	}
	// Defaults must not mutate a caller-owned slice.
	query.Filters = append([]DimensionFilter(nil), query.Filters...)
	for i, filter := range query.Filters {
		if !dimensions[filter.Dimension] || filter.Dimension == "date" || filter.Dimension == "hour" {
			return AnalyticsQuery{}, fmt.Errorf("invalid filter dimension")
		}
		if len(filter.Expression) > 4096 {
			return AnalyticsQuery{}, fmt.Errorf("filter expression exceeds 4096 bytes")
		}
		switch filter.Operator {
		case "", "equals", "notEquals", "contains", "notContains", "includingRegex", "excludingRegex":
		default:
			return AnalyticsQuery{}, fmt.Errorf("invalid filter operator")
		}
		if strings.TrimSpace(filter.Dimension) == "" || strings.TrimSpace(filter.Expression) == "" {
			return AnalyticsQuery{}, fmt.Errorf("filters[%d] requires dimension and expression", i)
		}
		if strings.TrimSpace(filter.Operator) == "" {
			query.Filters[i].Operator = "equals"
		}
	}

	if query.AggregationType == "byProperty" && (seen["page"] || query.SearchType == "discover" || query.SearchType == "googleNews") {
		return AnalyticsQuery{}, fmt.Errorf("byProperty is incompatible with page grouping, discover and googleNews")
	}
	if query.AggregationType == "byProperty" {
		for _, filter := range query.Filters {
			if filter.Dimension == "page" {
				return AnalyticsQuery{}, fmt.Errorf("byProperty is incompatible with page filtering")
			}
		}
	}
	return query, nil
}

// ValidateInspectQuery checks that both property and page URLs are present.
func ValidateInspectQuery(query InspectQuery) (InspectQuery, error) {
	query.SiteURL = strings.TrimSpace(query.SiteURL)
	query.InspectionURL = strings.TrimSpace(query.InspectionURL)
	query.LanguageCode = strings.TrimSpace(query.LanguageCode)
	if query.SiteURL == "" {
		return InspectQuery{}, fmt.Errorf("siteUrl is required")
	}
	if query.InspectionURL == "" {
		return InspectQuery{}, fmt.Errorf("inspectionUrl is required")
	}
	parsed, err := url.Parse(query.InspectionURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return InspectQuery{}, fmt.Errorf("inspectionUrl must be an absolute HTTP(S) URL")
	}
	return query, nil
}

// validateDate ensures value is a calendar date in YYYY-MM-DD form.
func validateDate(field, value string) error {
	if !datePattern.MatchString(value) {
		return fmt.Errorf("%s must be YYYY-MM-DD", field)
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%s is not a valid calendar date", field)
	}
	return nil
}
