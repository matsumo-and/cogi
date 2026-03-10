# Contributing to Cogi

Thank you for your interest in contributing to Cogi! This document provides guidelines and best practices for contributing to the project.

## Table of Contents

- [Project Philosophy](#project-philosophy)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Architecture Principles](#architecture-principles)
- [Coding Guidelines](#coding-guidelines)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Review Process](#code-review-process)

## Project Philosophy

### Core Values

1. **Local-First**: All functionality must work completely locally
2. **Simplicity**: Avoid complexity, minimize dependencies
3. **Practicality**: Focus on features that genuinely help with code search and understanding
4. **Performance**: Index 1-20 repositories within 5 minutes

### Project Name

**Cogi** = **Corgi** (dog) + **Cognitive**

Like a smart Corgi, Cogi aims to quickly explore and deeply understand codebases.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- SQLite 3.35 or higher (FTS5 support)
- Git
- [Ollama](https://ollama.ai/) (optional, for semantic search)

### Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/cogi.git
cd cogi

# Add upstream remote
git remote add upstream https://github.com/matsumo-and/cogi.git
```

## Development Setup

### Build

```bash
# Build with FTS5 support (REQUIRED)
make build

# Or manually
go build -tags fts5 ./cmd/cogi

# Install globally
make install
```

**Important**: Always build with `-tags fts5` to enable SQLite FTS5 support.

### Run Tests

```bash
# Run all tests with FTS5 support
make test

# Or manually
go test -tags fts5 ./...

# Run specific package tests
go test -tags fts5 ./internal/parser
```

### Development Tools

We use the following tools for code quality:

```bash
# Format code
make fmt

# Lint code (requires golangci-lint)
make lint

# Tidy dependencies
make tidy

# Run all checks (fmt + lint + tidy)
make check
```

## Architecture Principles

### SQLite Unified Architecture

```
SQLite: Everything managed in SQLite
  ├─ Symbol Index (structured data)
  ├─ Call/Import Graph
  ├─ Ownership Index
  ├─ FTS5 (full-text search, auto-synced via triggers)
  └─ Embeddings (vector embeddings stored as BLOB)
```

**Design Intent**:
- Single SQLite database (simple, local-first)
- Vectors stored as BLOBs, searched via cosine similarity
- Complete consistency between metadata and vectors
- Brute-force search is fast enough for 1-20 repositories

### Multi-Granularity Vector Index

Two levels of granularity:

1. **Class/Struct Level (Overview)**
   - Class definition + method signatures + documentation
   - Use: "What is this class?" "What methods does it have?"

2. **Function/Method Level (Details)**
   - Function body + documentation + signature
   - Use: "How is this implemented?"

### Incremental Updates

- Timestamp-based change detection
- Re-parse only changed files
- Partial updates to SQLite (including embeddings)
- Proper cleanup on file deletion
- Transaction-based consistency

## Coding Guidelines

### Directory Structure

```
cogi/
├── cmd/
│   └── cogi/          # CLI entry point
├── internal/
│   ├── db/            # SQLite (FTS5 + vector embeddings)
│   ├── parser/        # Tree-sitter parsers
│   ├── indexer/       # Index builder
│   ├── search/        # Search engine (keyword + semantic)
│   ├── graph/         # Call/Import Graph
│   ├── vector/        # Vector similarity computation
│   ├── embedding/     # Ollama integration
│   ├── export/        # Data export functionality
│   ├── ownership/     # Git blame-based ownership
│   └── config/        # Configuration management
├── SPEC.md            # Technical specification
├── CONTRIBUTING.md    # This file
└── README.md          # User documentation
```

### Go Coding Style

- Follow standard Go conventions
- Use `gofmt` and `golangci-lint`
- Explicit error handling
- Proper context usage (timeouts, cancellation)
- Document exported functions and types

### Code Quality Principles

1. **Avoid Dependency Bloat**
   - Use minimal external libraries
   - Prefer standard library when possible

2. **Simple Setup**
   - Users should be able to start easily
   - Work with default configuration

3. **No External Service Dependencies**
   - Everything must run locally
   - Avoid external API dependencies

4. **Avoid Over-Engineering**
   - Follow YAGNI (You Aren't Gonna Need It)
   - Prioritize current usefulness over future extensibility

5. **Preserve Performance**
   - Maintain 5-minute indexing target
   - Monitor memory usage (balance parallel processing)

### Performance Requirements

**Mandatory Optimizations**:
- Parallel processing: Use goroutines per file/repository
- Batch processing: Batch Ollama embeddings and SQLite inserts
- SQLite settings: WAL mode, appropriate cache_size
- FTS5 optimization: Periodic `PRAGMA optimize`
- Vector search: Brute-force is sufficient for 1-20 repositories

**Performance Targets**:
- Index building: Max 5 minutes (full scan)
- Incremental update: Seconds to tens of seconds
- Search response: Under 1 second

### Data Consistency

**Important**:
- Everything managed in SQLite ensures consistency
- Use transactions appropriately
- Implement rollback strategy on errors

**FTS5 Triggers**:
```sql
-- Always set up auto-sync triggers for INSERT/UPDATE/DELETE
CREATE TRIGGER symbols_ai AFTER INSERT ON symbols ...
CREATE TRIGGER symbols_au AFTER UPDATE ON symbols ...
CREATE TRIGGER symbols_ad AFTER DELETE ON symbols ...
```

### Tree-sitter Integration

**Language Priority**:
1. Phase 1: Go, TypeScript, Python (primary languages)
2. Phase 5: Additional languages (Rust, C#, Java, HTML, CSS, etc.)

**Parsing Considerations**:
- Limit large files (default 10MB)
- Always apply exclusion patterns (node_modules, vendor, etc.)
- Show warnings and skip on parse errors

### Ollama Integration

**Batch Size**:
- Default: 32 (configurable)
- Balance memory and throughput

**Error Handling**:
- Clear error message when Ollama isn't running
- Keyword search should work even if semantic search is disabled

## Testing

### Required Tests

- **Unit Tests**: Each component
- **Integration Tests**: DB, Ollama integration
- **E2E Tests**: CLI commands

### Test Data

- Use small sample repositories
- Cover representative code patterns for each language

### Running Tests

```bash
# All tests
make test

# Specific package
go test -tags fts5 ./internal/parser -v

# With coverage
go test -tags fts5 -cover ./...
```

## Submitting Changes

### Before Submitting

1. Run all checks: `make check` (runs fmt, lint, and tidy)
2. Ensure all tests pass: `make test`
3. Update documentation if needed
4. Add tests for new functionality

### Commit Messages

Follow conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Test additions or fixes
- `chore`: Build process or tooling changes

Example:
```
feat(parser): add Rust language support

Implement Tree-sitter based parser for Rust including:
- Function and method parsing
- Struct, enum, and trait detection
- Use statement import tracking

Closes #123
```

### Pull Request Process

1. Create a feature branch: `git checkout -b feature/your-feature-name`
2. Make your changes
3. Commit with descriptive messages
4. Push to your fork: `git push origin feature/your-feature-name`
5. Create a Pull Request on GitHub
6. Ensure CI passes
7. Address review feedback

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
How was this tested?

## Checklist
- [ ] All checks pass (`make check`)
- [ ] Tests pass locally (`make test`)
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if applicable)
```

## Code Review Process

### What We Look For

1. **Correctness**: Does it work as intended?
2. **Performance**: Does it meet performance requirements?
3. **Simplicity**: Is it as simple as possible?
4. **Consistency**: Does it follow project conventions?
5. **Tests**: Are there adequate tests?
6. **Documentation**: Is it properly documented?

### Review Timeline

- Initial review: Within 3-5 days
- Follow-up reviews: Within 2 days
- Merge decision: After approval from maintainers

## Language Parser Guidelines

When adding support for a new language:

1. Create `internal/parser/<language>.go`
2. Implement parsing for:
   - Functions/methods
   - Classes/structs/interfaces
   - Variables/constants
   - Import/use statements
   - Call sites within functions
3. Add language constant to `parser.go`
4. Update `DetectLanguage()` function
5. Add integration in `Parse()` switch statement
6. Write comprehensive tests
7. Update documentation

## Release Process

### Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/):
- MAJOR: Breaking changes
- MINOR: Backward-compatible feature additions
- PATCH: Bug fixes

### Release Checklist

- [ ] All tests pass
- [ ] Documentation updated (README, SPEC, CHANGELOG)
- [ ] Performance requirements met
- [ ] Cross-platform testing (macOS, Linux, Windows)
- [ ] Version bumped appropriately
- [ ] Git tag created
- [ ] Release notes written

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open a GitHub Issue
- **Security**: Email maintainers directly

## Resources

### Official Documentation

- [Tree-sitter Documentation](https://tree-sitter.github.io/tree-sitter/)
- [SQLite FTS5](https://www.sqlite.org/fts5.html)
- [Ollama Documentation](https://ollama.ai/)

### Go Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Cogi!
