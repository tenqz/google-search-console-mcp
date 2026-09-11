package gsc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// TestAnalyticsContract preserves incomplete data metadata and exposes pagination hints.
func TestAnalyticsContract(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["type"] != "web" || body["aggregationType"] != "auto" {
			t.Errorf("body=%v", body)
		}
		_, _ = w.Write([]byte(`{"rows":[{"keys":["2026-09-10"],"clicks":5,"impressions":50,"ctr":0.1,"position":3}],"metadata":{"firstIncompleteDate":"2026-09-10"}}`))
	}))
	defer ts.Close()
	client := NewClient(ts.Client(), Options{})
	client.webmasterBase = ts.URL
	query, err := ValidateAnalyticsQuery(AnalyticsQuery{SiteURL: "sc-domain:example.com", EndDate: "2026-09-10", RowLimit: 1, StartRow: 10, DataState: "ALL", AggregationType: "auto", Dimensions: []string{"date"}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.QueryAnalytics(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata == nil || got.Metadata.FirstIncompleteDate != "2026-09-10" || got.StartDate != "2026-08-14" || got.DataState != "all" || !got.MayHaveMore || got.NextStartRow == nil || *got.NextStartRow != 11 || got.Rows[0].Clicks != 5 || got.Rows[0].CTR != 0.1 {
		t.Fatalf("result=%+v", got)
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
