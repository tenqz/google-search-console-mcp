#!/bin/sh
set -eu
image=${1:?usage: docker-smoke.sh IMAGE [PLATFORM]}
platform=${2:-linux/amd64}
name="gsc-mcp-smoke-$$"
artifact=$(mktemp)
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -f "$artifact"' EXIT INT TERM
docker run --platform "$platform" --detach --name "$name" \
  --read-only --cap-drop ALL --security-opt no-new-privileges \
  -e MCP_DEMO=true -e MCP_AUTH_TOKEN=ci-demo-token "$image" >/dev/null
# Check the ELF machine field: image metadata alone can hide a wrong binary.
docker cp "$name":/mcp-server "$artifact"
machine=$(od -An -tu2 -j18 -N2 "$artifact" | tr -d ' ')
case "$platform:$machine" in
  linux/amd64:62|linux/arm64:183) ;;
  *) echo "Unexpected executable architecture: $platform / $machine" >&2; exit 1 ;;
esac
attempt=0
until docker exec "$name" /mcp-server --healthcheck http://127.0.0.1:8080/health; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 20 ]; then docker logs "$name"; exit 1; fi
  sleep 1
done
docker exec "$name" /mcp-server --smoke http://127.0.0.1:8080/mcp
