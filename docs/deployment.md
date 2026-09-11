# Deployment

The installation shares one service account and bearer token across a trusted owner or team. Anyone with the token can read every property available to that account. Use separate installations for separate trust boundaries.

## HTTPS

Configure the live installation from the README first. Point a public DNS name to the server and allow inbound TCP 80/443 (UDP 443 is optional). Caddy obtains and renews certificates:

```bash
export LOCAL_UID=$(id -u) LOCAL_GID=$(id -g)
export MCP_HOST=gsc.example.com
docker compose -f docker-compose.yml -f compose.https.yml up -d --build
```

Connect agents to `https://gsc.example.com/mcp` with the bearer header. Keep the token private. The proxy forwards to the container network; port 8080 is bound only to host loopback. Do not expose plaintext HTTP outside the trusted local machine. Keep Caddy's data volume across restarts for certificate renewal.

To verify the external route with a locally built binary:

```bash
make build
MCP_AUTH_TOKEN='your-private-token' bin/mcp-server --smoke https://gsc.example.com/mcp
```

This checks real access when Google credentials are configured. Do not paste real tokens or responses into public issues.

## Versioned image

After a release workflow publishes the image, replace source builds with:

```bash
export MCP_VERSION=1.0.0
docker compose pull
docker compose up -d --no-build
```

For the HTTPS deployment, include both `-f` options on these commands. Pin an explicit version or image digest. To roll back, set `MCP_VERSION` to a previously published version, pull and recreate the service. Configuration and credentials stay outside the image. The server stores no analytics database.

The runtime is a non-root static binary with CA certificates, a read-only filesystem, dropped Linux capabilities, and bounded process/memory limits in Compose. Rotate the bearer by updating `.env` and recreating the service. Rotate Google keys in your cloud account and replace the mounted file.
