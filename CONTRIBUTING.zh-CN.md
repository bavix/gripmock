# 为 GripMock 做贡献

**语言版本：** [English](CONTRIBUTING.md) | 简体中文

> 提示：本页面由机器翻译生成，内容可能存在不准确或不完整之处。请以英文原文 [`CONTRIBUTING.md`](CONTRIBUTING.md) 为准。

感谢你对 GripMock 贡献的兴趣！本文档提供了参与本项目贡献的指南。

## 开始

1. **Fork 仓库** 并在本地克隆你的 fork
2. **配置开发环境**：
   - 安装用于集成测试的 [grpctestify](https://github.com/gripmock/grpctestify-rust)（安装说明见 [grpctestify documentation](https://gripmock.github.io/grpctestify-rust/)）
   - 确保已安装并正确配置 Go

## 测试要求

### ⚠️ 关键规则

#### 1. gRPC 服务变更必须包含集成测试

**如果你改动、添加或修复了任何与 gRPC 服务功能相关的内容，你必须使用 `.gctf` 格式并通过 grpctestify 编写集成测试。**

集成测试位于 `examples/` 目录。`.gctf` 文件示例：

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**运行测试：**
```bash
make test              # 单元测试
grpctestify examples/  # 集成测试
make lint              # Linter
```

**测试应放置在：**
- 集成测试：`examples/projects/*/case_*.gctf`
- 单元测试：`internal/app/*_internal_test.go`

#### 2. 每个 PR 都必须包含测试

所有 Pull Request 都必须包含合适的测试，尤其是 bug 修复与新功能。

#### 3. 在本地运行测试

在提交 PR 前，请确保所有测试通过：

**对于使用 grpctestify 的集成测试：**
```bash
# 启动服务（在另一个终端）
go run main.go examples -s examples

# 运行集成测试
grpctestify examples/
```

**对于单元测试：**
```bash
make test
make lint
```

## 向后兼容

**除非已通过 issue 明确讨论并获批，否则所有更改都 MUST 保持向后兼容。**

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
- [ ] 你的分支已与主分支保持最新

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

- 查看现有 issues 与 discussions
- 使用 `question` 标签创建新 issue
- 查阅 [documentation](https://bavix.github.io/gripmock/)

## 附加资源

- [Project Documentation](https://bavix.github.io/gripmock/)
- [grpctestify Documentation](https://gripmock.github.io/grpctestify-rust/)

感谢你为 GripMock 做出贡献！🚀
