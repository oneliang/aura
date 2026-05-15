.PHONY: all build test clean run install deps check-coverage ci fmt lint docs dev tidy

# Variables
GO := go
BINARY := aura
MAIN_PATH := ./cmd/aura

# Build flags
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell TZ=Asia/Shanghai date '+%Y.%m.%d(%H:%M:%S)')
LDFLAGS := -ldflags "-X github.com/oneliang/aura/shared/pkg/version.Version=$(VERSION) -X github.com/oneliang/aura/shared/pkg/version.BuildTime=$(BUILD_TIME)"

# Default target
all: deps build

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) work sync

# Build the binary
build:
	@echo "Building $(BINARY)..."
	$(GO) build $(LDFLAGS) -o bin/$(BINARY) $(MAIN_PATH)

# Run the application
run: build
	@echo "Running $(BINARY)..."
	./bin/$(BINARY)

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Check coverage threshold
check-coverage:
	@echo "Checking coverage threshold..."
	@coverage=$$($(GO) test -cover ./modules/... 2>&1 | grep -oE '[0-9]+\.[0-9]+%' | tr -d '%' | head -1); \
	if [ -z "$$coverage" ]; then \
		echo "No coverage data found"; \
		exit 1; \
	fi; \
	echo "Coverage: $$coverage%"; \
	passed=$$(echo "$$coverage >= $(COVERAGE_THRESHOLD)" | bc -l 2>/dev/null || echo "0"); \
	if [ "$$passed" = "1" ]; then \
		echo "Coverage check passed (>= $(COVERAGE_THRESHOLD)%)"; \
	else \
		echo "Coverage check failed: $$coverage% < $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi

# CI target - runs all checks for CI/CD
ci: deps fmt lint test-coverage check-coverage build
	@echo "CI checks completed successfully"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY)..."
	cp bin/$(BINARY) $(GOPATH)/bin/

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Generate documentation
docs:
	@echo "Generating documentation..."
	$(GO) doc -all ./...

# Development mode with hot reload
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

# Check Go environment
check:
	@echo "Go version: $$(go version)"
	@echo "GOPATH: $(GOPATH)"
	@echo "GOPROXY: $$(go env GOPROXY)"

# Tidy modules
tidy:
	$(GO) mod tidy -all
	$(GO) work sync

# Help
help:
	@echo "Available targets:"
	@echo "  all            - Download deps and build (default)"
	@echo "  deps           - Download dependencies"
	@echo "  build          - Build the binary"
	@echo "  run            - Build and run"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  check-coverage - Check coverage threshold"
	@echo "  ci             - Run all CI checks"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install to GOPATH/bin"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  docs           - Generate documentation"
	@echo "  dev            - Development mode with hot reload"
	@echo "  check          - Check Go environment"
	@echo "  tidy           - Tidy modules"
	@echo ""
	@echo "Variables:"
	@echo "  COVERAGE_THRESHOLD=$(COVERAGE_THRESHOLD)%"
