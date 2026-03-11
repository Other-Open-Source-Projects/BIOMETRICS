# =============================================================================
# BIOMETRICS CLI - MAKEFILE
# =============================================================================
# Enterprise-Grade Build System for Go CLI Application
# Compliant with MANDATE 0.19: Modern CLI Toolchain (2026 Standard)
# =============================================================================

# -----------------------------------------------------------------------------
# VERSION INFORMATION
# -----------------------------------------------------------------------------
VERSION := 2.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo "untagged")

# Go build flags
LDFLAGS := -ldflags "-s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildTime=$(BUILD_TIME)' \
	-X 'main.GitCommit=$(GIT_COMMIT)' \
	-X 'main.GitBranch=$(GIT_BRANCH)' \
	-X 'main.GitTag=$(GIT_TAG)'"

# -----------------------------------------------------------------------------
# GO CONFIGURATION
# -----------------------------------------------------------------------------
GO := go
GO_VERSION := 1.24.0
GO_MODULES := ./...
CMD_PATH := ./cmd/controlplane
PKG_PATH := ./pkg/...

# Binary configuration
BINARY_NAME := biometrics
BINARY_DIR := bin
DIST_DIR := dist
OUTPUT := $(BINARY_DIR)/$(BINARY_NAME)

# -----------------------------------------------------------------------------
# TOOLS & DEPENDENCIES
# -----------------------------------------------------------------------------
RIPGREP := rg
FD := fd
GOLANGCI_LINT := golangci-lint
GOTESTSUM := gotestsum
GOIMPORTS := goimports
GOFUMPT := gofumpt

# Required Go tools
TOOLS := $(GOLANGCI_LINT) $(GOTESTSUM) $(GOIMPORTS) $(GOFUMPT)

# -----------------------------------------------------------------------------
# DIRECTORIES
# -----------------------------------------------------------------------------
SRC_DIRS := ./cmd ./pkg ./global ./local
TEST_DIRS := ./cmd/controlplane ./cmd/biometrics ./internal/api/http ./internal/blueprints ./internal/controlplane ./internal/policy ./internal/runtime/... ./internal/store/sqlite ./internal/planning
BENCHMARK_DIR := ./benchmarks
COVERAGE_DIR := coverage
REPORTS_DIR := reports

# Create directories if they don't exist
$(shell mkdir -p $(BINARY_DIR) $(DIST_DIR) $(COVERAGE_DIR) $(REPORTS_DIR))

# -----------------------------------------------------------------------------
# PHONY TARGETS
# -----------------------------------------------------------------------------
.PHONY: all \
	build \
	test \
	lint \
	format \
	clean \
	install \
	uninstall \
	docker \
	docker-build \
	docker-run \
	coverage \
	benchmark \
	vet \
	mod \
	tidy \
	deps \
	check \
	help \
	release \
	cross-compile \
	generate \
	mocks \
	security \
	audit

# =============================================================================
# PRIMARY TARGETS
# =============================================================================

# -----------------------------------------------------------------------------
# all: Default target - build, test, lint
# -----------------------------------------------------------------------------
all: build test lint
	@echo "All checks passed successfully!"
	@echo "Binary: $(OUTPUT)"
	@echo "Version: $(VERSION)"
	@echo "Git: $(GIT_BRANCH)/$(GIT_COMMIT)"

# -----------------------------------------------------------------------------
# build: Build the binary
# -----------------------------------------------------------------------------
build: mod
	@echo "Building $(BINARY_NAME)..."
	@echo "   Version: $(VERSION)"
	@echo "   Git: $(GIT_COMMIT)"
	@echo "   Time: $(BUILD_TIME)"
	$(GO) build $(LDFLAGS) -o $(OUTPUT) $(CMD_PATH)
	@echo "Build complete: $(OUTPUT)"
	@ls -lh $(OUTPUT)

# -----------------------------------------------------------------------------
# test: Run all tests with verbose output and race detection
# -----------------------------------------------------------------------------
test:
	@echo "Running tests..."
	@echo "   Packages: $(TEST_DIRS)"
	@echo "   Race detection: enabled"
	@echo "   Coverage: enabled"
	$(GO) test $(TEST_DIRS) \
		-v \
		-race \
		-coverprofile=$(COVERAGE_DIR)/coverage.out \
		-covermode=atomic \
		-timeout=5m \
		-parallel=4
	@echo "Tests passed"

# -----------------------------------------------------------------------------
# lint: Run golangci-lint with comprehensive checks
# -----------------------------------------------------------------------------
lint:
	@echo "Running linter..."
	@if ! command -v $(GOLANGCI_LINT) &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	$(GOLANGCI_LINT) run ./... \
		--enable=gofmt \
		--enable=govet \
		--enable=errcheck \
		--enable=staticcheck \
		--enable=unused \
		--enable=ineffassign \
		--enable=gosimple \
		--enable=structcheck \
		--enable=varcheck \
		--enable=misspell \
		--enable=unconvert \
		--deadline=5m \
		--max-same-issues=50
	@echo "Linting passed"

# -----------------------------------------------------------------------------
# coverage: Generate HTML coverage report
# -----------------------------------------------------------------------------
coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.txt
	@echo "Coverage report generated:"
	@echo "   HTML: $(COVERAGE_DIR)/coverage.html"
	@echo "   Text: $(COVERAGE_DIR)/coverage.txt"
	@echo ""
	@echo "Coverage Summary:"
	@grep "^total:" $(COVERAGE_DIR)/coverage.txt
	@echo ""
	@echo "To view: open $(COVERAGE_DIR)/coverage.html"

# -----------------------------------------------------------------------------
# clean: Remove build artifacts
# -----------------------------------------------------------------------------
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(DIST_DIR)/
	rm -rf $(BINARY_DIR)/
	rm -rf $(COVERAGE_DIR)/
	rm -rf $(REPORTS_DIR)/
	rm -f coverage.out
	rm -f coverage.html
	rm -f coverage.txt
	rm -f test-report.json
	find . -name "*.test" -type f -delete
	find . -name "*.out" -type f -delete
	find . -name "*.prof" -type f -delete
	@echo "Clean complete"

# =============================================================================
# INSTALLATION TARGETS
# =============================================================================

# -----------------------------------------------------------------------------
# install: Install binary to GOPATH/bin
# -----------------------------------------------------------------------------
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(OUTPUT) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed to: $(GOPATH)/bin/$(BINARY_NAME)"
	@echo "Add $(GOPATH)/bin to your PATH if not already done"

# -----------------------------------------------------------------------------
# uninstall: Remove binary from GOPATH/bin
# -----------------------------------------------------------------------------
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstalled from: $(GOPATH)/bin/$(BINARY_NAME)"

# =============================================================================
# DOCKER TARGETS
# =============================================================================

# -----------------------------------------------------------------------------
# docker: Build Docker image
# -----------------------------------------------------------------------------
docker: docker-build
	@echo "Docker image built successfully"

# -----------------------------------------------------------------------------
# docker-build: Build Docker image with proper tagging
# -----------------------------------------------------------------------------
docker-build:
	@echo "Building Docker image..."
	docker build \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		-t biometrics-cli:$(VERSION) \
		-t biometrics-cli:latest \
		-f Dockerfile \
		.
	@echo "Docker images built:"
	@docker images | grep $(BINARY_NAME)
	@echo ""
	@echo "Run with: docker run --rm $(BINARY_NAME):$(VERSION) --help"

# -----------------------------------------------------------------------------
# docker-run: Run container interactively
# -----------------------------------------------------------------------------
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it $(BINARY_NAME):latest $(CMD)

# =============================================================================
# CODE QUALITY TARGETS
# =============================================================================

# -----------------------------------------------------------------------------
# format: Format all Go files with gofumpt
# -----------------------------------------------------------------------------
format:
	@echo "Formatting code..."
	@if ! command -v $(GOFUMPT) &> /dev/null; then \
		echo "Installing gofumpt..."; \
		go install mvdan.cc/gofumpt@latest; \
	fi
	$(GOFUMPT) -l -w $(SRC_DIRS)
	@echo "Formatting complete"

# -----------------------------------------------------------------------------
# vet: Run go vet for suspicious constructs
# -----------------------------------------------------------------------------
vet:
	@echo "Running go vet..."
	$(GO) vet $(GO_MODULES)
	@echo "Vet passed"

# -----------------------------------------------------------------------------
# mod: Tidy Go modules
# -----------------------------------------------------------------------------
mod: tidy

# -----------------------------------------------------------------------------
# tidy: Clean up go.mod and go.sum
# -----------------------------------------------------------------------------
tidy:
	@echo "Tidying Go modules..."
	$(GO) mod tidy
	$(GO) mod verify
	@echo "Modules tidied"

# -----------------------------------------------------------------------------
# deps: List and verify dependencies
# -----------------------------------------------------------------------------
deps:
	@echo "Checking dependencies..."
	$(GO) list -m all
	$(GO) mod verify
	@echo "Dependencies verified"

# =============================================================================
# ADVANCED TARGETS
# =============================================================================

# -----------------------------------------------------------------------------
# benchmark: Run benchmarks
# -----------------------------------------------------------------------------
benchmark:
	@echo "Running benchmarks..."
	$(GO) test $(BENCHMARK_DIR)/... \
		-bench=. \
		-benchmem \
		-benchtime=1s \
		-run=^$$ \
		-timeout=10m \
		| tee $(REPORTS_DIR)/benchmark.txt
	@echo "Benchmarks complete: $(REPORTS_DIR)/benchmark.txt"

# -----------------------------------------------------------------------------
# check: Comprehensive health check
# -----------------------------------------------------------------------------
check: mod vet lint test coverage
	@echo "All health checks passed!"

# -----------------------------------------------------------------------------
# release: Create a release build
# -----------------------------------------------------------------------------
release: clean check cross-compile
	@echo "Release build complete!"
	@echo "Binaries in: $(DIST_DIR)/"
	ls -lh $(DIST_DIR)/

# -----------------------------------------------------------------------------
# cross-compile: Build for multiple platforms
# -----------------------------------------------------------------------------
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@echo "   Platforms: linux, darwin, windows"
	@echo "   Architectures: amd64, arm64"
	
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	
	@echo "Cross-compilation complete"
	@echo "Binaries:"
	ls -lh $(DIST_DIR)/

# -----------------------------------------------------------------------------
# generate: Run code generation
# -----------------------------------------------------------------------------
generate:
	@echo "Running code generation..."
	$(GO) generate $(GO_MODULES)
	@echo "Generation complete"

# -----------------------------------------------------------------------------
# mocks: Generate test mocks
# -----------------------------------------------------------------------------
mocks:
	@echo "Generating test mocks..."
	@if ! command -v mockgen &> /dev/null; then \
		echo "Installing mockgen..."; \
		go install github.com/golang/mock/mockgen@latest; \
	fi
	$(GO) generate ./pkg/...
	@echo "Mocks generated"

# -----------------------------------------------------------------------------
# security: Run security audit
# -----------------------------------------------------------------------------
security:
	@echo "Running security audit..."
	@if ! command -v govulncheck &> /dev/null; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...
	@echo "Security audit complete"

# -----------------------------------------------------------------------------
# audit: Full audit (security + lint + test)
# -----------------------------------------------------------------------------
audit: security lint test
	@echo "Full audit complete"

# =============================================================================
# UTILITY TARGETS
# =============================================================================

# -----------------------------------------------------------------------------
# help: Display this help message
# -----------------------------------------------------------------------------
help:
	@echo "BIOMETRICS CLI - Makefile Help"
	@echo "=================================="
	@echo ""
	@echo "Usage: make [target] [options]"
	@echo ""
	@echo "Primary Targets:"
	@echo "  all            Build, test, and lint (default)"
	@echo "  build          Build the binary"
	@echo "  test           Run tests with coverage"
	@echo "  lint           Run linters"
	@echo "  format         Format code"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Installation:"
	@echo "  install        Install to GOPATH/bin"
	@echo "  uninstall      Remove from GOPATH/bin"
	@echo ""
	@echo "Docker:"
	@echo "  docker         Build Docker image"
	@echo "  docker-build   Build with tags"
	@echo "  docker-run     Run container"
	@echo ""
	@echo "Code Quality:"
	@echo "  coverage       Generate coverage report"
	@echo "  vet            Run go vet"
	@echo "  mod            Tidy modules"
	@echo "  deps           Check dependencies"
	@echo ""
	@echo "Advanced:"
	@echo "  benchmark      Run benchmarks"
	@echo "  check          Full health check"
	@echo "  release        Create release build"
	@echo "  cross-compile  Build for multiple platforms"
	@echo "  generate       Run code generation"
	@echo "  mocks          Generate test mocks"
	@echo "  security       Security audit"
	@echo "  audit          Full audit"
	@echo ""
	@echo "Information:"
	@echo "  help           Display this help"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  GO_VERSION=$(GO_VERSION)"
	@echo "  BINARY=$(OUTPUT)"
	@echo ""

# =============================================================================
# END OF MAKEFILE
# =============================================================================
