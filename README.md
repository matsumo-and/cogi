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
- **AI Integration**: MCP (Model Context Protocol) support for Claude Desktop, Cursor, and other AI assistants
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

#### Option 1: Install from GitHub Releases (Recommended)

```bash
# macOS (Apple Silicon)
curl -L -o cogi https://github.com/matsumo-and/cogi/releases/latest/download/cogi-darwin-arm64
chmod +x cogi
sudo mv cogi /usr/local/bin/

# Linux (x86_64)
curl -L -o cogi https://github.com/matsumo-and/cogi/releases/latest/download/cogi-linux-amd64
chmod +x cogi
sudo mv cogi /usr/local/bin/

# Windows (PowerShell)
curl -L -o cogi.exe https://github.com/matsumo-and/cogi/releases/latest/download/cogi-windows-amd64.exe
# Move to a directory in your PATH, e.g., C:\Program Files\cogi\
```

#### Option 2: Build from Source

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
- `cogi list` - List all indexed repositories
- `cogi remove` - Remove a repository from the index
- `cogi search symbol` - Search for symbol definitions
- `cogi search keyword` - Full-text keyword search
- `cogi search semantic` - Natural language semantic search
- `cogi search hybrid` - Hybrid search combining keyword and semantic
- `cogi graph calls` - Visualize function call relationships
- `cogi graph imports` - Visualize module dependencies
- `cogi ownership` - Show code ownership information
- `cogi export` - Export indexed data to JSON
- `cogi mcp` - Start MCP server for AI integration

## MCP Integration (Model Context Protocol)

Cogi supports [Model Context Protocol (MCP)](https://modelcontextprotocol.io/), enabling AI assistants like Claude to directly interact with your codebase. This allows natural language interactions for code search, analysis, and repository management.

### Starting the MCP Server

```bash
cogi mcp
```

The server uses stdio transport and exposes 12 powerful tools for AI assistants.

### Available MCP Tools

#### Search Tools
- **cogi_search_symbol** - Search for code symbols by name or kind
- **cogi_search_keyword** - Full-text keyword search using SQLite FTS5
- **cogi_search_semantic** - Semantic search using vector embeddings
- **cogi_search_hybrid** - Hybrid search combining keyword and semantic

#### Repository Management
- **cogi_add_repository** - Add a repository to the index
- **cogi_remove_repository** - Remove a repository from the index
- **cogi_list_repositories** - List all indexed repositories
- **cogi_status** - Get status and statistics

#### Indexing
- **cogi_index** - Build or update code index

#### Code Analysis
- **cogi_graph_calls** - Get call graph (callers/callees)
- **cogi_graph_imports** - Get import graph (dependencies)
- **cogi_ownership** - Query code ownership based on git blame

### MCP Client Configuration

#### Claude Desktop

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "cogi": {
      "command": "/usr/local/bin/cogi",
      "args": ["mcp"]
    }
  }
}
```

After updating the config, restart Claude Desktop.

#### Cursor

Add to your Cursor settings (`Settings` > `Features` > `Language Models` > `MCP Servers`):

```json
{
  "mcpServers": {
    "cogi": {
      "command": "cogi",
      "args": ["mcp"]
    }
  }
}
```

#### Generic MCP Clients

For other MCP-compatible tools, use the standard stdio configuration:

```json
{
  "command": "cogi",
  "args": ["mcp"],
  "env": {}
}
```

### Usage Examples

Once configured, you can interact with Cogi through natural language in your AI assistant:

**Example conversations:**
- "Add the ~/my-project repository to Cogi"
- "Search for database connection functions"
- "Who wrote the authentication module?"
- "Show me the call graph for the handleRequest function"
- "What files import the utils package?"
- "Index all repositories"

The AI assistant will automatically use the appropriate Cogi MCP tools to fulfill your requests.

### Requirements for MCP

- Cogi must be installed and accessible in your PATH
- For semantic search features: Ollama must be running (`ollama serve`)
- Repositories must be added and indexed before searching

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
  model: mxbai-embed-large
  endpoint: http://localhost:11434
  dimension: 1024
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

## Embedding Models

Cogi uses [Ollama](https://ollama.ai/) for generating code embeddings for semantic search. The choice of embedding model significantly impacts search accuracy.

### Recommended Models

| Model | Dimensions | Best For | Performance |
|-------|-----------|----------|-------------|
| **mxbai-embed-large** (default) | 1024 | Best accuracy for code search | Slower, higher quality |
| nomic-embed-text | 768 | Balanced performance | Faster, good quality |
| all-minilm | 384 | Fast indexing | Fastest, lower quality |

### Setup

1. Install Ollama from [ollama.ai](https://ollama.ai/)

2. Pull your preferred embedding model:
```bash
# Recommended (default)
ollama pull mxbai-embed-large

# Alternative: Faster but less accurate
ollama pull nomic-embed-text
```

3. Start Ollama server:
```bash
ollama serve
```

4. Configure in `~/.cogi/config.yaml` (optional):
```yaml
embedding:
  model: mxbai-embed-large  # or nomic-embed-text
  dimension: 1024           # 768 for nomic-embed-text
```

### Changing Models

When switching embedding models with different dimensions, Cogi will automatically detect the mismatch and regenerate all embeddings during the next `cogi index` run.

**Note**: This regeneration may take several minutes depending on your codebase size.

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
