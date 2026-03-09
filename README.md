# Cogi 🐕

[![CI](https://github.com/matsumo-and/cogi/actions/workflows/ci.yml/badge.svg)](https://github.com/matsumo-and/cogi/actions/workflows/ci.yml)
[![Status](https://img.shields.io/badge/status-production-green)](https://github.com/matsumo-and/cogi)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

![Cogi Demo](assets/demo.gif)

**Cogi** turns your codebase into a knowledge base for AI.

## Features

- **Fully Local**: All processing runs locally without external dependencies
- **Fast Indexing**: Index multiple repositories in under 5 minutes
- **Multi-Language**: Full support for 10+ programming languages
- **Advanced Search**: Symbol, keyword, and semantic search capabilities
- **Code Analysis**: Call graphs, import graphs, and ownership tracking
- **Incremental Updates**: Efficiently re-index only changed files
- **Data Export**: Export indexed data in JSON format

## Supported Languages

### Full Support (Tree-sitter based)
Go, JavaScript, TypeScript, Python, Rust, Java, C#, HTML, CSS, JSON

### Text Fallback
Markdown, XML, YAML, TOML, INI, and other text files

Unknown file extensions are automatically processed with the text parser.

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/matsumo-and/cogi.git
cd cogi

# Build with FTS5 support
make build

# Or install globally
make install
```

### Basic Usage

```bash
# Add a repository
cogi add ~/my-project

# Build the index
cogi index

# Check status
cogi status

# Search for symbols
cogi search symbol "DatabaseConnection"

# Keyword search (full-text search)
cogi search keyword "parse json"

# Semantic search (requires Ollama)
cogi search semantic "function that opens a database"

# View call graph
cogi graph calls FunctionName --direction caller --depth 3

# View import dependencies
cogi graph imports path/to/file.go --direction dependency --depth 2

# Export data
cogi export --output data.json --type all
```

## Available Commands

- `cogi add` - Add a repository to the index
- `cogi index` - Build or update the code index
- `cogi status` - Show status and statistics
- `cogi search symbol` - Search for symbol definitions
- `cogi search keyword` - Full-text keyword search
- `cogi search semantic` - Natural language semantic search
- `cogi graph calls` - Visualize function call relationships
- `cogi graph imports` - Visualize module dependencies
- `cogi ownership` - Show code ownership information
- `cogi export` - Export indexed data to JSON

## Architecture

```
┌─────────────────────────────────────┐
│         CLI (Cobra)                 │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────┐
│       SQLite            │
│  ┌──────────────────┐   │
│  │ FTS5 (Search)    │   │
│  ├──────────────────┤   │
│  │ Vector (BLOB)    │   │
│  │ + Cosine Sim.    │   │
│  └──────────────────┘   │
└─────────────────────────┘
    │             │
┌───▼─────────┐ ┌▼────────┐
│ Tree-sitter │ │ Ollama  │
│  (Parser)   │ │ (Embed) │
└─────────────┘ └─────────┘
```

### Technology Stack

- **Parser**: Tree-sitter with language grammars
- **Database**: SQLite + FTS5 + Vector embeddings
- **Search**: SQLite BLOB + Cosine similarity (fully local)
- **Embeddings**: Ollama (nomic-embed-text, etc.)
- **CLI**: Cobra + Viper

## Core Components

### 1. Symbol Index
Indexes all code symbols (functions, classes, variables, types) across repositories.

### 2. Call Graph
Tracks function/method call relationships for impact analysis.

### 3. Import Graph
Maps module dependencies and detects circular imports.

### 4. Vector Index
Semantic search using code embeddings at multiple granularities:
- Class/struct level (overview)
- Function/method level (implementation details)

### 5. Ownership Index
Git blame-based code ownership tracking for collaboration insights.

## Requirements

- Go 1.21 or higher
- SQLite 3.35 or higher (FTS5 support)
  - **Important**: Build with `-tags fts5` (automatically handled by Makefile)
- [Ollama](https://ollama.ai/) (optional, for semantic search)

## Configuration

Default configuration file: `~/.cogi/config.yaml`

```yaml
database:
  path: ~/.cogi/data.db
  wal_mode: true
  cache_size_mb: 256

embedding:
  provider: ollama
  model: nomic-embed-text
  endpoint: http://localhost:11434
  dimension: 768
  batch_size: 32

indexing:
  max_file_size_mb: 10
  exclude_patterns:
    - "*/node_modules/*"
    - "*/vendor/*"
    - "*/.git/*"
    - "*/dist/*"
    - "*/build/*"

performance:
  max_workers: 8
```

## Performance

| Metric | Target |
|--------|--------|
| Repository Count | 1-20 |
| Full Index Time | < 5 minutes |
| Incremental Update | Seconds to tens of seconds |
| Search Response | < 1 second |

## Documentation

- [SPEC.md](./SPEC.md) - Technical specification
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Contribution guidelines

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Acknowledgments

- [Tree-sitter](https://tree-sitter.github.io/tree-sitter/)
- [Ollama](https://ollama.ai/)
- [SQLite](https://www.sqlite.org/)
