# 为 GripMock 做贡献

**语言版本：** [English](CONTRIBUTING.md) | 简体中文 | [日本語](CONTRIBUTING.ja-JP.md) | [Deutsch](CONTRIBUTING.de.md) | [Español](CONTRIBUTING.es.md)

> 提示：本页面由机器翻译生成，内容可能存在不准确或不完整之处。请以英文原文 [`CONTRIBUTING.md`](CONTRIBUTING.md) 为准。

## 开始

1. **Fork 仓库** 并在本地克隆你的 fork
2. **配置开发环境**：
   - 安装用于集成测试的 [grpctestify](https://github.com/gripmock/grpctestify-rust)（安装说明见 [grpctestify documentation](https://gripmock.github.io/grpctestify-rust/)）
   - 确保已安装并正确配置 Go

### ConnectRPC 测试

面向 ConnectRPC 服务的 **HTTP 客户端测试**（`.http` 文件）位于 `examples/projects/*/connectrpc-tests/`。

**使用 httpyac 运行：**
```bash
npx httpyac run examples/projects/greeter/connectrpc-tests/ --all
```

你也可以在 JetBrains IDE（GoLand、IntelliJ）中打开 `.http` 文件，点击每个请求旁边的运行图标。

## 测试要求

### 1. gRPC 服务变更必须包含集成测试

任何改变 gRPC 服务行为的变更，都需要用 grpctestify 编写 `.gctf` 格式的集成测试。

集成测试位于 `examples/` 目录。`.gctf` 文件示例：

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**测试应放置在：**
- 集成测试：`examples/projects/*/case_*.gctf`
- 单元测试：`internal/app/*_internal_test.go`

### 2. 每个 PR 都包含测试

bug 修复与新功能都需要。

### 3. 在本地运行测试

```bash
make test    # 单元测试
make lint    # Linter
```

集成测试需要在另一个终端中运行服务：

```bash
go run main.go examples -s examples
grpctestify examples/
```

## 向后兼容

所有更改都必须保持向后兼容，除非破坏性变更已通过 issue 讨论并获得批准。

### 破坏性变更流程

如果你需要引入破坏性变更：

1. **先创建 Issue**：提交包含以下内容的详细提案：
   - 你要解决的问题描述
   - 为什么必须引入破坏性变更
   - 面向现有用户的迁移方案

2. **等待批准**：未经维护者讨论并批准，不要实现破坏性变更

3. **提供迁移指南**：若获批准，请在 PR 中包含清晰的迁移说明

## Pull Request 流程

### 提交前

- [ ] 本地所有测试通过
- [ ] 代码遵循项目风格指南（`make lint`）
- [ ] 如有需要，文档已更新
- [ ] 你的分支已与 `master` 保持最新

### PR 描述

创建 PR 时，请包含：
- 变更说明
- 变更类型（bug 修复、新功能等）
- 测试信息（单元测试；若涉及 gRPC 服务变更则包含集成测试）
- 向后兼容状态
- 相关 issues

## 代码风格

- 遵循标准 Go 格式：`gofmt` 和 `goimports`
- 运行 linter：`make lint`
- 使用有意义的变量名和函数名
- 为导出函数与类型添加注释
- 将新代码放在 `internal/` 下合适的包中

## 文档

在以下情况请更新文档：
- 添加新功能
- 变更现有行为
- 修复会影响用户工作流的 bug

文档位置：
- 用户文档：`docs/guide/`
- 示例：`examples/` 目录
- 主 README：`README.md`

## 有问题？

先查看现有的 issues 与 discussions，然后使用 `question` 标签创建新 issue。

- [项目文档](https://bavix.github.io/gripmock/)
- [grpctestify 文档](https://gripmock.github.io/grpctestify-rust/)
