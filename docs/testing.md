# Testing

Every test function has a comment that states the behaviour it protects.

## How to run

```bash
go test ./...
```

Or with Compose:

```bash
docker compose --profile dev up -d tools
docker compose exec tools go test ./...
```

## Layout

| Package | What is covered |
|---------|-----------------|
| `internal/config` | Required secrets, insecure flag, MCP path |
| `internal/auth` | Bearer and `X-API-Key` |
| `internal/httpserver` | Public `/health`, protected `/mcp` |
| `internal/gsc` | Date defaults, REST mapping |
| `internal/mcpserver` | Tool validation and HTTP catalog |

Google is not called from unit tests.
