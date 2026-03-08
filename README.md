# Cogi 🐶

> **Cogi**
> コードベースを探索・理解する Code Intelligence Engine

[![CI](https://github.com/matsumo-and/cogi/actions/workflows/ci.yml/badge.svg)](https://github.com/matsumo-and/cogi/actions/workflows/ci.yml)
[![Status](https://img.shields.io/badge/status-development-yellow)](https://github.com/yourusername/cogi)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue)](https://go.dev/)

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

## 使い方

### インストール

```bash
# ソースからビルド
git clone https://github.com/yourusername/cogi.git
cd cogi

# FTS5サポート付きでビルド
make build

# またはインストール
make install
```

### クイックスタート

```bash
# リポジトリを追加
./cogi add ~/my-project

# インデックスを構築
./cogi index

# ステータス確認
./cogi status

# シンボル検索
./cogi search symbol New

# キーワード検索（全文検索）
./cogi search keyword "parse"

# Call Graph表示（呼び出し元を探す）
./cogi graph calls FunctionName --direction caller --depth 3

# Call Graph表示（呼び出し先を探す）
./cogi graph calls FunctionName --direction callee --depth 3

# Import Graph表示（依存関係を見る）
./cogi graph imports path/to/file.go --direction dependency --depth 2

# Import Graph表示（どこから使われているか）
./cogi graph imports path/to/file.go --direction importer --depth 2
```

### 現在利用可能な機能

- ✅ リポジトリ追加 (`cogi add`)
- ✅ インデックス構築 (`cogi index`)
- ✅ ステータス表示 (`cogi status`)
- ✅ シンボル検索 (`cogi search symbol`)
- ✅ キーワード検索 (`cogi search keyword`)
- ✅ Call Graph表示 (`cogi graph calls`)
- ✅ Import Graph表示 (`cogi graph imports`)
- ⏳ セマンティック検索（Phase 3で実装予定）
- ⏳ Ownership Index（Phase 4で実装予定）

### 前提条件

- Go 1.21以上
- SQLite 3.35以上（FTS5サポート）
- [Ollama](https://ollama.ai/)（セマンティック検索用、Phase 3）

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

- [x] **Phase 1: 基盤構築** ✅ 完了
  - [x] CLI骨格（Cobra + Viper）
  - [x] SQLiteスキーマ + FTS5
  - [x] 設定管理パッケージ
  - [x] Tree-sitter統合（Go, TypeScript, Python）
  - [x] Symbol Index構築
  - [x] 基本的な検索機能（シンボル検索、キーワード検索）
- [x] **Phase 2: グラフ構築** ✅ 完了
  - [x] Call Graph生成
  - [x] Import Graph生成
  - [x] グラフ検索・可視化機能
- [ ] **Phase 3: セマンティック検索**
  - [ ] Qdrant統合（起動・接続管理）
  - [ ] Ollama連携（埋め込み生成）
  - [ ] セマンティック検索実装
  - [ ] 複数粒度ベクトルインデックス
- [ ] **Phase 4: 高度な機能**
  - [ ] Ownership Index (git blame)
  - [ ] インクリメンタル更新
  - [ ] パフォーマンス最適化
- [ ] **Phase 5: 拡張**
  - [ ] 残り言語対応（Rust, C#, Java等）
  - [ ] 定期実行機能
  - [ ] エクスポート機能（JSONなど）
  - [ ] ドキュメント整備

## コントリビューション

コントリビューションを歓迎します！詳細は [CONTRIBUTING.md](./CONTRIBUTING.md) を参照してください。

## ライセンス

MIT License - 詳細は [LICENSE](./LICENSE) を参照してください。

## 謝辞

- [Tree-sitter](https://tree-sitter.github.io/tree-sitter/)
- [Qdrant](https://qdrant.tech/)
- [Ollama](https://ollama.ai/)
- [SQLite](https://www.sqlite.org/)
