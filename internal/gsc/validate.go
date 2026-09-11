package gsc

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

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

	if query.StartDate == "" && query.EndDate == "" {
		end := now.UTC().AddDate(0, 0, -1)
		query.EndDate = end.Format("2006-01-02")
		query.StartDate = end.AddDate(0, 0, -27).Format("2006-01-02")
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

	for i, filter := range query.Filters {
		if strings.TrimSpace(filter.Dimension) == "" || strings.TrimSpace(filter.Expression) == "" {
			return AnalyticsQuery{}, fmt.Errorf("filters[%d] requires dimension and expression", i)
		}
		if strings.TrimSpace(filter.Operator) == "" {
			query.Filters[i].Operator = "equals"
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
