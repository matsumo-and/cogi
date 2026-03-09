# Cogi - Technical Specification

> **Cogi**: A Code Intelligence Engine for exploring and understanding codebases

## Overview

Cogi is a local-first, multi-repository code intelligence engine that leverages Tree-sitter for parsing, SQLite for storage and full-text search, and vector embeddings for semantic search to enable advanced code search and RAG (Retrieval-Augmented Generation) capabilities.

## Goals

- Parse and index multi-repository codebases for RAG applications
- Enable semantic and keyword-based code search
- Support codebase specification understanding and analysis

---

## Supported Languages

### Full Support (Tree-sitter based)
- Go
- JavaScript / TypeScript
- Java
- Python
- Rust
- C#
- HTML / CSS
- JSON

### Text Fallback
- Markdown
- XML
- YAML / TOML / INI
- Other text files

Unsupported file extensions are automatically processed with the text parser to extract basic document structure.

---

## Architecture

### Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go |
| Parser | Tree-sitter (language grammars) |
| Metadata DB | SQLite |
| Full-text Search | SQLite FTS5 |
| Vector Search | SQLite BLOB + Cosine Similarity |
| Embeddings | Ollama (nomic-embed-text, etc.) |
| Interface | CLI |

### Database Architecture

**Unified SQLite Database**:
1. **SQLite**: Metadata, Symbol Index, Call/Import Graphs, Ownership Index
2. **SQLite FTS5**: Full-text search (keyword search) - built-in
3. **Vector Embeddings**: Stored as BLOB with cosine similarity search

### SQLite FTS5 Setup

**Features**:
- Built-in SQLite full-text search engine
- No additional dependencies or processes
- Auto-sync via triggers (maintenance-free)

**Search Capabilities**:
- Keyword search (AND/OR/NOT)
- Phrase search ("exact match")
- Prefix matching (prefix*)
- BM25 ranking

**Limitations**:
- No typo tolerance (exact matching)
- No automatic CamelCase splitting (can be implemented if needed)

---

## Core Components

### 1. Symbol Index

Indexes all symbols (functions, classes, variables, types) in the codebase.

**Stored Information**:
- Symbol name
- Symbol kind (function, class, method, variable, type, interface, etc.)
- File path
- Line and column numbers
- Scope information
- Documentation comments
- Visibility (public/private/protected)
- Repository ID

**Use Cases**:
- Symbol search
- Definition jump
- Reference search

### 2. Call Graph

Represents function/method call relationships as a directed graph.

**Stored Information**:
- Caller symbol ID
- Callee symbol ID
- Call location (file path, line number)
- Call type (direct call, method call, indirect call)

**Use Cases**:
- Impact analysis
- Dependency visualization
- Dead code detection

### 3. Import Graph

Represents file/module dependencies as a graph.

**Stored Information**:
- Source file path
- Imported module/package
- Import type (named, default, wildcard, etc.)
- Repository ID

**Use Cases**:
- Dependency visualization
- Circular dependency detection
- Module boundary analysis

### 4. Vector Index (Multi-Granularity)

Enables semantic search by vectorizing code at multiple levels.

#### Granularity Levels

**a. Class/Struct Level (Overview)**
- Target: Classes, structs, interfaces
- Content: Class definition + method signatures + documentation
- Use case: Understanding "what is this class" and "what methods does it have"

**b. Function/Method Level (Details)**
- Target: Individual functions/methods
- Content: Function body + documentation + signature
- Use case: Understanding "how is this implemented"

**Stored Information**:
- Symbol ID (links to Symbol Index)
- Embedding vector (dimension: model-dependent, e.g., 768)
- Granularity level (class/function)
- Repository ID
- File path
- Language
- Symbol kind
- Symbol name
- Text content (snippet for display)

**Embedding Model**:
- Executed via Ollama (e.g., `nomic-embed-text`, `mxbai-embed-large`)
- Local execution required

**Use Cases**:
- Semantic search (natural language queries for similar code)
- Code comprehension support

### 5. Ownership Index

Tracks code ownership based on git history.

**Stored Information**:
- File path
- Line range
- Author name
- Last update timestamp
- Commit count
- Repository ID

**Data Source**:
- Parsed from `git blame` output

**Use Cases**:
- Code ownership tracking
- Reviewer recommendation
- Change impact analysis

---

## Database Schema (SQLite)

### Table Design

```sql
-- Repository Management
CREATE TABLE repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    last_indexed_at INTEGER, -- Unix timestamp
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- File Management
CREATE TABLE files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    path TEXT NOT NULL, -- Relative path from repository root
    language TEXT NOT NULL,
    last_modified INTEGER NOT NULL, -- Unix timestamp
    file_hash TEXT NOT NULL, -- SHA256
    indexed_at INTEGER NOT NULL,
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    UNIQUE(repository_id, path)
);

-- Symbols
CREATE TABLE symbols (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL, -- function, class, method, variable, etc.
    start_line INTEGER NOT NULL,
    start_column INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    end_column INTEGER NOT NULL,
    scope TEXT, -- namespace, class name, etc.
    visibility TEXT, -- public, private, protected
    docstring TEXT,
    signature TEXT, -- Function signature, etc.
    code_body TEXT, -- Code body (for full-text search and vectorization)
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_kind ON symbols(kind);
CREATE INDEX idx_symbols_file_id ON symbols(file_id);

-- Call Graph
CREATE TABLE call_graph (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    caller_symbol_id INTEGER NOT NULL,
    callee_symbol_id INTEGER,
    callee_name TEXT NOT NULL, -- Record name even if ID is null (external functions)
    call_line INTEGER NOT NULL,
    call_column INTEGER NOT NULL,
    call_type TEXT, -- direct, method, indirect
    FOREIGN KEY (caller_symbol_id) REFERENCES symbols(id) ON DELETE CASCADE,
    FOREIGN KEY (callee_symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);
CREATE INDEX idx_call_graph_caller ON call_graph(caller_symbol_id);
CREATE INDEX idx_call_graph_callee ON call_graph(callee_symbol_id);

-- Import Graph
CREATE TABLE import_graph (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    import_path TEXT NOT NULL,
    import_type TEXT, -- named, default, wildcard
    imported_symbols TEXT, -- JSON array
    line_number INTEGER NOT NULL,
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);
CREATE INDEX idx_import_graph_file ON import_graph(file_id);
CREATE INDEX idx_import_graph_path ON import_graph(import_path);

-- Ownership
CREATE TABLE ownership (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    last_commit_hash TEXT NOT NULL,
    last_commit_date INTEGER NOT NULL, -- Unix timestamp
    commit_count INTEGER DEFAULT 1,
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);
CREATE INDEX idx_ownership_file ON ownership(file_id);
CREATE INDEX idx_ownership_author ON ownership(author_name);

-- Vector Embeddings
CREATE TABLE embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL,
    granularity TEXT NOT NULL, -- 'class' or 'function'
    vector BLOB NOT NULL, -- Embedding vector stored as BLOB
    content_hash TEXT NOT NULL, -- Hash of embedded content
    created_at INTEGER NOT NULL,
    FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);
CREATE INDEX idx_embeddings_symbol ON embeddings(symbol_id);
CREATE INDEX idx_embeddings_granularity ON embeddings(granularity);

-- Full-text Search Virtual Table (FTS5)
CREATE VIRTUAL TABLE symbols_fts USING fts5(
    symbol_id UNINDEXED,
    symbol_name,
    signature,
    docstring,
    code_body,
    file_path,
    content='symbols',
    content_rowid='id',
    tokenize='porter unicode61'  -- Porter stemmer + Unicode support
);

-- FTS5 Auto-sync Triggers
CREATE TRIGGER symbols_ai AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid, symbol_id, symbol_name, signature, docstring, code_body, file_path)
    SELECT new.id, new.id, new.name, new.signature, new.docstring, new.code_body,
           (SELECT path FROM files WHERE id = new.file_id);
END;

CREATE TRIGGER symbols_au AFTER UPDATE ON symbols BEGIN
    UPDATE symbols_fts
    SET symbol_name = new.name,
        signature = new.signature,
        docstring = new.docstring,
        code_body = new.code_body
    WHERE rowid = new.id;
END;

CREATE TRIGGER symbols_ad AFTER DELETE ON symbols BEGIN
    DELETE FROM symbols_fts WHERE rowid = old.id;
END;
```

---

## Update Strategy

### Change Detection

- **Timestamp-based**: Compare file `last_modified` with DB `indexed_at`
- **Hash Verification**: Use SHA256 hash to detect actual content changes

### Incremental Updates

1. Scan repositories for changed files
2. Re-parse only changed files
3. Partially update related indexes (Symbol, Call Graph, Vector, etc.)
4. Clean up indexes for deleted files

### Update Triggers

- **Manual**: Explicit index update via CLI command
- **Scheduled**: Periodic updates via cron or similar

---

## Performance Requirements

| Item | Requirement |
|------|-------------|
| Repository Count | 1-20 |
| Full Index Time | Max 5 minutes |
| Incremental Update | Seconds to tens of seconds (depends on changed file count) |
| Search Response | < 1 second |

### Optimization Strategies

- Parallel processing: Leverage goroutines per file/repository
- Batch embedding: Batch Ollama embedding requests
- FTS5 optimization: Periodic `PRAGMA optimize`
- SQLite settings: WAL mode, appropriate cache_size
- Vector search: Brute-force cosine similarity is sufficient for 1-20 repositories

---

## CLI Interface

### Command Design

```bash
# Repository Management
cogi add <repo-path> [--name <name>]
cogi remove <repo-name>

# Indexing
cogi index [--repo <name>] [--full]

# Search
cogi search symbol <query> [--kind <type>] [--repo <name>]
cogi search keyword <query> [--lang <language>] [--repo <name>]
cogi search semantic <query> [--granularity <class|function>] [--limit <n>]

# Graph Visualization
cogi graph calls <symbol-name> [--depth <n>] [--direction <caller|callee>]
cogi graph imports <file-path> [--depth <n>] [--direction <dependency|importer>]

# Ownership
cogi ownership <file-path> [--line <n>]

# Export
cogi export [--output <file>] [--type <all|symbols>] [--repo <name>]

# Status
cogi status

# Configuration
cogi config [--set <key>=<value>]
```

---

## References

- Tree-sitter: https://tree-sitter.github.io/tree-sitter/
- SQLite FTS5: https://www.sqlite.org/fts5.html
- SQLite BLOB: https://www.sqlite.org/datatype3.html
- Ollama: https://ollama.ai/
