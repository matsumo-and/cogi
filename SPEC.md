# Cogi - Code Intelligence Engine

> **Cogi**
> コードベースを探索・理解する Code Intelligence Engine

## 概要

マルチリポジトリ対応のローカル実行可能なCode Intelligence Engine。Tree-sitterを使用してコードをパースし、複数のインデックスを構築することで、高度なコード検索とRAG（Retrieval-Augmented Generation）を実現する。

## 目的

- マルチリポジトリのコードをパースしてRAGとして活用
- セマンティック検索とキーワード検索によるコード検索
- コードベースの仕様理解支援

---

## 対応言語

以下の言語を対象とする：

- Go
- JavaScript / TypeScript
- Java
- Python
- Rust
- C#
- HTML / CSS
- XML
- Markdown
- Plain Text

---

## アーキテクチャ

### 技術スタック

| コンポーネント | 技術 |
|--------------|------|
| 実装言語 | Go |
| パーサー | Tree-sitter (各言語のgrammar) |
| メタデータDB | SQLite |
| 全文検索 | SQLite FTS5 |
| ベクトル検索 | Qdrant |
| 埋め込みモデル | Ollama (nomic-embed-text等) |
| インターフェース | CLI |

### データベース構成（ハイブリッド）

1. **SQLite**: メタデータ、Symbol Index、Call/Import Graph、Ownership Index
2. **SQLite FTS5**: 全文検索（キーワード検索）- SQLite組み込み
3. **Qdrant**: ベクトル検索（セマンティック検索）+ メタデータフィルタリング

### SQLite FTS5セットアップ

**特徴**:
- SQLiteに組み込みの全文検索エンジン
- 追加の依存関係・プロセス不要
- トリガーで自動同期（メンテナンスフリー）

**検索機能**:
- キーワード検索（AND/OR/NOT）
- フレーズ検索（"exact match"）
- 前方一致（prefix*）
- BM25ランキング

**制約**:
- タイポ耐性なし（完全一致）
- CamelCase自動分割なし（必要に応じて実装）

### Qdrantセットアップ

**起動方法**:
- **組み込みモード**: CLIがQdrantプロセスを自動起動・管理

**コレクション構成**:
```json
{
  "collection_name": "cogi",
  "vectors": {
    "size": 768,
    "distance": "Cosine"
  },
  "payload_schema": {
    "symbol_id": "integer",
    "granularity": "keyword",
    "repository_id": "integer",
    "file_path": "text",
    "language": "keyword",
    "symbol_kind": "keyword",
    "symbol_name": "text",
    "snippet": "text"
  }
}
```

---

## コアコンポーネント

### 1. Symbol Index

コードベース内の全シンボル（関数、クラス、変数、型など）をインデックス化。

**格納情報**:
- シンボル名
- シンボル種別 (function, class, method, variable, type, interface, etc.)
- ファイルパス
- 行番号・列番号
- スコープ情報
- ドキュメントコメント
- 可視性 (public/private/protected)
- リポジトリID

**用途**:
- シンボル検索
- 定義ジャンプ
- リファレンス検索

### 2. Call Graph

関数・メソッドの呼び出し関係を有向グラフとして表現。

**格納情報**:
- 呼び出し元シンボルID
- 呼び出し先シンボルID
- 呼び出し位置（ファイルパス、行番号）
- 呼び出し種別 (direct call, method call, indirect call)

**用途**:
- 影響範囲分析
- 依存関係の可視化
- デッドコード検出

### 3. Import Graph

ファイル/モジュール間の依存関係をグラフとして表現。

**格納情報**:
- インポート元ファイルパス
- インポート先モジュール/パッケージ
- インポート種別 (named, default, wildcard, etc.)
- リポジトリID

**用途**:
- 依存関係の可視化
- 循環依存検出
- モジュール境界の分析

### 4. Vector Index（複数粒度）

コードを意味ベクトル化して検索可能にする。

#### 粒度レベル

**a. クラス/構造体レベル（概要）**
- 対象: クラス、構造体、インターフェース全体
- 内容: クラス定義 + メソッドシグネチャ一覧 + ドキュメント
- 用途: 「どんなクラスか」「どんなメソッドがあるか」の理解

**b. 関数/メソッドレベル（詳細）**
- 対象: 個別の関数・メソッド
- 内容: 関数本体 + ドキュメント + シグネチャ
- 用途: 「どう実装されているか」の理解

**格納情報（Qdrantのpayload）**:
- シンボルID（Symbol Indexとの紐付け）
- 埋め込みベクトル（次元数: モデル依存、例: 768次元）
- 粒度レベル (class/function)
- リポジトリID
- ファイルパス
- 言語
- シンボル種別
- シンボル名
- テキスト内容（スニペット、検索結果表示用）

**埋め込みモデル**:
- Ollamaで実行（例: `nomic-embed-text`, `mxbai-embed-large`）
- ローカル実行必須

**Qdrantの利点**:
- メタデータとベクトルを一元管理
- フィルタリング検索（例: 「Goの関数のみ」「特定リポジトリのみ」）
- ハイブリッド検索（ベクトル類似度 + メタデータ条件）

**用途**:
- セマンティック検索（自然言語クエリで類似コード検索）
- コード理解支援

### 5. Ownership Index

各ファイル・関数の変更履歴から所有者情報を取得。

**格納情報**:
- ファイルパス
- 行範囲
- 著者名
- 最終更新日時
- コミット回数
- リポジトリID

**データソース**:
- `git blame` の結果をパース

**用途**:
- コードオーナーシップの把握
- レビュワー推薦
- 変更影響範囲の分析

---

## データスキーマ（SQLite）

### テーブル設計例

```sql
-- リポジトリ管理
CREATE TABLE repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    last_indexed_at INTEGER, -- Unix timestamp
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- ファイル管理
CREATE TABLE files (
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
CREATE TABLE symbols (
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
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_kind ON symbols(kind);
CREATE INDEX idx_symbols_file_id ON symbols(file_id);

-- Call Graph
CREATE TABLE call_graph (
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

-- Vector Index メタデータ（Qdrant同期用）
CREATE TABLE embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL,
    granularity TEXT NOT NULL, -- 'class' or 'function'
    qdrant_point_id TEXT NOT NULL UNIQUE, -- QdrantのPoint UUID
    content_hash TEXT NOT NULL, -- 埋め込み対象コンテンツのハッシュ
    created_at INTEGER NOT NULL,
    FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);
CREATE INDEX idx_embeddings_symbol ON embeddings(symbol_id);
CREATE INDEX idx_embeddings_granularity ON embeddings(granularity);
CREATE INDEX idx_embeddings_qdrant_id ON embeddings(qdrant_point_id);

-- 全文検索用仮想テーブル（FTS5）
CREATE VIRTUAL TABLE symbols_fts USING fts5(
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
CREATE TRIGGER symbols_ai AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid, symbol_id, symbol_name, signature, docstring, code_body, file_path)
    SELECT new.id, new.id, new.name, new.signature, new.docstring, new.code_body,
           (SELECT path FROM files WHERE id = new.file_id);
END;

-- FTS5の自動同期トリガー（UPDATE）
CREATE TRIGGER symbols_au AFTER UPDATE ON symbols BEGIN
    UPDATE symbols_fts
    SET symbol_name = new.name,
        signature = new.signature,
        docstring = new.docstring,
        code_body = new.code_body
    WHERE rowid = new.id;
END;

-- FTS5の自動同期トリガー（DELETE）
CREATE TRIGGER symbols_ad AFTER DELETE ON symbols BEGIN
    DELETE FROM symbols_fts WHERE rowid = old.id;
END;
```

---

## 更新戦略

### 差分検知

- **タイムスタンプベース**: ファイルの `last_modified` とDBの `indexed_at` を比較
- **ハッシュ検証**: 内容のSHA256ハッシュで実質的な変更を検知

### インクリメンタル更新

1. リポジトリをスキャンして変更ファイルを検出
2. 変更ファイルのみ再パース
3. 関連するインデックス（Symbol, Call Graph, Vector等）を部分更新
4. 削除されたファイルのインデックスをクリーンアップ

### 更新トリガー

- **手動実行**: CLI コマンドによる明示的なインデックス更新
- **定期実行**: cron等で定期的にインデックス更新を実行

---

## パフォーマンス要件

| 項目 | 要件 |
|------|------|
| 対象リポジトリ数 | 1〜20 |
| インデックス構築時間 | 最長5分（フルスキャン時） |
| インクリメンタル更新 | 変更ファイル数に応じて数秒〜数十秒 |
| 検索レスポンス | 1秒以内 |

### 最適化戦略

- 並列処理: ファイル/リポジトリ単位でゴルーチンを活用
- バッチ埋め込み: Ollamaへの埋め込みリクエストをバッチ化
- FTS5最適化: `PRAGMA optimize` で定期的にインデックス最適化
- バッチアップサート: Qdrantへのポイント追加をバッチ化
- 永続化: Qdrantのスナップショット機能で高速起動
- SQLite設定: WALモード、適切なcache_size設定

---

## CLI インターフェース

### コマンド設計（案）

```bash
# リポジトリ追加
cogi add <repo-path> [--name <name>]

# リポジトリ削除
cogi remove <repo-name>

# インデックス構築・更新
cogi index [--repo <name>] [--full]

# シンボル検索
cogi search symbol <query> [--kind <type>] [--repo <name>]

# キーワード検索（全文検索）
cogi search keyword <query> [--lang <language>] [--repo <name>]

# セマンティック検索
cogi search semantic <query> [--granularity <class|function>] [--limit <n>]

# Call Graph 表示
cogi graph calls <symbol-name> [--depth <n>] [--direction <caller|callee>]

# Import Graph 表示
cogi graph imports <file-path> [--depth <n>]

# Ownership 表示
cogi ownership <file-path> [--line <n>]

# ステータス確認
cogi status

# 設定確認・変更
cogi config [--set <key>=<value>]
```

---

## 実装フェーズ

### Phase 1: 基盤構築
- [ ] プロジェクト構造とCLI骨格
- [ ] SQLiteスキーマ作成
- [ ] Tree-sitter統合（主要言語: Go, TypeScript, Python）
- [ ] Symbol Index構築

### Phase 2: グラフ構築
- [ ] Call Graph生成
- [ ] Import Graph生成
- [ ] グラフ検索・可視化機能

### Phase 3: 検索機能
- [ ] FTS5全文検索実装（キーワード検索）
- [ ] Qdrant統合（起動・接続管理）
- [ ] Ollama連携（埋め込み生成）
- [ ] セマンティック検索実装（Qdrantクライアント）

### Phase 4: 高度な機能
- [ ] インクリメンタル更新
- [ ] 複数粒度ベクトルインデックス
- [ ] パフォーマンス最適化

### Phase 5: 拡張
- [ ] 残り言語対応（Rust, C#, Java等）
- [ ] 定期実行機能
- [ ] エクスポート機能（JSONなど）
- [ ] ドキュメント整備

---

## 依存ライブラリ（予定）

```go
// パーサー
github.com/smacker/go-tree-sitter
github.com/smacker/go-tree-sitter/[language]

// データベース（FTS5込み）
github.com/mattn/go-sqlite3

// ベクトル検索
github.com/qdrant/go-client

// Ollama連携
github.com/ollama/ollama/api

// CLI
github.com/spf13/cobra
github.com/spf13/viper

// ユーティリティ
github.com/go-git/go-git/v5
```

---

## 設定ファイル（案）

```yaml
# config.yaml
database:
  path: ~/.cogi/data.db
  # SQLite設定
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

---

## 今後の検討事項

- [ ] リポジトリ間の依存関係の扱い（モノレポ対応など）
- [ ] プライベートリポジトリの認証
- [ ] Webインターフェース（将来的に）
- [ ] LSP（Language Server Protocol）対応
- [ ] IDE拡張（VSCode, IntelliJ等）
- [ ] チーム共有機能（ネットワーク越しの検索）

---

## 参考

- Tree-sitter: https://tree-sitter.github.io/tree-sitter/
- SQLite FTS5: https://www.sqlite.org/fts5.html
- Ollama: https://ollama.ai/
