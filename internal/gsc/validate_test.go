package gsc_test

import (
	"testing"
	"time"

	"github.com/tenqz/google-search-console-mcp/internal/gsc"
)

// TestValidateAnalyticsQueryRequiresSiteURL documents that a property identifier is mandatory.
func TestValidateAnalyticsQueryRequiresSiteURL(t *testing.T) {
	_, err := gsc.ValidateAnalyticsQuery(gsc.AnalyticsQuery{}, time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC))

	if err == nil {
		t.Fatal("expected an error when siteUrl is empty")
	}
}

// TestValidateAnalyticsQueryDefaultsDateRange documents the 28-day window ending yesterday.
func TestValidateAnalyticsQueryDefaultsDateRange(t *testing.T) {
	now := time.Date(2026, 9, 11, 15, 0, 0, 0, time.UTC)

	got, err := gsc.ValidateAnalyticsQuery(gsc.AnalyticsQuery{SiteURL: "sc-domain:example.com"}, now)
	if err != nil {
		t.Fatalf("ValidateAnalyticsQuery() error = %v", err)
	}

	if got.StartDate != "2026-08-14" {
		t.Fatalf("StartDate = %q, want 2026-08-14", got.StartDate)
	}
	if got.EndDate != "2026-09-10" {
		t.Fatalf("EndDate = %q, want 2026-09-10", got.EndDate)
	}
}

// TestValidateAnalyticsQueryRejectsInvertedRange documents startDate must not be after endDate.
func TestValidateAnalyticsQueryRejectsInvertedRange(t *testing.T) {
	_, err := gsc.ValidateAnalyticsQuery(gsc.AnalyticsQuery{
		SiteURL:   "https://example.com/",
		StartDate: "2026-09-10",
		EndDate:   "2026-09-01",
	}, time.Time{})

	if err == nil {
		t.Fatal("expected an error for an inverted date range")
	}
}

// TestValidateAnalyticsQueryRejectsInvalidDate documents YYYY-MM-DD calendar validation.
func TestValidateAnalyticsQueryRejectsInvalidDate(t *testing.T) {
	_, err := gsc.ValidateAnalyticsQuery(gsc.AnalyticsQuery{
		SiteURL:   "https://example.com/",
		StartDate: "2026-13-40",
		EndDate:   "2026-09-01",
	}, time.Time{})

	if err == nil {
		t.Fatal("expected an error for an invalid calendar date")
	}
}

// TestValidateAnalyticsQueryDefaultsFilterOperator documents that omitted operators become equals.
func TestValidateAnalyticsQueryDefaultsFilterOperator(t *testing.T) {
	got, err := gsc.ValidateAnalyticsQuery(gsc.AnalyticsQuery{
		SiteURL:   "https://example.com/",
		StartDate: "2026-09-01",
		EndDate:   "2026-09-10",
		Filters: []gsc.DimensionFilter{{
			Dimension:  "query",
			Expression: "shoes",
		}},
	}, time.Time{})
	if err != nil {
		t.Fatalf("ValidateAnalyticsQuery() error = %v", err)
	}

	if got.Filters[0].Operator != "equals" {
		t.Fatalf("Operator = %q, want equals", got.Filters[0].Operator)
	}
}

// TestValidateInspectQueryRequiresBothURLs documents inspect_url input rules.
func TestValidateInspectQueryRequiresBothURLs(t *testing.T) {
	_, err := gsc.ValidateInspectQuery(gsc.InspectQuery{SiteURL: "https://example.com/"})

	if err == nil {
		t.Fatal("expected an error when inspectionUrl is empty")
	}
}
