# OpenVibeCoding 用户指南

[Português](../pt-BR/guia.md) · [English](../guide.md) · [Español](../es/guia.md) · [简体中文](guide.md)

OpenVibeCoding（OVC）是用 Go 和 Bubble Tea 编写的终端编程智能体。它可以检查项目、提出可审查的文件修改、运行经批准的命令、保留任务上下文、使用 Chromium 测试网站，并在平台支持时执行经过批准的桌面操作。

## 安装与启动

从 [Releases](https://github.com/iagoluch/OpenVibeCoding/releases/latest) 下载二进制包，或使用安装脚本。脚本会下载适合操作系统和 CPU 的压缩包，并验证 SHA-256。

Linux/macOS：
```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
# 如有需要，将 ~/.local/bin 加入 PATH：
export PATH="$HOME/.local/bin:$PATH"
ovc --workspace /项目/路径
```

Windows（PowerShell）：运行 `./scripts/install.ps1 -AddToPath`，然后重新打开终端。如果执行策略阻止脚本，可运行 `powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath`。要固定版本，可在 PowerShell 中使用 `-Version v0.1.0`，或在 shell 中设置 `OPENVIBECODING_VERSION=v0.1.0`。

从源码编译需要 **Go 1.26.2 或更高版本**：运行 `go build -o ovc ./cmd/ovc`，再运行 `bash scripts/install.sh ./ovc`（Windows：`./scripts/install.ps1 -Binary ./ovc.exe -AddToPath`）。`bash scripts/package.sh` 会为 Linux/macOS/Windows 的 amd64/arm64 生成压缩包和 `SHA256SUMS`；打包还需要 Python 3。使用发布版二进制不需要 Go。浏览器 QA 还需要 Node 和 Chromium。

## 首次配置与凭据

首次运行时会创建默认配置。运行 `ovc --setup` 可启动引导程序：它会检测 Ollama 和 LM Studio 服务，列出免费套餐提供商及其密钥申请页面，可导入已在 OpenCode 中配置的凭据，将密钥存入操作系统密钥链，并在启用前发起一次真实请求。如果首选模型已停用、触发频率限制或正忙，引导程序会尝试该提供商的下一个模型。

已知提供商时，可通过终端的隐藏输入录入凭据：
```sh
ovc --credential openai
ovc --credential anthropic
ovc --credential gemini
```

默认凭据后端是操作系统密钥存储：macOS Keychain、Windows Credential Manager 或 Linux Secret Service。如果不可用，必须显式选择 `--credential-file` 使用权限受限的本地文件；程序不会静默地退回明文存储。`OPENAI_API_KEY`、`ANTHROPIC_API_KEY` 和 `GEMINI_API_KEY` 等环境变量会覆盖已保存的凭据。不要在对话或 endpoint URL 中粘贴密钥。

使用 `ovc --doctor` 检查配置和能力，不会输出密钥。如果 runtime 运行时凭据或配置发生变化，请使用 `ovc --runtime stop` 停止空闲 runtime，然后重新连接。`/model` 和 `/harness` 命令会立即生效。

## 支持的服务提供商

| 适配器/提供商 | 支持能力 |
| --- | --- |
| OpenAI | 原生 Chat Completions 流式响应、工具调用和可选用量信息 |
| Anthropic Claude | 原生 Messages API 流式响应、工具调用与用量 |
| Google Gemini | 原生流式响应、函数调用/结果、图像证据与 thought signature 保留 |
| DeepSeek、Kimi、Qwen、Mistral | 兼容 OpenAI API 的 endpoint |
| 免费套餐预设 | OpenRouter（`:free` 模型）、Groq、Cerebras、NVIDIA NIM、Z.AI 与 Google AI Studio；可在 `ovc --setup` 中查看 |
| 本地/自定义服务 | 任意兼容 endpoint；允许 loopback HTTP，设置向导会检测 Ollama/LM Studio |

服务提供商注册表管理模型、上下文预算、工具能力、输出限制、温度和价格。请向服务提供商确认模型可用性和区域 endpoint。确认免费的路线显示为 US$ 0；付费平台显示每百万 token 的公开价格；无法可靠确认的价格会标为未知而不会猜测。请求具有超时、响应上限、取消机制、HTTP 429/5xx 重试和脱敏错误。为保护凭据，程序拒绝重定向；HTTP 仅允许用于 loopback。

预设随每个版本更新。`config.json` 只保存你自己修改过的项目；未修改的条目会自动获取新的模型 ID、上下文和价格。

## 开发工作流

1. 要求智能体检查项目并制定修改计划。文件工具会遵守受保护名称和 `.gitignore` 规则、限制读取/搜索范围，并拒绝工作区之外的路径。
2. 审查统一 diff。**Ctrl+Y** 批准当前操作，**Ctrl+N** 拒绝。支持替换、创建、移动和删除文件。文件已发生变化时，旧提案会失效。
3. 批准测试/build 命令。在受支持的 Linux 系统上，命令默认在 Bubblewrap 中运行，只能访问工作区，系统运行时文件为只读，且默认无网络。如果无法强制隔离，程序会拒绝执行。
4. 检查实际 stdout、stderr、退出码和 `/changes`。之后可以继续已保存的会话。

`--allow-unsandboxed-shell` 是显式选择的主机命令模式，适用于所有平台。每条命令仍需批准，但会以当前用户权限执行，并能访问主机文件/网络；这不等于操作系统沙箱。请阅读[安全保证](security.md)。

### 控制与命令

Enter 发送；Ctrl+J 或 Alt+Enter 插入换行；Alt+上/下方向键回溯输入；Tab 补全 slash 命令；PgUp/PgDn 滚动。Esc 或 Ctrl+C 取消当前操作。空闲时按 Ctrl+C 退出客户端。文件审查按操作/文件进行并显示完整统一 diff，不支持按 hunk 选择。

| 命令 | 用途 |
| --- | --- |
| `/help`、`/status`、`/config` | 帮助、会话/runtime 状态和生效配置 |
| `/model [provider/model]`、`/provider [name]` | 选择已注册的服务提供商/模型 |
| `/harness [review\|inspect]` | `review` 允许经批准的操作；`inspect` 仅允许读取项目 |
| `/sessions`、`/resume <id>`、`/new` | 浏览持久会话 |
| `/rename <title>`、`/delete <other-session-id>` | 管理会话 |
| `/init`、`/undo`、`/doctor` | 提议 `AGENTS.md`、撤销上次应用的修改、诊断 runtime |
| `/context`、`/cost`、`/changes` | 上下文、已保存用量/成本、观察到的文件变化 |
| `/capabilities` | 检查自动化和 sandbox 依赖 |
| `/permissions [browser\|desktop\|clipboard\|network\|install on\|off]` | 查看或修改当前客户端权限 |
| `/clear`、`/exit` | 清除显示内容但不删除数据；退出 |

工作区根目录的 `AGENTS.md` 会加入每轮系统上下文（最多 16 KiB）。它描述项目约定，但属于参考信息，不是授权。运行 `/init` 会让智能体检查项目，并通过常规审查流程提出该文件。新会话默认使用第一条请求作为标题，可用 `/rename` 修改。

`/undo` 每次撤销一项已应用更改，并可跨重启使用。如果文件在操作后发生变化、路径被重新创建或当前仍有活跃任务，撤销会被拒绝。授予能力不会取消单项操作审批。网络和安装默认关闭；安装软件包需要单独授权。已批准的 shell 是通用编程环境，因此实际隔离由操作系统边界决定，而不仅是工具分类。

## 共享 runtime 与会话

第一个终端启动本地 runtime，后续终端会复用它。daemon 负责 SQLite、服务提供商配置、审计记录和任务协调。IPC 使用经过认证的 loopback HTTP、私有元数据文件中的随机 token，以及操作系统锁来防止重复 daemon。浏览器不能通过 cross-origin 请求调用它。

```sh
ovc --runtime start
ovc --runtime status
ovc --runtime stop
ovc --resume SESSION_ID --workspace /项目/路径
ovc --export SESSION_ID > session.json
```

每个客户端具有独立身份与审批记录。任务按会话**和工作区**串行执行。关闭一个客户端只取消它自己的请求，不影响其他客户端的任务。daemon 会保持运行，直到被显式停止；有活跃任务时不允许停止。操作系统终止进程时，会在关闭存储前取消任务。

SQLite 保存原始消息、工具结果、审批、任务生命周期、配置变更、摘要和 token/成本用量。数据库迁移具有事务性。中断后，工具操作结果会标记为不确定，不会自动重放修改。导出包含历史、摘要、用量和审计信息；导出前会话必须空闲。截图是临时数据，不会存入会话数据库。

上下文准备会逐步总结较早的完整对话，同时保留近期消息和原始历史。来源 ID 可用于通过 `recall` 检索原始证据。token 估算较保守；精确用量取决于服务提供商。摘要可能不完美：原始证据始终优先，历史审批不能授权新操作。当前任务若超出预算，会返回明确错误，而不是静默丢弃内容。

## 浏览器 QA

```sh
ovc --setup-browser
# 在 TUI 中：
/permissions browser on
```

设置程序会在应用 QA 目录中安装 Playwright 1.58.2 和 Chromium。审查并批准包含允许来源和步骤的工作流：`open`、`fill`、`click`、`press`、`select`、`expect_text`、`expect_url`、`screenshot`。每个工作流使用新的浏览器上下文；未授权来源、service worker、WebSocket 和下载都会被阻止。结果包括实际观察到的 DOM 文本、标题/URL、断言、控制台消息、页面错误、失败请求和可选截图。支持图像的模型可在当前任务中检查截图，之后会丢弃截图。请使用合成数据和专门的测试环境。

## 桌面操作

启用 `/permissions desktop on`；剪贴板需要另外启用 `/permissions clipboard on`。每项操作均需审查。支持的操作取决于平台后端，包括检查、截图、移动/点击指针、滚动、输入文字/按键、启动/聚焦/关闭应用，以及剪贴板访问。

| 平台 | 后端与边界 |
| --- | --- |
| Linux X11 | 使用 `xdotool`、ImageMagick `import` 截图和 `xclip` 剪贴板；需要活动的 `DISPLAY`。集成后端明确不支持 Wayland。 |
| macOS | AppleScript/System Events、`screencapture` 和通过 JavaScript for Automation 调用的 CoreGraphics。需要用户授予辅助功能/屏幕录制权限。 |
| Windows | PowerShell、Win32 API、Windows Forms 输入/剪贴板/截图。需要交互式桌面，不能控制 UAC/安全桌面，也不会绕过操作系统权限。 |

`automation.applications` 是启动/聚焦应用的允许列表，不是全局鼠标/键盘操作的进程隔离。批准输入之前，请检查屏幕上显示的目标。外部适配器通过 `automation.desktop_backend` 配置为可执行文件/参数数组。参阅[自动化协议](automation.md)。

## 配置、维护与验证

参阅 [config.example.json](../../config.example.json)。数据保存在平台的 `os.UserConfigDir()/openvibecoding` 下（例如 `~/.config`、macOS Application Support 或 Windows AppData）；`OPENVIBECODING_HOME` 可覆盖此目录。`OPENVIBECODING_PROVIDER` 和 `OPENVIBECODING_MODEL` 可覆盖默认设置。服务提供商密钥不会写入 JSON 配置或导出数据。

目录包含 `config.json`、SQLite 会话文件、`runtime.json`/`runtime.lock`、仅记录元数据的 `runtime.log`、可选 `credentials.json` 和 QA 依赖。Unix 文件/目录权限受限；Windows 使用用户配置文件 ACL。`--debug` 会记录诊断元数据，而不是请求正文或密钥。复制数据或升级二进制前，请先停止 runtime。

可通过新版安装脚本升级（`OPENVIBECODING_VERSION=v…` 或 PowerShell 的 `-Version v…`）；配置和会话会保留。停止 runtime 后，使用 `scripts/uninstall.sh` 或 `scripts/uninstall.ps1` 删除启动器。只有打算清除会话和凭据时，才单独删除数据。必要时将 `~/.local/bin` 加入 shell 启动配置。PowerShell 安装不会更改执行策略，也不会请求管理员权限；若策略阻止脚本，可运行 `powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath`。

验证请参阅[测试与故障排查](testing.md)、[架构](architecture.md)和[技术参考](sources.md)。本地检查涵盖 Linux sandbox、浏览器/桌面行为、服务提供商协议 fixture、runtime 并发/认证/审批、持久化和上下文恢复。macOS/Windows 原生执行、相应系统的密钥链集成和收费的真实服务请求仍需要这些平台/账号。交叉编译不能证明原生行为。
