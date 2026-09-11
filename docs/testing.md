# Testing

Run `make check` with Go 1.25+ and a current security patch. It verifies formatting, `go mod tidy -diff`, vet, build, race tests and the same pinned linter used by CI. The linter is installed under `.tools/`; the global installation is untouched. An ignored vendor directory is not used.

`make docker-test` builds the runtime and calls all four tools against deterministic demo data. `docker compose -f compose.demo.yml up -d --build` reproduces the documented onboarding path. CI runs Go 1.25 and 1.26 and smoke-tests Linux amd64 and arm64 images. The security workflow runs govulncheck on changes and weekly.

The HTTP integration tests use the real MCP SDK, bearer middleware, Google client and OAuth token handling with a fake upstream transport. They cover successful calls, validation, upstream failures and token reuse without real credentials. Regression tests cover cancellation, retries, limits, Pacific date boundaries and freshness metadata.

For live acceptance, configure your own Google account and HTTPS endpoint, then run `MCP_AUTH_TOKEN=... bin/mcp-server --smoke https://your-host/mcp`. Demo tests do not prove Google permissions, DNS/TLS configuration or a particular desktop client's behavior. Do not supply real credentials to PR workflows.

ACDD requires reviewing and checking each commit, not just the branch tip. Include the regression test with its fix and keep dependent configuration changes together.
