.PHONY: test lint

GOBIN := $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(GOBIN)/golangci-lint

test:
	go test ./...

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

$(GOLANGCI_LINT):
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
