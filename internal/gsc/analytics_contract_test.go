package gsc

import (
	"testing"
	"time"
)

// TestPacificCalendarDefaults checks midnight boundaries and both DST transitions.
func TestPacificCalendarDefaults(t *testing.T) {
	for _, tc := range []struct{ now, end string }{{"2026-09-12T02:00:00Z", "2026-09-10"}, {"2026-03-09T06:00:00Z", "2026-03-07"}, {"2026-11-02T07:00:00Z", "2026-10-31"}} {
		now, _ := time.Parse(time.RFC3339, tc.now)
		got, err := ValidateAnalyticsQuery(AnalyticsQuery{SiteURL: "sc-domain:example.com"}, now)
		if err != nil || got.EndDate != tc.end {
			t.Fatalf("%s: %v %v", tc.now, got, err)
		}
	}
}

// TestAnalyticsValidationRejectsInvalidCombinations catches agent mistakes before consuming quota.
func TestAnalyticsValidationRejectsInvalidCombinations(t *testing.T) {
	for _, query := range []AnalyticsQuery{
		{Dimensions: []string{"page", "page"}}, {DataState: "unknown"}, {Dimensions: []string{"hour"}},
		{SearchType: "unknown"}, {Filters: []DimensionFilter{{Dimension: "query", Operator: "bad", Expression: "x"}}},
		{AggregationType: "byProperty", Dimensions: []string{"page"}},
	} {
		query.SiteURL = "sc-domain:example.com"
		if _, err := ValidateAnalyticsQuery(query, time.Now()); err == nil {
			t.Fatalf("accepted %+v", query)
		}
	}
}
