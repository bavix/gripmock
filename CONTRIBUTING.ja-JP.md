# GripMock への貢献

**言語：** [English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md) | 日本語 | [Deutsch](CONTRIBUTING.de.md) | [Español](CONTRIBUTING.es.md)

> 注意：このページは機械翻訳により生成されています。内容に不正確または不完全な箇所がある可能性があります。英語の原文 [`CONTRIBUTING.md`](CONTRIBUTING.md) を参照としてください。

GripMock への貢献に興味を持っていただきありがとうございます！このドキュメントはプロジェクトへの貢献のためのガイドラインを提供します。

## はじめに

1. **リポジトリをフォーク** し、フォークをローカルにクローンします
2. **開発環境をセットアップ** します：
   - 統合テスト用の [grpctestify](https://github.com/gripmock/grpctestify-rust) をインストールします（インストール手順は [grpctestify ドキュメント](https://gripmock.github.io/grpctestify-rust/) を参照）
   - Go がインストールおよび設定されていることを確認します

### ConnectRPC テスト

**HTTP クライアントテスト**（`.http` ファイル）は `examples/projects/*/connectrpc-tests.http` にあります。

**httpyac での実行：**
```bash
npx httpyac run examples/projects/greeter/connectrpc-tests/ --all
```

JetBrains IDE（GoLand、IntelliJ）で `.http` ファイルを開き、各リクエストの横にある実行アイコンをクリックすることもできます。

## テスト要件

### ⚠️ 重要なルール

#### 1. gRPC サーバーの変更には統合テストが必要

**gRPC サーバー機能に関連するものを変更、追加、修正した場合は、`.gctf` 形式で grpctestify を使用した統合テストを作成する必要があります。**

統合テストは `examples/` ディレクトリにあります。`.gctf` ファイルの例：

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**テストの実行：**
```bash
make test              # ユニットテスト
grpctestify examples/  # 統合テスト
make lint              # リンター
```

**テストの配置場所：**
- 統合テスト：`examples/projects/*/case_*.gctf`
- ユニットテスト：`internal/app/*_internal_test.go`

#### 2. すべての PR にテストを含める必要があります

すべてのプルリクエストには、特にバグ修正と新機能に対して、適切なテストを含める必要があります。

#### 3. ローカルでテストを実行

PR を提出する前に、すべてのテストが合格することを確認してください：

**grpctestify を使用した統合テスト：**
```bash
# サーバーを起動（別のターミナルで）
go run main.go examples -s examples

# 統合テストを実行
grpctestify examples/
```

**ユニットテスト：**
```bash
make test
make lint
```

## 後方互換性

**すべての変更は後方互換性を維持する必要があります**。issue を通じて明示的に議論され承認されない限り、破壊的変更は許可されません。

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
- [ ] ブランチがメインブランチと最新の状態である

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

- 既存の issue と discussions を確認
- `question` ラベルで新しい issue を開く
- [ドキュメント](https://bavix.github.io/gripmock/) を参照

## 追加リソース

- [プロジェクトドキュメント](https://bavix.github.io/gripmock/)
- [grpctestify ドキュメント](https://gripmock.github.io/grpctestify-rust/)

GripMock への貢献ありがとうございます！🚀
