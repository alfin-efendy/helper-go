# Makefile for helper-go library
# This file provides common development and CI/CD commands

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Project parameters
BINARY_NAME=helper-go
BINARY_UNIX=$(BINARY_NAME)_unix
PACKAGES=./...
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: all build clean test test-verbose test-race test-cover deps fmt lint vet security security-report vuln-check vuln-report complexity complexity-report misspell ineffassign quality ci help

# Default target
all: test build

## Build the application
build:
	@echo "$(BLUE)Building...$(NC)"
	$(GOBUILD) -v $(PACKAGES)

## Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

## Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	$(GOTEST) -v $(PACKAGES)

## Run tests with verbose output
test-verbose:
	@echo "$(BLUE)Running tests with verbose output...$(NC)"
	$(GOTEST) -v -race $(PACKAGES)

## Run tests with race detection
test-race:
	@echo "$(BLUE)Running tests with race detection...$(NC)"
	$(GOTEST) -race $(PACKAGES)

## Run tests with coverage
test-cover:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GOTEST) -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(PACKAGES)
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(GREEN)Coverage report generated: $(COVERAGE_HTML)$(NC)"

## Run benchmark tests
benchmark:
	@echo "$(BLUE)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem $(PACKAGES)

## Download and verify dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) verify
	$(GOMOD) tidy

## Update dependencies
deps-update:
	@echo "$(BLUE)Updating dependencies...$(NC)"
	$(GOGET) -u $(PACKAGES)
	$(GOMOD) tidy

## Format code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GOFMT) -s -w .

## Run linter
lint:
	@echo "$(BLUE)Running linter...$(NC)"
	$(GOLINT) run --timeout=10m --max-issues-per-linter=0 --max-same-issues=0 --tests=false

## Run linter with fixes
lint-fix:
	@echo "$(BLUE)Running linter with auto-fixes...$(NC)"
	$(GOLINT) run --timeout=10m --max-issues-per-linter=0 --max-same-issues=0 --tests=false --fix

## Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	$(GOCMD) vet $(PACKAGES)

## Run security checks
security:
	@echo "$(BLUE)Running security checks...$(NC)"
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec $(PACKAGES)

## Run vulnerability check
vuln-check:
	@echo "$(BLUE)Running vulnerability check...$(NC)"
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck $(PACKAGES)

## Run security checks (non-failing for CI)
security-report:
	@echo "$(BLUE)Running security report...$(NC)"
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec $(PACKAGES) || echo "$(YELLOW)Security issues found (see above)$(NC)"

## Run vulnerability check (non-failing for CI)
vuln-report:
	@echo "$(BLUE)Running vulnerability report...$(NC)"
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck $(PACKAGES) || echo "$(YELLOW)Vulnerabilities found (see above)$(NC)"

## Install development tools
install-tools:
	@echo "$(BLUE)Installing development tools...$(NC)"
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install github.com/client9/misspell/cmd/misspell@latest
	go install github.com/gordonklaus/ineffassign@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin

## Check code complexity
complexity:
	@echo "$(BLUE)Checking code complexity...$(NC)"
	@if ! command -v gocyclo >/dev/null 2>&1; then \
		echo "Installing gocyclo..."; \
		go install github.com/fzipp/gocyclo/cmd/gocyclo@latest; \
	fi
	gocyclo -over 10 .

## Check code complexity (non-failing for CI)
complexity-report:
	@echo "$(BLUE)Checking code complexity...$(NC)"
	@if ! command -v gocyclo >/dev/null 2>&1; then \
		echo "Installing gocyclo..."; \
		go install github.com/fzipp/gocyclo/cmd/gocyclo@latest; \
	fi
	@gocyclo -over 10 . || echo "$(YELLOW)High complexity functions found (see above)$(NC)"

## Check for misspellings
misspell:
	@echo "$(BLUE)Checking for misspellings...$(NC)"
	@if ! command -v misspell >/dev/null 2>&1; then \
		echo "Installing misspell..."; \
		go install github.com/client9/misspell/cmd/misspell@latest; \
	fi
	find . -name '*.go' | xargs misspell -error

## Check for inefficient assignments
ineffassign:
	@echo "$(BLUE)Checking for inefficient assignments...$(NC)"
	@if ! command -v ineffassign >/dev/null 2>&1; then \
		echo "Installing ineffassign..."; \
		go install github.com/gordonklaus/ineffassign@latest; \
	fi
	ineffassign $(PACKAGES)

## Run all quality checks
quality: fmt vet lint test-race complexity-report misspell ineffassign security-report vuln-report
	@echo "$(GREEN)All quality checks completed!$(NC)"

## Prepare for CI/CD (run all checks)
ci: deps quality test-cover
	@echo "$(GREEN)CI/CD preparation completed!$(NC)"

## Generate documentation
docs:
	@echo "$(BLUE)Generating documentation...$(NC)"
	$(GOCMD) doc -all . > docs.txt
	@echo "$(GREEN)Documentation generated: docs.txt$(NC)"

## Create a new release (example: make release VERSION=v1.0.0)
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Please provide VERSION (e.g., make release VERSION=v1.0.0)$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Creating release $(VERSION)...$(NC)"
	@echo "Running quality checks first..."
	@$(MAKE) ci
	@echo "$(GREEN)Quality checks passed!$(NC)"
	@echo "$(BLUE)Creating git tag...$(NC)"
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "$(GREEN)Release $(VERSION) created and pushed!$(NC)"

## Show git status and recent commits
status:
	@echo "$(BLUE)Git Status:$(NC)"
	git status --porcelain
	@echo ""
	@echo "$(BLUE)Recent Commits:$(NC)"
	git log --oneline -5

## Show current version
version:
	@git describe --tags --always --dirty 2>/dev/null || echo "No version tags found"

## Show help
help:
	@echo "$(BLUE)Available commands:$(NC)"
	@echo ""
	@grep -E '^## .*' $(MAKEFILE_LIST) | sed 's/## //' | while read line; do \
		target=$$(echo "$$line" | cut -d':' -f1); \
		description=$$(echo "$$line" | cut -d':' -f2-); \
		printf "  $(GREEN)%-20s$(NC) %s\n" "$$target" "$$description"; \
	done
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make test              # Run tests"
	@echo "  make test-cover        # Run tests with coverage"
	@echo "  make quality           # Run all quality checks"
	@echo "  make ci                # Prepare for CI/CD"
	@echo "  make release VERSION=v1.0.0  # Create release"

# Cross compilation
build-linux:
	@echo "$(BLUE)Building for Linux...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v

build-windows:
	@echo "$(BLUE)Building for Windows...$(NC)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME).exe -v

build-darwin:
	@echo "$(BLUE)Building for macOS...$(NC)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)_darwin -v

build-all: build-linux build-windows build-darwin
	@echo "$(GREEN)Cross-compilation completed!$(NC)"
