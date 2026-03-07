# Cogi 🐶

> **Cogi**
> コードベースを探索・理解する Code Intelligence Engine

[![Status](https://img.shields.io/badge/status-development-yellow)](https://github.com/yourusername/cogi)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

## 概要

**Cogi**はマルチリポジトリ対応のローカル実行可能なCode Intelligence Engineです。
Tree-sitter、SQLite FTS5、Qdrantを組み合わせて、高度なコード検索とRAG（Retrieval-Augmented Generation）を実現します。

### 特徴

- 🏠 **完全ローカル実行**: すべての処理がローカル環境で完結
- 🚀 **高速インデックス**: 最長5分でマルチリポジトリをインデックス化
- 🎯 **多様な検索**: キーワード、セマンティック、シンボル検索に対応
- 🔄 **インクリメンタル更新**: 変更ファイルのみを効率的に再インデックス
- 🌍 **多言語対応**: 10以上のプログラミング言語をサポート

## 主な機能

### 検索機能

- 🔍 **シンボル検索**: 関数・クラス・変数などの定義を高速検索
- 📝 **全文検索**: SQLite FTS5によるキーワード検索（BM25ランキング）
- 🧠 **セマンティック検索**: Qdrant + Ollamaによる自然言語検索

### グラフ機能

- 🕸️ **Call Graph**: 関数呼び出し関係の可視化・分析
- 📦 **Import Graph**: モジュール依存関係の追跡・循環依存検出

### 分析機能

- 👤 **Ownership Index**: git blameベースのコード所有者情報
- 📊 **影響範囲分析**: コード変更の影響を可視化

## 対応言語

Go, JavaScript, TypeScript, Java, Python, Rust, C#, HTML, CSS, XML, Markdown

## 使い方（予定）

### インストール

```bash
# Homebrewでインストール（予定）
brew install cogi

# または、ソースからビルド
git clone https://github.com/yourusername/cogi.git
cd cogi
go build -o cogi ./cmd/cogi
```

### 基本的な使い方

```bash
# リポジトリを追加
cogi add ~/my-project

# インデックスを構築
cogi index

# シンボル検索
cogi search symbol getUserById

# セマンティック検索（自然言語）
cogi search semantic "JSONをパースする処理"

# Call Graph表示
cogi graph calls handleRequest --depth 3

# Ownership表示
cogi ownership src/main.go --line 42
```

### 前提条件

- Go 1.21以上
- SQLite 3.35以上（FTS5サポート）
- [Ollama](https://ollama.ai/)（セマンティック検索用）

## アーキテクチャ

```
┌─────────────────────────────────────┐
│         CLI (Cobra)                 │
└──────────┬──────────────────────────┘
           │
    ┌──────┴──────┐
    │             │
┌───▼────┐   ┌───▼─────────────┐
│ SQLite │   │ Qdrant (Vector) │
│  FTS5  │   │  + Ollama       │
└────────┘   └─────────────────┘
    │
┌───▼───────────┐
│ Tree-sitter   │
│ (Parser)      │
└───────────────┘
```

### 技術スタック

- **パーサー**: Tree-sitter（各言語のgrammar）
- **メタデータDB**: SQLite + FTS5
- **ベクトル検索**: Qdrant
- **埋め込みモデル**: Ollama (nomic-embed-text等)
- **CLI**: Cobra + Viper

## ドキュメント

- [SPEC.md](./SPEC.md) - 詳細な仕様
- [CLAUDE.md](./CLAUDE.md) - 開発ガイドライン

## 開発ステータス

🚧 **現在開発中**

### ロードマップ

- [ ] Phase 1: 基盤構築（CLI、DB、Tree-sitter統合）
- [ ] Phase 2: グラフ構築（Call/Import Graph）
- [ ] Phase 3: 検索機能（FTS5、Qdrant、Ollama）
- [ ] Phase 4: 高度な機能（Ownership、インクリメンタル更新）
- [ ] Phase 5: 拡張（定期実行、エクスポート機能）

## コントリビューション

コントリビューションを歓迎します！詳細は [CONTRIBUTING.md](./CONTRIBUTING.md) を参照してください。

## ライセンス

MIT License - 詳細は [LICENSE](./LICENSE) を参照してください。

## 謝辞

- [Tree-sitter](https://tree-sitter.github.io/tree-sitter/)
- [Qdrant](https://qdrant.tech/)
- [Ollama](https://ollama.ai/)
- [SQLite](https://www.sqlite.org/)
