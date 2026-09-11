# MCP endpoint

The process serves [Streamable HTTP](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports) on `MCP_PATH` (default `/mcp`) in **stateless** mode so a reverse proxy and a remote agent can call it without sticky sessions.

## Agent URL

- Local Compose: `http://<server-ip>:8080/mcp`
- Behind TLS: `https://gsc-mcp.example.com/mcp`

Health is a separate route: `GET /health` → `{"status":"ok"}`.

## Auth

Send one of:

```
Authorization: Bearer <MCP_AUTH_TOKEN>
X-API-Key: <MCP_AUTH_TOKEN>
```

Without a valid secret the handler returns `401`.

## Tools

Agents should typically:

1. `list_sites` to learn `siteUrl` values
2. `query_analytics` with `siteUrl`, optional `startDate`/`endDate`, optional `dimensions` (`query`, `page`, `date`, `country`, `device`)
3. `inspect_url` when a specific page’s index status is needed
4. `list_sitemaps` to see submitted sitemaps

If dates are omitted, `query_analytics` uses the last 28 days ending yesterday.
