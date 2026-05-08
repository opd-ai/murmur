# MURMUR Makefile
# Build and development targets for the MURMUR decentralized social network

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=gofumpt
BINARY_NAME=murmur
BINARY_DIR=bin
VERSION=$(shell cat VERSION 2>/dev/null || echo "0.0.0-dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"
LDFLAGS_STATIC=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -s -w"

# Protobuf parameters
PROTOC=protoc
PROTO_DIR=proto
PROTO_GO_OUT=proto

# Build targets
.PHONY: all build clean test lint fmt proto install help wasm-site

all: fmt lint test build

build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/murmur
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/murmur-tui ./cmd/murmur-tui

wasm-site:
	@echo "Building WASM GitHub Pages bundle..."
	@VERSION=$(VERSION) COMMIT=$(COMMIT) ./scripts/build-wasm-site.sh

build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/murmur
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/murmur

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/murmur
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/murmur

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/murmur

# Static builds (headless mode, no GUI/Ebitengine)
build-static:
	@echo "Building static $(BINARY_NAME) $(VERSION) (headless)..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS_STATIC) -tags noebiten -o $(BINARY_DIR)/$(BINARY_NAME)-static ./cmd/murmur

build-static-all: build-static-linux build-static-darwin build-static-windows

build-static-linux:
	@echo "Building static for Linux (headless)..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS_STATIC) -tags noebiten -o $(BINARY_DIR)/$(BINARY_NAME)-static-linux-amd64 ./cmd/murmur
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS_STATIC) -tags noebiten -o $(BINARY_DIR)/$(BINARY_NAME)-static-linux-arm64 ./cmd/murmur

build-static-darwin:
	@echo "Building static for macOS (headless)..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS_STATIC) -tags noebiten -o $(BINARY_DIR)/$(BINARY_NAME)-static-darwin-amd64 ./cmd/murmur
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS_STATIC) -tags noebiten -o $(BINARY_DIR)/$(BINARY_NAME)-static-darwin-arm64 ./cmd/murmur

build-static-windows:
	@echo "Building static for Windows (headless)..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS_STATIC) -tags noebiten -o $(BINARY_DIR)/$(BINARY_NAME)-static-windows-amd64.exe ./cmd/murmur

install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(LDFLAGS) ./cmd/murmur

# Package targets
package:
	@echo "Creating release packages..."
	@./scripts/package.sh all

package-linux:
	@echo "Creating Linux packages..."
	@./scripts/package.sh linux-amd64
	@./scripts/package.sh linux-arm64

package-darwin:
	@echo "Creating macOS packages..."
	@./scripts/package.sh darwin-amd64
	@./scripts/package.sh darwin-arm64

package-windows:
	@echo "Creating Windows packages..."
	@./scripts/package.sh windows-amd64

clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR) dist
	@$(GOCMD) clean

# Test targets
test:
	@echo "Running tests..."
	$(GOTEST) -race ./...

test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-coverage:
	@echo "Running coverage tests for critical packages..."
	@echo "Testing pkg/identity/..."
	@$(GOTEST) -coverprofile=coverage-identity.out ./pkg/identity/... >/dev/null 2>&1
	@echo "Testing pkg/content/..."
	@$(GOTEST) -coverprofile=coverage-content.out ./pkg/content/... >/dev/null 2>&1
	@echo "Testing pkg/anonymous/..."
	@$(GOTEST) -coverprofile=coverage-anonymous.out ./pkg/anonymous/... >/dev/null 2>&1
	@echo ""
	@echo "Coverage summary:"
	@echo "  pkg/identity/:  " $$($(GOCMD) tool cover -func=coverage-identity.out | tail -1 | awk '{print $$NF}')
	@echo "  pkg/content/:   " $$($(GOCMD) tool cover -func=coverage-content.out | tail -1 | awk '{print $$NF}')
	@echo "  pkg/anonymous/: " $$($(GOCMD) tool cover -func=coverage-anonymous.out | tail -1 | awk '{print $$NF}')
	@echo ""
	@echo "Target: >80% coverage for critical packages"
	@rm -f coverage-identity.out coverage-content.out coverage-anonymous.out

test-verbose:
	@echo "Running tests (verbose)..."
	$(GOTEST) -race -v ./...

# Lint and format targets
lint:
	@echo "Running vet..."
	$(GOVET) ./...

fmt:
	@echo "Formatting code..."
	@if command -v $(GOFMT) > /dev/null 2>&1; then \
		$(GOFMT) -w -extra .; \
	else \
		echo "gofumpt not found, using go fmt"; \
		$(GOCMD) fmt ./...; \
	fi

fmt-check:
	@echo "Checking format..."
	@if command -v $(GOFMT) > /dev/null 2>&1; then \
		$(GOFMT) -d -extra . | grep -q . && { echo "Code not formatted"; exit 1; } || echo "Format OK"; \
	else \
		test -z "$$($(GOCMD) fmt ./... 2>&1)" || { echo "Code not formatted"; exit 1; }; \
	fi

# Proto targets
proto:
	@echo "Generating protobuf code..."
	$(PROTOC) --go_out=$(PROTO_GO_OUT) --go_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

proto-check:
	@echo "Checking protobuf files..."
	$(PROTOC) --go_out=$(PROTO_GO_OUT) --go_opt=paths=source_relative \
		--dry_run $(PROTO_DIR)/*.proto

# Development targets
dev: fmt lint test build
	@echo "Development build complete"

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_DIR)/$(BINARY_NAME)

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOCMD) mod download

deps-update:
	@echo "Updating dependencies..."
	$(GOCMD) get -u ./...
	$(GOCMD) mod tidy

# Help
help:
	@echo "MURMUR Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build              - Build the binary for current platform"
	@echo "                       (builds bin/murmur and bin/murmur-tui)"
	@echo "  build-all          - Build for all platforms (linux, darwin, windows)"
	@echo "  build-linux        - Build for Linux (amd64, arm64)"
	@echo "  build-darwin       - Build for macOS (amd64, arm64)"
	@echo "  build-windows      - Build for Windows (amd64)"
	@echo "  build-static       - Build static binary (headless, no GUI)"
	@echo "  build-static-all   - Build static binaries for all platforms"
	@echo "  build-static-linux - Build static for Linux (amd64, arm64)"
	@echo "  build-static-darwin- Build static for macOS (amd64, arm64)"
	@echo "  build-static-windows- Build static for Windows (amd64)"
	@echo "  install            - Install the binary"
	@echo "  clean              - Remove build artifacts and dist directory"
	@echo ""
	@echo "Package targets:"
	@echo "  package        - Create release packages for all platforms"
	@echo "  package-linux  - Create Linux packages (amd64, arm64)"
	@echo "  package-darwin - Create macOS packages (amd64, arm64)"
	@echo "  package-windows- Create Windows package (amd64)"
	@echo ""
	@echo "Test targets:"
	@echo "  test          - Run tests with race detector"
	@echo "  test-cover    - Run tests with coverage report (HTML)"
	@echo "  test-coverage - Run coverage tests for critical packages (identity, content, anonymous)"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo ""
	@echo "Format and lint targets:"
	@echo "  fmt           - Format code with gofumpt (or go fmt)"
	@echo "  fmt-check     - Check if code is formatted"
	@echo "  lint          - Run go vet"
	@echo ""
	@echo "Proto targets:"
	@echo "  proto         - Generate Go code from protobuf files"
	@echo ""
	@echo "Development targets:"
	@echo "  dev           - Format, lint, test, and build"
	@echo "  run           - Build and run the application"
	@echo "  deps          - Download dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo ""
	@echo "Other:"
	@echo "  all           - Format, lint, test, and build (default)"
	@echo "  help          - Show this help message"
