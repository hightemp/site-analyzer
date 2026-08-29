.DEFAULT_GOAL := help

.PHONY: all build install test test-coverage vet lint fmt check snapshot \
	release publish clean help release-validate

GO ?= go
BINARY := bin/site-analyzer
PACKAGE := ./cmd/site-analyzer
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
REMOTE ?= origin
RELEASE_BRANCH ?= main
GOLANGCI_LINT_VERSION := v2.12.2
GORELEASER_VERSION := v2.12.7
LDFLAGS := -s -w -X site-analyzer/internal/app.version=$(VERSION)

all: check build

build:
	$(GO) build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" -o $(BINARY) $(PACKAGE)

install:
	$(GO) install -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" $(PACKAGE)

test:
	$(GO) test -race ./...

test-coverage:
	$(GO) test -coverprofile=coverage.out ./internal/...
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

lint:
	$(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

fmt:
	gofmt -w $$(rg --files -g '*.go')

check: test vet lint

snapshot:
	$(GO) run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) release --snapshot --clean

release-validate:
	@test -n "$(VERSION)" || (echo "VERSION is required; example: make release VERSION=v1.2.3" >&2; exit 1)
	@echo "$(VERSION)" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$$' || (echo "VERSION must be a semantic version prefixed with v, for example v1.2.3" >&2; exit 1)
	@test -z "$$(git status --porcelain)" || (echo "working tree must be clean before publishing a release" >&2; exit 1)
	@test "$$(git branch --show-current)" = "$(RELEASE_BRANCH)" || (echo "releases must be published from $(RELEASE_BRANCH)" >&2; exit 1)
	@git fetch "$(REMOTE)" "$(RELEASE_BRANCH)" --tags
	@test "$$(git rev-parse HEAD)" = "$$(git rev-parse "$(REMOTE)/$(RELEASE_BRANCH)")" || (echo "local $(RELEASE_BRANCH) must match $(REMOTE)/$(RELEASE_BRANCH)" >&2; exit 1)
	@! git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null || (echo "tag $(VERSION) already exists" >&2; exit 1)

release: release-validate check
	git tag -a "$(VERSION)" -m "Release $(VERSION)"
	git push "$(REMOTE)" "$(VERSION)"
	@echo "Release workflow triggered for $(VERSION)"

publish: release

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist

help:
	@echo "Available targets:"
	@echo "  build          Build $(BINARY) with version metadata"
	@echo "  install        Install site-analyzer into GOPATH/bin"
	@echo "  test           Run tests with the race detector"
	@echo "  test-coverage  Generate a coverage report"
	@echo "  vet            Run go vet"
	@echo "  lint           Run golangci-lint"
	@echo "  check          Run test, vet, and lint"
	@echo "  snapshot       Build local release artifacts with GoReleaser"
	@echo "  release        Publish a tag: make release VERSION=v1.2.3"
	@echo "  clean          Remove generated build and release artifacts"
