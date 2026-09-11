<p align="center">
  <h1 align="center">Google Search Console MCP</h1>
  <p align="center">Go MCP server for Google Search Console over Streamable HTTP</p>
</p>

<p align="center">
  <a href="https://github.com/tenqz/google-search-console-mcp/actions/workflows/tests.yml"><img src="https://img.shields.io/github/actions/workflow/status/tenqz/google-search-console-mcp/tests.yml?branch=main&label=tests" alt="Tests"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
  <a href="https://pkg.go.dev/github.com/tenqz/google-search-console-mcp"><img src="https://pkg.go.dev/badge/github.com/tenqz/google-search-console-mcp.svg" alt="Go Reference"></a>
</p>

## About

Remote [Model Context Protocol](https://modelcontextprotocol.io/) server that exposes Google Search Console to AI agents. Deploy with Docker, give the agent `https://your-host/mcp`, and query sites, search analytics, URL inspection, and sitemaps.

**Version 0.1.0** is an MVP for personal and small-team use.

## Features

- List Search Console properties
- Query Performance data (clicks, impressions, CTR, position)
- Inspect URL index status
- List sitemaps
- Streamable HTTP MCP transport
- Bearer / API-key protection
- Unit tests for config, auth, HTTP, GSC mapping, and tools

## Requirements

- Go 1.24+
- Docker (for the published image)
- Google Cloud service account with Search Console API enabled
- The service account invited to each Search Console property

## Installation

```bash
git clone https://github.com/tenqz/google-search-console-mcp.git
cd google-search-console-mcp
```

## Quick Start

```bash
cp .env.example .env
# set MCP_AUTH_TOKEN
# place credentials/service-account.json
docker compose up -d --build
```

Health: `GET /health`  
MCP: `POST /mcp` with `Authorization: Bearer <MCP_AUTH_TOKEN>`

Agent config:

```json
{
  "mcpServers": {
    "search-console": {
      "url": "https://your-host.example/mcp",
      "headers": {
        "Authorization": "Bearer <MCP_AUTH_TOKEN>"
      }
    }
  }
}
```

Google setup: [docs/google-setup.md](docs/google-setup.md). MCP details: [docs/mcp.md](docs/mcp.md).

## MCP Tools

| Tool | Description |
|------|-------------|
| `list_sites` | Properties and permission levels |
| `query_analytics` | Search performance rows |
| `inspect_url` | Index coverage for a URL |
| `list_sitemaps` | Sitemaps known to Google |

## Environment

| Variable | Meaning |
|----------|-----------|
| `HTTP_ADDR` | Bind address, default `:8080` |
| `MCP_PATH` | MCP path, default `/mcp` |
| `MCP_AUTH_TOKEN` | Shared secret (required unless insecure) |
| `MCP_ALLOW_INSECURE` | Disable auth for local debugging |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service-account JSON |
| `GOOGLE_CREDENTIALS_JSON` | Inline service-account JSON |

## Testing

```bash
go test ./...
```

Guidelines: [docs/testing.md](docs/testing.md).

## Architecture

- `cmd/mcp-server` — process entrypoint
- `internal/config` — environment
- `internal/gsc` — Search Console REST client
- `internal/mcpserver` — MCP tools
- `internal/auth` / `internal/httpserver` — HTTP and bearer auth

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT License. See [LICENSE](LICENSE).

## Contact

**Author:** Oleg Patsay  
**Email:** smmartbiz@gmail.com  
**GitHub:** [tenqz/google-search-console-mcp](https://github.com/tenqz/google-search-console-mcp)
