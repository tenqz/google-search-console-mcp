package mcpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tenqz/search-console/internal/gsc"
	"github.com/tenqz/search-console/internal/mcpserver"
)

// TestStreamableHTTPExposesCatalog documents that a remote agent can list the MVP tools over HTTP.
func TestStreamableHTTPExposesCatalog(t *testing.T) {
	server := mcpserver.New(&fakeConsole{sites: []gsc.Site{{URL: "https://example.com/"}}})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true})
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	want := map[string]bool{
		"list_sites":      false,
		"query_analytics": false,
		"inspect_url":     false,
		"list_sitemaps":   false,
	}
	for _, tool := range listed.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("missing tool %s", name)
		}
	}
}
