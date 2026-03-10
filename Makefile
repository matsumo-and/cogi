# Makefile for Cogi - Code Intelligence Engine
# All commands use -tags fts5 for SQLite FTS5 support

.PHONY: build install clean test test-coverage release run fmt lint tidy check help all

# Default target
all: build

# Build the binary with FTS5 support
build:
	@echo "Building cogi with FTS5 support..."
	go build -tags "fts5" -o cogi ./cmd/cogi
	@echo "✓ Build complete: ./cogi"

# Build for development (with race detector)
build-dev:
	@echo "Building cogi for development..."
	go build -tags "fts5" -race -o cogi ./cmd/cogi
	@echo "✓ Development build complete"

# Install to $GOPATH/bin
install:
	@echo "Installing cogi..."
	go install -tags "fts5" ./cmd/cogi
	@echo "✓ Installed to $(shell go env GOPATH)/bin/cogi"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f cogi
	rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# Clean database (use with caution!)
clean-db:
	@echo "⚠️  Warning: This will delete the Cogi database!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		rm -rf ~/.cogi/data.db*; \
		echo "✓ Database cleaned"; \
	else \
		echo "Cancelled"; \
	fi

# Run tests
test:
	@echo "Running tests with FTS5 support..."
	go test -tags "fts5" -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -tags "fts5" -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# Build for release
release:
	@echo "Building release binary..."
	go build -tags "fts5" -ldflags="-s -w" -o cogi ./cmd/cogi
	@echo "✓ Release build complete"

# Run the binary
run: build
	./cogi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not installed."; \
		echo "Install with: brew install golangci-lint"; \
		exit 1; \
	}
	golangci-lint run ./...
	@echo "✓ Lint complete"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy
	@echo "✓ Tidy complete"

# Run all checks (format, lint, tidy)
check: fmt lint tidy
	@echo "✓ All checks complete"

# Show help
help:
	@echo "Cogi Makefile - Code Intelligence Engine"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Important: All commands use -tags fts5 for SQLite FTS5 support"
	@echo ""
	@echo "Build targets:"
	@echo "  build         - Build the binary (default)"
	@echo "  build-dev     - Build with race detector for development"
	@echo "  install       - Build and install to GOPATH/bin"
	@echo "  release       - Build optimized release binary"
	@echo ""
	@echo "Test targets:"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo ""
	@echo "Development targets:"
	@echo "  run           - Build and run cogi"
	@echo "  fmt           - Format code with go fmt"
	@echo "  lint          - Run golangci-lint"
	@echo "  tidy          - Tidy go.mod dependencies"
	@echo "  check         - Run fmt, lint, and tidy"
	@echo ""
	@echo "Cleanup targets:"
	@echo "  clean         - Remove build artifacts"
	@echo "  clean-db      - Remove Cogi database (with confirmation)"
	@echo ""
	@echo "Other:"
	@echo "  help          - Show this help message"
