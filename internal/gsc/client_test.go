package gsc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMapAnalyticsResultEmptyOnNilRows documents an empty payload when Google returns no rows.
func TestMapAnalyticsResultEmptyOnNilRows(t *testing.T) {
	got := mapAnalyticsResult(analyticsResponse{})

	if got.Rows == nil {
		t.Fatal("Rows must be a non-nil empty slice")
	}
}

// TestMapAnalyticsResultCopiesMetrics documents Search Analytics row mapping.
func TestMapAnalyticsResultCopiesMetrics(t *testing.T) {
	got := mapAnalyticsResult(analyticsResponse{
		ResponseAggregationType: "auto",
		Rows: []analyticsRowJSON{{
			Keys: []string{"shoes"}, Clicks: 10, Impressions: 100, CTR: 0.1, Position: 4.2,
		}},
	})

	if got.RowCount != 1 {
		t.Fatalf("RowCount = %d, want 1", got.RowCount)
	}
}

// TestMapInspectResultCopiesIndexStatus documents the fields agents need from URL Inspection.
func TestMapInspectResultCopiesIndexStatus(t *testing.T) {
	got := mapInspectResult(inspectResponse{
		InspectionResult: &inspectResultJSON{
			InspectionResultLink: "https://search.google.com/search-console/inspect",
			IndexStatusResult: &struct {
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
			}{
				CoverageState: "Submitted and indexed",
				Verdict:       "PASS",
			},
		},
	})

	if got.CoverageState != "Submitted and indexed" {
		t.Fatalf("CoverageState = %q", got.CoverageState)
	}
}

// TestClientListSitesHitsWebmasterEndpoint documents the REST path used for sites.list.
func TestClientListSitesHitsWebmasterEndpoint(t *testing.T) {
	var path string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"siteEntry": []map[string]string{{
				"siteUrl": "sc-domain:example.com", "permissionLevel": "siteFullUser",
			}},
		})
	}))
	t.Cleanup(ts.Close)

	client := &Client{http: ts.Client(), webmasterBase: ts.URL, inspectURL: ts.URL}
	sites, err := client.ListSites(context.Background())
	if err != nil {
		t.Fatalf("ListSites() error = %v", err)
	}
	if path != "/sites" {
		t.Fatalf("path = %q, want /sites", path)
	}
	if len(sites) != 1 {
		t.Fatalf("len(sites) = %d", len(sites))
	}
}

// TestNewClientFromJSONRequiresServiceAccountFields documents credential JSON validation.
func TestNewClientFromJSONRequiresServiceAccountFields(t *testing.T) {
	_, err := NewClientFromJSON(context.Background(), []byte(`{"type":"service_account"}`))

	if err == nil {
		t.Fatal("expected an error for incomplete service account JSON")
	}
}
