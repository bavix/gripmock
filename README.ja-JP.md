![GripMock](https://github.com/bavix/gripmock/assets/5111255/d1fc10ef-2149-4302-8e24-aef4fdfe043c)

[![Coverage Status](https://coveralls.io/repos/github/bavix/gripmock/badge.svg?branch=master)](https://coveralls.io/github/bavix/gripmock?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/bavix/gripmock/v3)](https://goreportcard.com/report/github.com/bavix/gripmock/v3)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# GripMock 🚀

**言語：** [English](README.md) | [简体中文](README.zh-CN.md) | 日本語 | [Deutsch](README.de.md) | [Español](README.es.md)

> 注意：このページは機械翻訳により生成されています。内容に不正確または不完全な箇所がある可能性があります。英語の原文 [`README.md`](README.md) を参照としてください。

**テストと開発のための最速かつ最も信頼性の高い gRPC モックサーバー。**

GripMock は `.proto` ファイルまたはコンパイル済みの `.pb` 記述子からモックサーバーを作成し、gRPC テストをシンプルかつ効率的にします。エンドツーエンドテスト、開発環境、CI/CD パイプラインに最適です。

![greeter](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-greeter.gif)

## ✨ 機能

- **ネイティブランタイム** - 実行時の gRPC コード生成が不要なシングルプロセスエンジン
- **記述子ソース** - `.proto`、コンパイル済み `.pb`、BSR モジュール、gRPC reflection から API をロード
- **動的 `.pb` サービスのロード** - 再起動不要で実行時にコンパイル済み protobuf 記述子を API 経由でロード
- **ホットスタブ管理** - サーバー再起動なしで API/UI からスタブを作成・更新・削除
- **柔軟なマッチング** - `equals`、`contains`、`matches`、`glob`、ヘッダー、優先度、マッチ制限
- **配列順序の柔軟性** - 配列順序を無視してテストの脆弱性を低減
- **動的テンプレート** - リクエストペイロード、ヘッダー、ストリームコンテキストからレスポンスを構築
- **完全な gRPC カバレッジ** - Unary、サーバーストリーミング、クライアントストリーミング、双方向ストリーミング
- **エラー・詳細・遅延シミュレーション** - 実際の gRPC ステータスコード、詳細（`Any`）、レスポンスタイミングを返却
- **TLS および mTLS サポート** - ネイティブ TLS オプションでセキュアな gRPC/HTTP テスト環境を実行
- **高度な Protobuf タイプサポート** - well-known および extended protobuf タイプをサポート（`google.protobuf.*`、`google.type.*`）
- **YAML/JSON + Schema** - 両方の形式でスタブを作成、JSON Schema IDE バリデーション対応
- **プラグインエコシステム** - Go プラグインで関数を拡張、ビルダーイメージタグに対応
- **組み込み Faker テンプレート** - テンプレート内で直接リアルな擬似 person/contact/geo/network データを生成（`faker.*`）
- **OpenTelemetry トレーシング** - gRPC および HTTP パスの OTLP トレーシング（`otelgrpc` + `otelhttp`）
- **Prometheus メトリクス（`/metrics`）** - ランタイム/プロセスメトリクス（`go_*`、`process_*`）および GripMock メトリクス
- **運用 API** - ヘルスエンドポイント、descriptors API、stubs API、Web ダッシュボード
- **Embedded SDK（試験運用）** - Go テスト内で GripMock を実行し、検証ヘルパーを提供
- **MCP API（試験運用）** - エージェントおよびツール統合のためのストリーム可能な MCP エンドポイント
- **Upstream Modes（試験運用）** - 実際のアップストリームサービスからローカルモックへの段階的移行のための `proxy`、`replay`、`capture` モード

## 📚 ドキュメント

**[完全なドキュメント](https://bavix.github.io/gripmock)** - サンプル付きの完全なガイド

- **Descriptor API（`/api/descriptors`）**：検証可能な curl ワークフロー付きでコンパイル済み proto 記述子（`.pb`）を実行時にロード：[ドキュメント](https://bavix.github.io/gripmock/guide/api/descriptors)
- **Upstream Modes（試験運用）**：`proxy`、`replay`、`capture` と実用的なロールアウトガイダンス：[ドキュメント](https://bavix.github.io/gripmock/guide/modes)
- **Embedded SDK（試験運用）**：`sdk.NewServer`、`Match()`、`Return()` と検証を使用したプロセス内テスト：[ドキュメント](https://bavix.github.io/gripmock/guide/embedded-sdk)
- **Faker リファレンス**：組み込み Faker のキーごとのカタログと例：[ドキュメント](https://bavix.github.io/gripmock/guide/stubs/faker)
- **OpenTelemetry + メトリクス**：トレーシング環境変数と `/metrics` の動作：[ドキュメント](https://bavix.github.io/gripmock/guide/introduction/advanced-usage)
- **GitHub Actions（CI/CD）**：公式ワークフローアクションで GripMock を自動的にダウンロード、起動、準備完了待機、停止：[ドキュメント](https://bavix.github.io/gripmock/guide/ci-cd/github-actions)

## 🧬 プロジェクトの進化

GripMock は [tokopedia/gripmock](https://github.com/tokopedia/gripmock) のフォークとして開始され、その後独立した完全に書き直されたプロジェクトに進化しました。

現在の GripMock は実用的なテストワークフローに焦点を当てています：

- ネイティブプロセス内アーキテクチャ（実行時のコード生成なし）
- 柔軟な記述子ソースとランタイム操作（ホットスタブ + descriptors API）
- 本番品質のテスト機能（ストリーミング、テンプレート、アップストリームモード、プラグイン、SDK、MCP）

アーキテクチャの詳細とベンチマーク方法については：[パフォーマンス比較](https://bavix.github.io/gripmock/guide/introduction/performance-comparison)

## 🖥️ Web インターフェース

![gripmock-ui](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-ui.gif)

`http://localhost:4771/` で Web ダッシュボードにアクセスし、スタブを視覚的に管理できます。

## 🚀 クイックスタート

### インストール

お好みのインストール方法を選択してください：

#### Homebrew（推奨）
```bash
brew tap gripmock/tap
brew install --cask gripmock
```

#### Shell Script
```bash
curl -s https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.sh | sh -s
```

#### PowerShell（Windows）
```powershell
irm https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.ps1 | iex
```

#### Docker
```bash
docker pull bavix/gripmock
```

プラグインビルドには、ペアのビルダーイメージを使用します：

```bash
docker pull bavix/gripmock:v3.17.2-builder
```

#### Go インストール
```bash
go install github.com/bavix/gripmock/v3@latest
```

### 基本的な使い方

**`.proto` ファイルで起動：**
```bash
gripmock service.proto
```

**静的スタブを追加：**
```bash
gripmock --stub stubs/ service.proto
```

**BSR（Buf Schema Registry）から直接 API をロード：**
```bash
gripmock --stub third_party/bsr/eliza buf.build/connectrpc/eliza
```

**稼働中の gRPC サーバーの reflection から API をロード：**
```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
```

オプション付き：
```bash
gripmock grpc://localhost:50051?timeout=10s
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
gripmock grpc://localhost:50051?bearer=<token>
```

**reflection 上のアップストリームモード（試験運用）：**
```bash
# GripMock を通じた純粋なリバースプロキシ
gripmock grpc+proxy://localhost:50051

# ローカルスタブ優先、マッチャーミス時にアップストリームにフォールバック
gripmock grpc+replay://localhost:50051

# リプレイ + アップストリームのミスを GripMock スタブに自動記録
gripmock grpc+capture://localhost:50051
```

プライベート BSR モジュール：
```bash
BSR_BUF_TOKEN=<token> gripmock --stub stubs/ buf.build/acme/private-api
```

セルフホスト BSR：
```bash
BSR_SELF_BASE_URL=https://bsr.company.local \
BSR_SELF_TOKEN=<token> \
gripmock --stub stubs/ bsr.company.local/team/payments
```

**Docker を使用：**
```bash
docker run -p 4770:4770 -p 4771:4771 -p 4769:4769 \
  -v $(pwd)/stubs:/stubs \
  -v $(pwd)/proto:/proto \
  bavix/gripmock --stub=/stubs /proto/service.proto
```

- **ポート 4770**: gRPC サーバー
- **ポート 4771**: Web UI および REST API
- **ポート 4769**: ゲートウェイ（gRPC-web / ConnectRPC）

### 可観測性（v3.10.0）

```bash
OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
gripmock --stub stubs/ service.proto
```

- `GET /metrics` は常に利用可能
- トレーシングエクスポートは `OTEL_ENABLED=true` の場合のみ有効

## 🤖 GitHub Actions（CI/CD）

公式アクション [`bavix/gripmock-action`](https://github.com/bavix/gripmock-action) を使用して CI パイプラインで GripMock を実行します。

```yaml
name: test

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5

      - name: Start GripMock
        uses: bavix/gripmock-action@v1
        with:
          source: proto/service.proto
          stub: stubs

      - name: Run tests
        run: go test ./...
```

アクションの機能：

- GitHub Releases から GripMock をダウンロード（`latest` または固定 `version`）
- バックグラウンドで GripMock を起動し、準備完了を待機（`/api/health/readiness`）
- アドレスを outputs（`grpc-addr`、`http-addr`）として公開
- ポストステップで GripMock を自動停止

詳細な例と完全な inputs/outputs については：[GitHub Actions ガイド](https://bavix.github.io/gripmock/guide/ci-cd/github-actions)

## 📖 例

[`examples`](https://github.com/bavix/gripmock/tree/master/examples) フォルダの包括的な例を参照してください：

- **ストリーミング** - サーバー、クライアント、双方向ストリーミング
- **ファイルアップロード** - チャンクファイルアップロードのテスト
- **リアルタイムチャット** - 双方向通信
- **データフィード** - 継続的なデータストリーミング
- **認証** - ヘッダーベースの認証テスト
- **パフォーマンス** - 高スループットシナリオ

### Greeter：動的スタブデモ

スタブ（汎用）：

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
# examples/projects/greeter/stub_say_hello.yaml
- service: helloworld.Greeter
  method: SayHello
  input:
    matches:
      name: ".+"
  output:
    data:
      message: "Hello, {{.Request.name}}!"
```

注意事項：
- 動的テンプレートは `output` にのみ配置（例：`data`、`headers`、`stream`）
- `input` のマッチングは静的に保つ（`equals`/`contains`/`matches` に `{{ ... }}` を使用しない）

```bash
# サーバーを起動
go run main.go examples/projects/greeter/service.proto --stub examples/projects/greeter

# grpcurl で呼び出し
grpcurl -plaintext -d '{"name":"Alex"}' localhost:4770 helloworld.Greeter/SayHello
```

期待されるレスポンス：

```json
{
  "message": "Hello, Alex!"
}
```

## 🔧 スタビング

### 基本的なスタブの例

```yaml
service: Greeter
method: SayHello
input:
  equals:
    name: "gripmock"
output:
  data:
    message: "Hello GripMock!"
```

### 高度な機能

**優先度システム：**
```yaml
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      id: "admin"
  output:
    data:
      role: "administrator"

- service: UserService
  method: GetUser
  priority: 1
  input:
    contains:
      id: "user"
  output:
    data:
      role: "user"
```

**ストリーミングサポート：**
```yaml
service: TrackService
method: StreamData
input:
  equals:
    sensor_id: "GPS001"
output:
  stream:
    - position: {"lat": 40.7128, "lng": -74.0060}
      timestamp: "2024-01-01T12:00:00Z"
    - position: {"lat": 40.7130, "lng": -74.0062}
      timestamp: "2024-01-01T12:00:05Z"
```

### 動的テンプレート

GripMock は `output` セクションで Go の `text/template` 構文を使用した動的テンプレートをサポートしています。

- リクエストフィールドへのアクセス：`{{.Request.field}}`
- ヘッダーへのアクセス：`{{.Headers.header_name}}`
- クライアントストリーミングコンテキスト：`{{.Requests}}`、`{{len .Requests}}`、`{{(index .Requests 0).field}}`
- 双方向ストリーミング：`{{.MessageIndex}}` で現在のメッセージインデックス（0 始まり）
- 数学ヘルパー：`sum`、`avg`、`mul`、`min`、`max`、`add`、`sub`、`div`
- ユーティリティ：`json`、`split`、`join`、`upper`、`lower`、`title`、`sprintf`、`int`、`int64`、`float`、`round`、`floor`、`ceil`
- 組み込み faker：`faker.Person.*`、`faker.Contact.*`、`faker.Geo.*`、`faker.Network.*`、`faker.Identity.*`

重要なルール：
- `input.equals`、`input.contains`、`input.matches` 内で動的テンプレートを使用しないでください（マッチングは静的にする必要があります）
- サーバーストリーミングで `output.stream` と `output.error`/`output.code` の両方が設定されている場合、メッセージが最初に送信され、その後エラーが返されます。`output.stream` が空の場合は、エラーが即座に返されます

**ヘッダーマッチング：**
```yaml
service: AuthService
method: ValidateToken
headers:
  equals:
    authorization: "Bearer valid-token"
input:
  equals:
    token: "abc123"
output:
  data:
    valid: true
    user_id: "user123"
```

## 🔍 入力マッチング

GripMock は 4 つの強力なマッチング戦略をサポートしています：

### 1. 完全一致（`equals`）
```yaml
input:
  equals:
    name: "gripmock"
    age: 25
    active: true
```

### 2. 部分一致（`contains`）
```yaml
input:
  contains:
    name: "grip"
```

### 3. 正規表現一致（`matches`）
```yaml
input:
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
```

### 4. Glob 一致（`glob`）
```yaml
input:
  glob:
    filename: "*.txt"
    path: "/usr/local/*"
```

## 🛠️ API

### REST API エンドポイント

- `GET /api/stubs` - すべてのスタブを一覧表示
- `POST /api/descriptors` - 実行時に protobuf 記述子セット（`FileDescriptorSet`）をロード
- `POST /api/stubs` - 新しいスタブを追加
- `POST /api/stubs/search` - 一致するスタブを検索
- `DELETE /api/stubs` - すべてのスタブをクリア
- `GET /api/health/liveness` - ヘルスチェック
- `GET /api/health/readiness` - 準備完了チェック

### API 使用例

```bash
# スタブを追加
curl -X POST http://localhost:4771/api/stubs \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "input": {"equals": {"name": "world"}},
    "output": {"data": {"message": "Hello World!"}}
  }'

# 一致するスタブを検索
curl -X POST http://localhost:4771/api/stubs/search \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": {"name": "world"}
  }'
```

## 📋 JSON Schema サポート

スタブファイルにスキーマ検証を追加して IDE サポートを有効にします：

**JSON ファイル：**
```json
{
  "$schema": "https://bavix.github.io/gripmock/schema/stub.json",
  "service": "MyService",
  "method": "MyMethod"
}
```

**YAML ファイル：**
```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
service: MyService
method: MyMethod
```

## 🌐 BSR 統合

GripMock は Buf Schema Registry との簡略化された統合をサポートしています：

### 設定

```bash
# パブリック BSR（デフォルト）
BSR_BUF_BASE_URL=https://buf.build
BSR_BUF_TOKEN=<token>

# セルフホスト BSR
BSR_SELF_BASE_URL=https://bsr.company.local
BSR_SELF_TOKEN=<token>
```

### 使用方法

```bash
# パブリックモジュール
gripmock buf.build/connectrpc/eliza

# セルフホストモジュール
gripmock bsr.company.local/team/payments:main

# スタブ付き
gripmock --stub stubs/ bsr.company.local/team/payments
```

### ルーティング

GripMock は自動的にモジュールをルーティングします：
- `buf.build/owner/repo` → Buf プロファイルを使用
- `bsr.company.local/owner/repo` → Self プロファイルを使用

詳細については：[BSR ドキュメント](https://bavix.github.io/gripmock/guide/sources/bsr)

## 🔎 gRPC Reflection ソース

GripMock はエンドポイントスキームを使用した gRPC reflection からの記述子ロードをサポートしています：

- `grpc://host:port`（非セキュア）
- `grpcs://host:port`（TLS）

サポートされているクエリパラメータ：

- `timeout`（デフォルト `5s`）
- `bearer`（Authorization トークン）
- `serverName`（TLS SNI オーバーライド）

例：

```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
```

完全なガイド：[gRPC Reflection ソース](https://bavix.github.io/gripmock/guide/sources/grpc-reflection)

## 🔁 Upstream Modes（試験運用）

⚠️ **試験運用機能**：Upstream modes は予告なく変更される可能性があります。

Upstream modes は reflection ソース上で動作し、ランタイムの動作を定義します：

- `proxy` - 純粋なリバースプロキシ
- `replay` - ローカル優先 + アップストリームフォールバック
- `capture` - リプレイ + アップストリームミスの自動スタブ記録

モードガイド：

- [Upstream Modes 概要](https://bavix.github.io/gripmock/guide/modes)
- [Proxy モード](https://bavix.github.io/gripmock/guide/modes/proxy)
- [Replay モード](https://bavix.github.io/gripmock/guide/modes/replay)
- [Capture モード](https://bavix.github.io/gripmock/guide/modes/capture)

## 📊 ベンチマークチャート

![イメージサイズベンチマーク](docs/public/bench/image-size.svg)
![起動準備完了ベンチマーク](docs/public/bench/startup-ready.svg)
![レイテンシ百分位ベンチマーク](docs/public/bench/latency-percentiles.svg)
![スループットベンチマーク](docs/public/bench/throughput-rps.svg)

## 🔗 便利なリソース

- 📖 **[ドキュメント](https://bavix.github.io/gripmock)** - 完全なガイドと例
- 🧪 **[Testcontainers を使用した gRPC テスト](https://medium.com/skyro-tech/testing-grpc-client-with-mock-server-and-testcontainers-f51cb8a6be9a)** - 著者 [@AndrewIISM](https://github.com/AndrewIISM)
- 📋 **[JSON Schema](https://bavix.github.io/gripmock/schema/stub.json)** - スタブ検証スキーマ
- 🔗 **[OpenAPI](https://bavix.github.io/gripmock-openapi/)** - REST API ドキュメント

## 🤝 貢献

貢献を歓迎します！詳細は[コントリビューティングガイド](CONTRIBUTING.md)（日本語版：[CONTRIBUTING.ja-JP.md](CONTRIBUTING.ja-JP.md)）をご覧ください。

## 📄 ライセンス

このプロジェクトは **MIT ライセンス**の下でライセンスされています。詳細は [LICENSE](LICENSE) ファイルを参照してください。

---

**GripMock コミュニティから ❤️ を込めて**
