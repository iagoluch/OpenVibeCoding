# 架构

[Português](../pt-BR/arquitetura.md) · [English](../architecture.md) · [Español](../es/arquitectura.md) · [简体中文](architecture.md)

## 模块

| 模块 | 职责 |
| --- | --- |
| `cmd/ovc` | CLI、首次配置、隐藏式凭据输入、runtime 生命周期、诊断/导出/浏览器设置 |
| `internal/config` | 模型/服务注册表、协议选择、请求参数、权限和安全凭据配置 |
| `internal/credentials` | 操作系统密钥链与显式的受限文件回退 |
| `internal/provider` | OpenAI 兼容、原生 Anthropic 与原生 Gemini 流式适配器；工具、图像、用量、错误/重试 |
| `internal/store` | SQLite 迁移、会话、原始交互、审计记录、摘要、用量与导出 |
| `internal/runtime` | 认证的本地 IPC、OS 所有权锁、daemon 复用、按会话/工作区协调与审批路由 |
| `internal/agent` | 有界模型/工具循环、规划指令、harness 策略、审批、增量上下文与证据检索 |
| `internal/workspace` | 受限文件工具、ignore 规则、搜索、Git 检查、可审查写入/移动/删除、shell 和 Linux sandbox |
| `internal/automation` | 内置 Playwright worker、原生平台后端、截图、能力检查和外部适配器协议 |
| `internal/tui` | Bubble Tea 流式记录、输入/历史、模型/harness 切换、权限、审查与会话导航 |
| `scripts` | 构建、压缩包/校验和生成、二进制下载/安装、升级与卸载 |

## 任务生命周期

客户端连接现有 runtime 或启动一个新 runtime，然后选择持久会话、服务提供商/模型/harness。runtime 验证选择，并在本轮期间协调会话和工作区。原始消息会在发送请求之前保存；上下文准备逐步总结较早的交互而不删除它们。服务提供商的工具调用会经过组装和验证。读取工具在其边界内执行；需要审批的操作会流式发送给负责的客户端，并在收到答复前暂停。执行前会记录审计信息，观察到的输出和用量随后被持久化。取消操作会传播到服务提供商、进程和 worker。

请求执行期间，TUI 仍可处理窗口尺寸变化、滚动、取消和审查。每轮工具工作流最多调用模型 16 次。浏览器/桌面步骤是实际 worker/API 操作；不支持的环境会返回错误。截图从持久化工具输出中移除，只在临时内存中附加到后续模型请求。

## 持久化与恢复

SQLite 是权威数据源。架构版本 1 引入会话/消息/审计，版本 2 添加摘要与用量。迁移具有事务性。WAL 和单一数据库连接可序列化服务内的写入；daemon 锁防止多个 runtime 争用。导出要求会话空闲，包含历史、审计、摘要和成本证据。中断的工具交互在恢复时会收到明确的未知结果；操作不会盲目重放。

## 平台边界与验证

内置 shell sandbox 是 Linux 上的 Bubblewrap。其他平台无法提供所需的强制隔离，因此会拒绝执行；主机模式必须显式选择。浏览器自动化在受支持主机上通过 Node/Playwright 运行。原生控制在 Linux 上使用 X11 工具，在 macOS 上使用 AppleScript/CoreGraphics，在 Windows 上使用 PowerShell/Win32，也可配置外部 JSON 适配器。原生桌面 API 和 OS 密钥链行为需要在目标主机上通过用户授权进行验证。

仓库包含单元、集成和安全测试，以及针对原生操作系统执行的 CI。Linux 本地测试验证实际浏览器、sandbox 和 X11 操作。交叉编译不能证明 macOS/Windows 的原生 runtime 行为。真实服务提供商验收需要有效凭据。请参阅[测试与故障排查](testing.md)及[技术参考](sources.md)。
