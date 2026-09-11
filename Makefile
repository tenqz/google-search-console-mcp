.PHONY: check fmt tidy vet build test lint

GOFLAGS ?= -mod=readonly
export GOFLAGS
LINT_VERSION := v2.13.2
LINT_BIN := $(CURDIR)/.tools/$(LINT_VERSION)/golangci-lint

check: fmt tidy vet build test lint

fmt:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)

tidy:
	go mod tidy -diff

vet:
	go vet ./...

build:
	go build -trimpath -o bin/mcp-server ./cmd/mcp-server

test:
	go test -race -cover ./...

lint: $(LINT_BIN)
	$(LINT_BIN) run

$(LINT_BIN):
	GOBIN=$(CURDIR)/.tools/$(LINT_VERSION) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION)
