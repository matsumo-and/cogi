.PHONY: build install clean test

# Build the binary with FTS5 support
build:
	go build -tags "fts5" -o cogi ./cmd/cogi

# Install to $GOPATH/bin
install:
	go install -tags "fts5" ./cmd/cogi

# Clean build artifacts
clean:
	rm -f cogi
	rm -rf ~/.cogi/data.db*

# Run tests
test:
	go test -tags "fts5" -v ./...

# Build for release
release:
	go build -tags "fts5" -ldflags="-s -w" -o cogi ./cmd/cogi
