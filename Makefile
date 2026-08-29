.PHONY: build test test-coverage vet lint fmt clean

BINARY := bin/site-analyzer
GOLANGCI_LINT_VERSION := v2.12.2

build:
	go build -trimpath -buildvcs=false -o $(BINARY) ./cmd/site-analyzer

test:
	go test -race ./...

test-coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out

vet:
	go vet ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

fmt:
	gofmt -w $$(rg --files -g '*.go')

clean:
	rm -f $(BINARY)
