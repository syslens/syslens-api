# SysLens 贡献指南

感谢您对 SysLens 项目的关注！我们欢迎任何形式的贡献，包括但不限于代码贡献、文档改进、问题报告和功能建议。

## 目录

- [贡献流程](#贡献流程)
- [开发环境设置](#开发环境设置)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [分支管理](#分支管理)
- [测试要求](#测试要求)
- [文档要求](#文档要求)
- [问题报告](#问题报告)
- [功能建议](#功能建议)
- [行为准则](#行为准则)

## 贡献流程

1. **Fork 项目仓库**：在 GitHub 上 Fork 项目仓库到您自己的账号下
2. **克隆您的 Fork**：`git clone https://github.com/您的用户名/syslens-api.git`
3. **添加上游仓库**：`git remote add upstream https://github.com/syslens/syslens-api.git`
4. **创建功能分支**：`git checkout -b feature/your-feature-name`
5. **进行开发**：按照代码规范进行开发
6. **提交更改**：按照提交规范提交您的更改
7. **推送到您的 Fork**：`git push origin feature/your-feature-name`
8. **创建 Pull Request**：在 GitHub 上创建 Pull Request，将您的分支合并到上游仓库的 `develop` 分支

## 开发环境设置

### 系统要求

- Go 1.18 或更高版本
- Git
- Make
- Docker 和 Docker Compose（可选，用于容器化开发）

### 设置步骤

1. **安装依赖**：

```bash
make deps
```

2. **构建项目**：

```bash
make build-all
```

3. **运行测试**：

```bash
make test
```

4. **启动开发环境**：

```bash
# 启动主控端
make run-server

# 启动节点端
make run-agent
```

## 代码规范

请遵循项目中的 [GOLANG_STANDARDS.md](GOLANG_STANDARDS.md) 文件定义的代码规范。主要包括：

- 代码风格和格式化
- 命名规范
- 项目结构
- 错误处理
- 日志规范
- 测试规范
- 文档规范
- 性能优化
- 安全规范

## 提交规范

我们使用语义化的提交消息格式，提交消息应遵循以下格式：

```
<类型>: <描述>

[可选的详细描述]

[可选的关闭问题引用]
```

### 提交类型

- **Fix**：修复代码中的错误、缺陷或漏洞
- **Feature**：添加新的功能、模块或文件
- **Refactor**：对现有代码进行重构，改善结构、性能或可读性
- **Optimize**：优化代码、算法或性能方面的改进
- **Documentation**：更新或添加文档内容

### 提交示例

```
Fix: 修复登录页面的样式问题
Feature: 新增节点分组功能
Refactor: 重构数据上报模块
Optimize: 优化数据库查询性能
Documentation: 更新API文档
```

## 分支管理

我们采用 Git Flow 分支模型：

- **main**：主分支，包含稳定版本
- **develop**：开发分支，包含最新开发代码
- **feature/\***：功能分支，用于开发新功能
- **bugfix/\***：修复分支，用于修复 bug
- **release/\***：发布分支，用于准备发布

### 分支命名规范

- 功能分支：`feature/功能名称`
- 修复分支：`bugfix/问题描述`
- 发布分支：`release/版本号`

## 测试要求

- 所有新功能和 bug 修复必须包含测试
- 单元测试覆盖率应达到 80% 以上
- 提交前运行所有测试确保通过
- 遵循 [GOLANG_STANDARDS.md](GOLANG_STANDARDS.md) 中的测试规范

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定包的测试
go test -v ./internal/agent/...

# 运行测试并生成覆盖率报告
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 文档要求

- 所有新功能必须包含文档
- 更新现有功能时，同步更新相关文档
- 文档应放在 `docs/` 目录下
- 代码中的注释应遵循 godoc 格式

### 文档类型

- **架构文档**：描述系统架构和设计决策
- **API 文档**：描述 API 接口和使用方法
- **部署文档**：描述如何部署和配置系统
- **开发文档**：描述如何设置开发环境和参与开发

## 问题报告

如果您发现 bug 或有改进建议，请在 GitHub 上创建 Issue。Issue 应包含：

- 问题的详细描述
- 复现步骤
- 期望行为
- 实际行为
- 环境信息（操作系统、Go 版本等）
- 相关日志或截图

## 功能建议

如果您有功能建议，请在 GitHub 上创建 Issue，并标记为 "enhancement"。功能建议应包含：

- 功能的详细描述
- 使用场景和需求
- 可能的实现方案
- 预期效果

## 行为准则

我们致力于为每个人提供一个友好、包容和欢迎的环境。请遵循以下行为准则：

- 尊重所有参与者
- 接受建设性的批评
- 关注最有利于项目的事情
- 展示同理心

违反行为准则的行为将不被容忍，可能会导致被禁止参与项目。

## 许可证

通过向 SysLens 项目贡献代码，您同意您的贡献将根据项目的 Apache 2.0 许可证进行许可。

## 联系方式

如果您有任何问题或需要帮助，请通过以下方式联系我们：

- 在 GitHub 上创建 Issue
- 发送邮件至 [项目维护者邮箱]

感谢您的贡献！
