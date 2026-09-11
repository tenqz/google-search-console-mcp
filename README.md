# Google Search Console MCP

![Search analytics connected to an AI agent through MCP](docs/assets/hero.png)

**Direct Search Console access for AI agents.**

[![Tests](https://github.com/tenqz/google-search-console-mcp/actions/workflows/tests.yml/badge.svg)](https://github.com/tenqz/google-search-console-mcp/actions/workflows/tests.yml)
[![Code quality](https://github.com/tenqz/google-search-console-mcp/actions/workflows/code-quality.yml/badge.svg)](https://github.com/tenqz/google-search-console-mcp/actions/workflows/code-quality.yml)
[![MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

A small, self-hosted Go server that lets MCP agents read Google Search Console properties, search performance, indexing status and sitemaps. Four read-only tools, Streamable HTTP, one service account per installation. Independent community project; not affiliated with Google.

Ask your agent to compare completed periods, find pages with high impressions and low CTR, or explain Google's indexing status for a URL. The server retrieves the data; your agent interprets it.

## Try it without credentials

Requires Docker with Compose. This builds the current checkout and uses clearly labelled synthetic data, with no Google requests:

```bash
git clone https://github.com/tenqz/google-search-console-mcp.git
cd google-search-console-mcp
docker compose -f compose.demo.yml up -d --build
docker compose -f compose.demo.yml exec mcp /mcp-server --smoke http://127.0.0.1:8080/mcp
```

Connect your MCP client to `http://localhost:8080/mcp` with header `Authorization: Bearer demo-token`. The smoke command checks discovery and calls all four tools. Stop with `docker compose -f compose.demo.yml down` before starting a live installation.

## Connect Google Search Console

1. Follow [Google setup](docs/google-setup.md): enable the API, create a service account and grant its email access to your Search Console property.
2. Save its key as `credentials/service-account.json` and run `chmod 600 credentials/service-account.json`.
3. Copy `.env.example` to `.env`. Replace `MCP_AUTH_TOKEN` with the output of `openssl rand -hex 32`.
4. Start the server:

```bash
export LOCAL_UID=$(id -u) LOCAL_GID=$(id -g)
docker compose up -d --build
curl --fail http://localhost:8080/health
docker compose exec mcp /mcp-server --smoke http://127.0.0.1:8080/mcp
```

The UID/GID settings let the container read your private key without making it world-readable. The smoke check uses the first accessible property; URL inspection uses its root URL. Google access and quota errors are reported by the tools.

For access from another machine, use [HTTPS deployment](docs/deployment.md). Connect agents using the templates in [MCP clients and tools](docs/mcp.md). A health response confirms the process is running; the smoke check also verifies MCP and Google access.

## Tools

| Tool | Input | Result |
| --- | --- | --- |
| `list_sites` | `{}` | Accessible properties and permission levels |
| `query_analytics` | `siteUrl`, optional dates, dimensions, filters and paging | Clicks, impressions, CTR, position, effective dates and freshness metadata |
| `inspect_url` | `siteUrl`, `inspectionUrl`, optional `languageCode` | Indexing verdict, crawl state, canonicals and sitemap references |
| `list_sitemaps` | `siteUrl` | Sitemap URLs, processing status, errors and warnings |

Each tool publishes an output schema and read-only annotations. Results include both JSON text and structured content. Tool failures include a stable error code, a safe message and retry information when available.

```json
{
  "siteUrl": "sc-domain:example.com",
  "startDate": "2026-08-01",
  "endDate": "2026-08-28",
  "dimensions": ["page", "query"],
  "rowLimit": 100
}
```

Analytics dates use Pacific Time, including daylight saving time. The default is 28 days ending yesterday, with `dataState: final`. Recent final data may be missing. `all` and `hourly_all` can expose incomplete data; freshness metadata is preserved. Google returns top rows and may omit anonymized queries. `mayHaveMore` and `nextStartRow` are paging hints, **not a guarantee of a complete export**. Inspection reports Google's stored index information, not a live crawl.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Listening address; Compose exposes it on host loopback |
| `MCP_PATH` | `/mcp` | MCP endpoint |
| `MCP_AUTH_TOKEN` | Required | Shared bearer secret |
| `GOOGLE_APPLICATION_CREDENTIALS` | Required unless JSON/demo | Service-account key file |
| `GOOGLE_CREDENTIALS_JSON` | Empty | Alternative inline service-account JSON |
| `GOOGLE_REQUEST_TIMEOUT` | `30s` | Total Google request budget, including queue, OAuth and retries |
| `GOOGLE_MAX_CONCURRENT` | `8` | Concurrent Google work per process |
| `GOOGLE_MAX_ATTEMPTS` | `3` | Total attempts for transient failures |
| `MCP_MAX_BODY_BYTES` | `1048576` | Maximum incoming MCP request size |
| `MCP_DEMO` | `false` | Synthetic data; cannot be combined with credentials |
| `MCP_ALLOW_INSECURE` | `false` | Token-free debugging; requires a loopback listening address |

Google responses are capped at 8 MiB. Reduce `rowLimit` if a response exceeds that bound. Logs contain operation outcomes and durations, without request arguments, key material or raw Google error bodies.

## Troubleshooting

- **401:** check the bearer header and the server's token.
- **Google 403:** enable the API and grant the service-account email access to the exact property. Domain properties use `sc-domain:example.com`.
- **429:** respect retry information and reduce concurrency. Retries share the request time budget.
- **413:** reduce the MCP request size; raise the configured bound only when necessary.
- **Timeout:** reduce query scope or concurrency before raising `GOOGLE_REQUEST_TIMEOUT`.
- **Empty analytics:** verify the property, dates, filters and data freshness. Empty rows can be a valid response.

## Development and releases

Use Go 1.25+ with a current security patch; release containers use Go 1.26.6. Run `make check` for formatting, module consistency, vet, build, race tests and pinned lint. Run `make docker-test` for a container smoke check. See [testing](docs/testing.md), [architecture](docs/architecture.md), [contributing](CONTRIBUTING.md) and [security](SECURITY.md).

The release workflow verifies a `v1.0.0` tag against the binary version, checks both container architectures and publishes versioned images to `ghcr.io/tenqz/google-search-console-mcp`. Images become available after the tag workflow succeeds. See [deployment](docs/deployment.md) for updates and rollback. Licensed under [MIT](LICENSE).
