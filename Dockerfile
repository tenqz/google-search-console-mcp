FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine AS build
WORKDIR /src
ENV GOFLAGS=-mod=readonly GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/mcp-server ./cmd/mcp-server

FROM scratch AS runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/mcp-server /mcp-server
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 CMD ["/mcp-server", "--healthcheck", "http://127.0.0.1:8080/health"]
ENTRYPOINT ["/mcp-server"]
