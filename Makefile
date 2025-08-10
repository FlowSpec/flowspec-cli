# FlowSpec CLI Makefile

.PHONY: all build test clean install lint fmt vet coverage help release

# ==============================================================================
# Variables
# ==============================================================================

BINARY_NAME=flowspec-cli
BUILD_DIR=build
MAIN_PATH=./cmd/flowspec-cli
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION ?= $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS = -ldflags "\
	-X main.version=$(VERSION) \
	-X main.commit=$(GIT_COMMIT) \
	-X main.date=$(BUILD_DATE) \
	-s -w"

# Release flags (optimized build)
RELEASE_LDFLAGS = -ldflags "\
	-X main.version=$(VERSION) \
	-X main.commit=$(GIT_COMMIT) \
	-X main.date=$(BUILD_DATE) \
	-s -w -extldflags '-static'"

# ==============================================================================
# Main Targets
# ==============================================================================

all: build ## Run main checks and build the binary

build: ## Build the binary
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build files and caches
	@echo "Cleaning build files..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f $(BINARY_NAME)
	rm -rf performance_reports/
	rm -rf coverage/

install: ## Install the binary to GOPATH
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PATH)

# ==============================================================================
# Quality & CI/CD
# ==============================================================================

fmt: ## Format Go source code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet to check for suspicious constructs
	@echo "Running go vet..."
	go vet ./...

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint is not installed, skipping lint check"; \
		echo "💡 Installation: https://golangci-lint.run/usage/install/"; \
	fi

coverage: ## Generate test coverage report
	@echo "Generating test coverage report..."
	@if [ -f "scripts/coverage.sh" ]; then \
		./scripts/coverage.sh; \
	else \
		go test -coverprofile=coverage.out ./...; \
		go tool cover -html=coverage.out -o coverage.html; \
		echo "Coverage report generated: coverage.html"; \
	fi

quality: fmt vet lint ## Run all code quality checks

ci: quality test coverage build ## Run full CI checks

ci-dev: fmt vet lint test build ## Run CI checks for development (skips coverage threshold)
	@echo "Running dev mode coverage check..."
	@DEV_MODE=true ./scripts/coverage.sh || true
	@echo "✅ Dev mode CI checks complete"

# ==============================================================================
# Release & Deployment
# ==============================================================================

build-release: ## Build a release version (optimized and static)
	@echo "Building release version $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(RELEASE_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-all: ## Build multi-platform binaries for release
	@echo "Building multi-platform binaries $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building Linux AMD64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(RELEASE_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 $(MAIN_PATH)
	@echo "Building Linux ARM64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(RELEASE_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64 $(MAIN_PATH)
	@echo "Building macOS AMD64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(RELEASE_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64 $(MAIN_PATH)
	@echo "Building macOS ARM64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(RELEASE_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64 $(MAIN_PATH)
	@echo "Building Windows AMD64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(RELEASE_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.exe $(MAIN_PATH)
	@echo "✅ Multi-platform build complete"

package: build-all ## Create release packages (tar.gz)
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/packages
	@for binary in $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-*; do \
		if [ -f "$$binary" ]; then \
			base=$$(basename $$binary); \
			platform=$$(echo $$base | sed 's/$(BINARY_NAME)-$(VERSION)-//'); \
			mkdir -p $(BUILD_DIR)/packages/$$platform; \
			cp $$binary $(BUILD_DIR)/packages/$$platform/$(BINARY_NAME)$$(echo $$platform | grep -q windows && echo .exe || echo ""); \
			cp README.md $(BUILD_DIR)/packages/$$platform/; \
			cp LICENSE $(BUILD_DIR)/packages/$$platform/; \
			cp CHANGELOG.md $(BUILD_DIR)/packages/$$platform/; \
			cd $(BUILD_DIR)/packages && tar -czf $(BINARY_NAME)-$(VERSION)-$$platform.tar.gz $$platform/; \
			cd ../..; \
		fi \
	done
	@echo "✅ Release packages created"

release-prepare: clean deps quality test coverage ## Prepare for a release
	@echo "🚀 Preparing release $(VERSION)..."
	@if [ "$(VERSION)" = "0.1.0-dev" ]; then \
		echo "❌ Please set a correct version number: make release-prepare VERSION=x.y.z"; \
		exit 1; \
	fi
	@echo "✅ Release preparation checks passed"

release: release-prepare build-all package ## Run the full release process
	@echo "🎉 Release $(VERSION) complete!"
	@echo "📦 Release packages are in: $(BUILD_DIR)/packages/"
	@ls -la $(BUILD_DIR)/packages/*.tar.gz
	@echo ""
	@echo "📋 Release checklist:"
	@echo "  ✅ Code quality checks passed"
	@echo "  ✅ Test coverage met"
	@echo "  ✅ Multi-platform binaries built"
	@echo "  ✅ Release packages created"
	@echo ""
	@echo "🔄 Next steps:"
	@echo "  1. Create Git tag: make tag VERSION=$(VERSION)"
	@echo "  2. Push tag: git push origin v$(VERSION)"
	@echo "  3. Create a Release on GitHub"
	@echo "  4. Upload packages to the GitHub Release"

release-check: ## Check if the repository is in a releasable state
	@echo "🔍 Checking release readiness..."
	@echo "Checking Git status..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Working directory is not clean. Please commit all changes."; \
		git status --short; \
		exit 1; \
	fi
	@echo "✅ Git working directory is clean"
	@echo "Checking version number..."
	@if [ "$(VERSION)" = "0.1.0-dev" ]; then \
		echo "❌ Please set a correct version number"; \
		exit 1; \
	fi
	@echo "✅ Version: $(VERSION)"
	@echo "Checking if tag already exists..."
	@if git tag -l | grep -q "^v$(VERSION)$$"; then \
		echo "❌ Tag v$(VERSION) already exists"; \
		exit 1; \
	fi
	@echo "✅ Tag v$(VERSION) is available"
	@echo "🎯 Release readiness check passed"

tag: ## Create a new Git tag for a release
	@if [ "$(VERSION)" = "0.1.0-dev" ]; then \
		echo "❌ Cannot create tag for dev version"; \
		exit 1; \
	fi
	@echo "Creating tag v$(VERSION)..."
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@echo "✅ Tag v$(VERSION) created"
	@echo "💡 Push the tag using 'git push origin v$(VERSION)'"

# ==============================================================================
# Development & Utility
# ==============================================================================

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

dev: build ## Build and run in development mode
	@echo "Running in development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) --help

version: ## Display version information
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

# ==============================================================================
# Performance Testing
# ==============================================================================

performance-test: ## Run performance tests using the script
	@echo "Running performance tests..."
	@if [ -f "scripts/performance-test.sh" ]; then \
		./scripts/performance-test.sh; \
	else \
		echo "⚠️  Performance test script not found"; \
	fi

stress-test: ## Run stress tests
	@echo "Running stress tests..."
	go test -v -run "TestStress" ./cmd/flowspec-cli/ -timeout 90m

benchmark: ## Run all benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -count=3 ./...

# ==============================================================================
# Help
# ==============================================================================

help: ## Display this help message
	@echo "FlowSpec CLI Build System"
	@echo "========================="
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Example: make release VERSION=1.2.3"