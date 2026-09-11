package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tenqz/google-search-console-mcp/internal/config"
	"github.com/tenqz/google-search-console-mcp/internal/gsc"
	"github.com/tenqz/google-search-console-mcp/internal/httpserver"
	"github.com/tenqz/google-search-console-mcp/internal/mcpserver"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDemoSmoke exercises the release check through the real HTTP handler with public fixtures.
func TestDemoSmoke(t *testing.T) {
	server := mcpserver.New(gsc.NewDemo())
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	ts := httptest.NewServer(httpserver.New(config.Config{MCPPath: "/mcp", AuthToken: "test"}, handler))
	defer ts.Close()
	if err := checkHealth(ts.URL + "/health"); err != nil {
		t.Fatal(err)
	}
	if err := smoke(ts.URL+"/mcp", "test"); err != nil {
		t.Fatal(err)
	}
	if err := smoke(ts.URL+"/mcp", "wrong"); err == nil {
		t.Fatal("wrong token accepted")
	}
}
