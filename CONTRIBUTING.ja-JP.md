# GripMock への貢献

**言語：** [English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md) | 日本語 | [Deutsch](CONTRIBUTING.de.md) | [Español](CONTRIBUTING.es.md)

> 注意：このページは機械翻訳により生成されています。内容に不正確または不完全な箇所がある可能性があります。英語の原文 [`CONTRIBUTING.md`](CONTRIBUTING.md) を参照としてください。

## はじめに

1. **リポジトリをフォーク** し、フォークをローカルにクローンします
2. **開発環境をセットアップ** します：
   - 統合テスト用の [grpctestify](https://github.com/gripmock/grpctestify-rust) をインストールします（インストール手順は [grpctestify ドキュメント](https://gripmock.github.io/grpctestify-rust/) を参照）
   - Go がインストールおよび設定されていることを確認します

### ゲートウェイのテスト（ConnectRPC と gRPC-Web）

grpctestify は 3 つのワイヤープロトコルすべてに対応しているため、同じ `.gctf`
ファイルがそれぞれで通る必要があります。ゲートウェイのポートを指定し、
プロトコルを切り替えます：

```bash
GRPCTESTIFY_ADDRESS=localhost:4769 grpctestify run --protocol connectrpc examples/projects/greeter/
GRPCTESTIFY_ADDRESS=localhost:4769 grpctestify run --protocol grpc-web  examples/projects/greeter/
```

## テスト要件

### 1. gRPC サーバーの変更には統合テストが必要

gRPC サーバーの挙動を変える変更には、grpctestify による `.gctf` 形式の統合テストが必要です。

統合テストは `examples/` ディレクトリにあります。`.gctf` ファイルの例：

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**テストの配置場所：**
- 統合テスト：`examples/projects/*/case_*.gctf`
- ユニットテスト：`internal/app/*_internal_test.go`

### 2. すべての PR にテストを含める

バグ修正も新機能も、どちらにも必要です。

### 3. ローカルでテストを実行

```bash
make test    # ユニットテスト
make lint    # リンター
```

統合テストには、別のターミナルで起動したサーバーが必要です：

```bash
go run main.go examples -s examples
grpctestify examples/
```

## 後方互換性

すべての変更は後方互換性を維持する必要があります。破壊的変更は、issue で議論され承認された場合に限り許可されます。

### 破壊的変更のプロセス

破壊的変更を導入する必要がある場合：

1. **最初に Issue を作成**：以下を含む詳細な提案を issue として開きます：
   - 解決しようとしている問題の説明
   - 破壊的変更が必要な理由
   - 既存ユーザーのための移行パス

2. **承認を待つ**：メンテナーとの議論と承認なしに破壊的変更を実装しないでください

3. **移行ガイドを提供**：承認された場合、PR に明確な移行手順を含めてください

## プルリクエストプロセス

### 提出前

- [ ] すべてのテストがローカルで合格
- [ ] コードがプロジェクトのスタイルガイドに従っている（`make lint`）
- [ ] 必要に応じてドキュメントが更新されている
- [ ] ブランチが `master` と最新の状態である

### PR の説明

PR を作成する際は、以下を含めてください：
- 変更の説明
- 変更の種類（バグ修正、新機能など）
- テスト情報（ユニットテスト、gRPC サーバー変更の場合は統合テスト）
- 後方互換性の状態
- 関連する issue

## コードスタイル

- 標準の Go フォーマットに従う：`gofmt` と `goimports`
- リンターを実行：`make lint`
- 意味のある変数名と関数名を使用
- エクスポートされた関数と型にコメントを追加
- 新しいコードは `internal/` 下の適切なパッケージに配置

## ドキュメント

以下の場合にドキュメントを更新してください：
- 新機能の追加
- 既存の動作の変更
- ユーザーワークフローに影響するバグの修正

ドキュメントの場所：
- ユーザードキュメント：`docs/guide/`
- 例：`examples/` ディレクトリ
- メイン README：`README.md`

## 質問は？

まず既存の issue と discussions を確認し、その上で `question` ラベルを付けた
新しい issue を開いてください。

- [プロジェクトドキュメント](https://bavix.github.io/gripmock/)
- [grpctestify ドキュメント](https://gripmock.github.io/grpctestify-rust/)
