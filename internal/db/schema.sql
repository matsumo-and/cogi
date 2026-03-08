-- Cogi Database Schema
-- SQLite with FTS5 support

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- リポジトリ管理
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    path TEXT NOT NULL UNIQUE,
    last_indexed_at INTEGER, -- Unix timestamp
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- ファイル管理
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    path TEXT NOT NULL, -- リポジトリルートからの相対パス
    language TEXT NOT NULL,
    last_modified INTEGER NOT NULL, -- Unix timestamp
    file_hash TEXT NOT NULL, -- SHA256
    indexed_at INTEGER NOT NULL,
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    UNIQUE(repository_id, path)
);

-- シンボル
CREATE TABLE IF NOT EXISTS symbols (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL, -- function, class, method, variable, etc.
    start_line INTEGER NOT NULL,
    start_column INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    end_column INTEGER NOT NULL,
    scope TEXT, -- namespace, class名など
    visibility TEXT, -- public, private, protected
    docstring TEXT,
    signature TEXT, -- 関数シグネチャなど
    code_body TEXT, -- コード本体（全文検索・ベクトル化用）
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);
CREATE INDEX IF NOT EXISTS idx_symbols_file_id ON symbols(file_id);

-- Call Graph
CREATE TABLE IF NOT EXISTS call_graph (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    caller_symbol_id INTEGER NOT NULL,
    callee_symbol_id INTEGER,
    callee_name TEXT NOT NULL, -- 外部関数の場合はIDがnullでも名前を記録
    call_line INTEGER NOT NULL,
    call_column INTEGER NOT NULL,
    call_type TEXT, -- direct, method, indirect
    FOREIGN KEY (caller_symbol_id) REFERENCES symbols(id) ON DELETE CASCADE,
    FOREIGN KEY (callee_symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_call_graph_caller ON call_graph(caller_symbol_id);
CREATE INDEX IF NOT EXISTS idx_call_graph_callee ON call_graph(callee_symbol_id);

-- Import Graph
CREATE TABLE IF NOT EXISTS import_graph (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    import_path TEXT NOT NULL,
    import_type TEXT, -- named, default, wildcard
    imported_symbols TEXT, -- JSON array
    line_number INTEGER NOT NULL,
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_import_graph_file ON import_graph(file_id);
CREATE INDEX IF NOT EXISTS idx_import_graph_path ON import_graph(import_path);

-- Ownership
CREATE TABLE IF NOT EXISTS ownership (
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

CREATE INDEX IF NOT EXISTS idx_ownership_file ON ownership(file_id);
CREATE INDEX IF NOT EXISTS idx_ownership_author ON ownership(author_name);

-- Vector Index（ベクトル埋め込み）
CREATE TABLE IF NOT EXISTS embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL,
    granularity TEXT NOT NULL, -- 'class' or 'function'
    vector BLOB NOT NULL, -- ベクトル（float32配列をバイナリ化）
    dimension INTEGER NOT NULL, -- ベクトル次元数
    content_hash TEXT NOT NULL, -- 埋め込み対象コンテンツのハッシュ
    created_at INTEGER NOT NULL,
    FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_embeddings_symbol ON embeddings(symbol_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_granularity ON embeddings(granularity);

-- 全文検索用仮想テーブル（FTS5）
CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
    symbol_id UNINDEXED,
    symbol_name,
    signature,
    docstring,
    code_body,
    file_path,
    content='symbols',
    content_rowid='id',
    tokenize='porter unicode61'  -- ポーターステマー + Unicode対応
);

-- FTS5の自動同期トリガー（INSERT）
CREATE TRIGGER IF NOT EXISTS symbols_ai AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid, symbol_id, symbol_name, signature, docstring, code_body, file_path)
    SELECT new.id, new.id, new.name, new.signature, new.docstring, new.code_body,
           (SELECT path FROM files WHERE id = new.file_id);
END;

-- FTS5の自動同期トリガー（UPDATE）
CREATE TRIGGER IF NOT EXISTS symbols_au AFTER UPDATE ON symbols BEGIN
    UPDATE symbols_fts
    SET symbol_name = new.name,
        signature = new.signature,
        docstring = new.docstring,
        code_body = new.code_body
    WHERE rowid = new.id;
END;

-- FTS5の自動同期トリガー（DELETE）
CREATE TRIGGER IF NOT EXISTS symbols_ad AFTER DELETE ON symbols BEGIN
    DELETE FROM symbols_fts WHERE rowid = old.id;
END;
