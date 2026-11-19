ARENA := bin/arena
CLI := bin/connect4
COVERAGE_FILE := coverage.out
COVERAGE_THRESHOLD := 80
GO_PACKAGES := ./...
PYTHON ?= python3

# Tailwind CSS standalone CLI, pinned with per-platform checksums (ADR-012,
# dependency-management.md rules 4 and 9). Checksums are from the release's
# sha256sums.txt.
TAILWIND_VERSION := 4.1.11
TAILWIND := .tools/tailwindcss-$(TAILWIND_VERSION)
TAILWIND_OS := $(shell uname -s | sed 's/Darwin/macos/;s/Linux/linux/')
TAILWIND_ARCH := $(shell uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/')
TAILWIND_ASSET := tailwindcss-$(TAILWIND_OS)-$(TAILWIND_ARCH)
TAILWIND_SHA256_macos-arm64 := f5984b9c005c3e67841c33906c7a7c92e85e405f61e029e9bb62e880dd662e79
TAILWIND_SHA256_macos-x64 := 76e27326506d10d50e65b751795f0537f9304ecb100abe835ec138c41774f38c
TAILWIND_SHA256_linux-x64 := 64805b84af4292e043ea6f86d242f191c0ac75359c1a498455dfe6c642afdbab
TAILWIND_SHA256_linux-arm64 := 0409aa4222969f47fa6f4160fe5387e79bf7269e7afe0e8b22f7532c98e1d314
SHA256 := $(shell command -v sha256sum >/dev/null 2>&1 && echo sha256sum || echo shasum -a 256)
CSS := internal/web/static/app.css

.DEFAULT_GOAL := help

.PHONY: help build css run check health clean install install-tools fmt fmt-check vet lint test coverage vuln docker-build docker-smoke tailwind compose-config tidy

help: ## Show available commands
	@echo "Core commands:"
	@echo "  make build    Build the arena server and connect4 CLI"
	@echo "  make run      Run the arena server"
	@echo "  make check    Run all local quality gates"
	@echo "  make health   Show repository health"
	@echo "  make clean    Remove generated artifacts"
	@echo
	@echo "Utility commands:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z_-]+:.*## / && $$1 !~ /^(help|build|run|check|health|clean)$$/ {printf "  make %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: css ## Core: build the arena server and connect4 CLI (spec: interface/commands.md)
	@CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(ARENA) ./cmd/arena
	@CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(CLI) ./cmd/connect4
	@echo "Project is ready to run."

css: $(CSS) ## Build the browser stylesheet with the pinned Tailwind CLI (ADR-012)

tailwind: $(TAILWIND) ## Download and verify the pinned Tailwind CLI (ADR-012)

$(CSS): $(TAILWIND) internal/web/styles/app.css $(wildcard internal/web/templates/*.html)
	@$(TAILWIND) -i internal/web/styles/app.css -o $(CSS) --minify 2>/dev/null
	@echo "Built $(CSS)."

$(TAILWIND):
	@mkdir -p .tools
	@echo "Downloading Tailwind CSS $(TAILWIND_VERSION) ($(TAILWIND_ASSET))..."
	@curl -fsSL --retry 3 -o $@.tmp https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/$(TAILWIND_ASSET)
	@echo "$(TAILWIND_SHA256_$(TAILWIND_OS)-$(TAILWIND_ARCH))  $@.tmp" | $(SHA256) -c - >/dev/null || { rm -f $@.tmp; echo "Tailwind checksum mismatch for $(TAILWIND_ASSET)" >&2; exit 1; }
	@chmod +x $@.tmp
	@# macOS 27 kills the release's linker-only ad hoc signature; re-sign locally, after the checksum check.
	@if [ "$(TAILWIND_OS)" = macos ]; then codesign --force --sign - $@.tmp 2>/dev/null; fi
	@mv $@.tmp $@

run: build ## Core: run the arena server (spec: interface/commands.md)
	@echo "Project is running."
	@./$(ARENA)

check: fmt-check vet lint test coverage vuln ## Core: run all local quality gates (spec: interface/commands.md)
	@echo "PASS"

health: ## Core: show repository health (spec: interface/commands.md)
	@$(PYTHON) scripts/health.py

clean: ## Core: remove generated artifacts (spec: interface/commands.md)
	@rm -rf bin $(COVERAGE_FILE) coverage.html $(CSS)
	@echo "Project is clean."

install: install-tools ## Install development tools and git hooks
	pre-commit install
	pre-commit install --hook-type commit-msg

# govulncheck type-checks with the Go it was built with, so it is built with
# this module's toolchain: an older Go cannot parse newer standard library code.
install-tools: ## Install pinned Go development tools
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	GOTOOLCHAIN=$$(go env GOVERSION) go install golang.org/x/vuln/cmd/govulncheck@v1.9.0

fmt: ## Format Go files (spec: go.md:2-3)
	gofmt -w .
	goimports -w .

fmt-check: ## Check Go formatting without modifying files
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	@test -z "$$(goimports -l .)" || { goimports -l .; exit 1; }

vet: ## Run go vet (spec: go.md:15)
	go vet $(GO_PACKAGES)

lint: ## Run golangci-lint (spec: go.md:16)
	golangci-lint run $(GO_PACKAGES)

test: ## Run tests with race detection (spec: go.md:12)
	go test -race $(GO_PACKAGES)

coverage: ## Enforce application line coverage (spec: unit-testing.md:15)
	go test -race -coverpkg=./internal/... -coverprofile=$(COVERAGE_FILE) ./tests/...
	@coverage=$$(go tool cover -func=$(COVERAGE_FILE) | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	awk -v coverage="$$coverage" -v threshold="$(COVERAGE_THRESHOLD)" 'BEGIN { \
		printf "Total coverage: %.1f%% (minimum: %.1f%%)\n", coverage, threshold; \
		exit !(coverage >= threshold) \
	}'

vuln: ## Scan dependencies for known vulnerabilities
	govulncheck $(GO_PACKAGES)

docker-build: ## Build the production container with BuildKit
	DOCKER_BUILDKIT=1 docker build --tag connect4arena:local .

docker-smoke: docker-build ## Run the container and check it serves /healthz and the stylesheet
	@sh scripts/docker-smoke.sh connect4arena:local

compose-config: ## Validate Compose configuration
	docker compose config --quiet

tidy: ## Tidy and verify Go modules (spec: go.md:17)
	go mod tidy
	git diff --exit-code -- go.mod go.sum
