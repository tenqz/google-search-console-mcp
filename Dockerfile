FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mcp-server ./cmd/mcp-server

FROM build AS test
CMD ["go", "test", "./..."]

FROM alpine:3.21 AS runtime
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/mcp-server /usr/local/bin/mcp-server
USER nobody
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/usr/local/bin/mcp-server"]
