<p align="center">
  <img src="assets/ovc-brand-banner.jpg" alt="OpenVibeCoding 官方视觉：白色圆盘上的蓝色标志与 Byte 吉祥物，蓝黑色电影感背景，不含界面模型。" width="100%">
</p>

<p align="center">
  <strong>面向桌面应用体验打造的 AI 开发工作空间。</strong><br>
  审查修改、批准操作、验证结果——始终由你掌控。
</p>

<p align="center"><strong>集成桌面应用：尚未发布 · CLI/TUI：目前可用</strong></p>

<p align="center">
  <a href="https://github.com/iagoluch/OpenVibeCoding/releases/latest">⬇️ 体验当前版本（CLI/TUI）</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/iagoluch/OpenVibeCoding/stargazers">⭐ 为项目点亮 Star</a>
</p>

<p align="center">
  <a href="README.md">🇧🇷 Português (Brasil)</a> ·
  <a href="README.en.md">🇺🇸 English</a> ·
  <a href="README.es.md">🇪🇸 Español</a> ·
  <a href="README.zh-CN.md">🇨🇳 简体中文</a>
</p>

<p align="center">
  <a href="#安装">安装</a> ·
  <a href="docs/zh-CN/guide.md">使用指南</a> ·
  <a href="docs/zh-CN/security.md">安全</a> ·
  <a href="docs/zh-CN/architecture.md">架构</a>
</p>

<p align="center"><a href="https://github.com/iagoluch/OpenVibeCoding/stargazers">⭐ 如果它对你有帮助，请为项目点亮 Star</a></p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.2%2B-69CFFF?logo=go&logoColor=white" alt="Go 1.26.2 或更高版本">
  <img src="https://img.shields.io/badge/license-MIT-69CFFF" alt="MIT 许可证">
  <img src="https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-informational" alt="Linux、macOS 和 Windows">
</p>

## OpenVibeCoding 是什么？

**OpenVibeCoding（OVC）** 是一个开源 AI 开发工作空间，正在朝集成式桌面应用方向发展。目标是让开发者可以在一个统一的环境中与智能体协作、审查修改方案，并查看执行证据。

**当前状态说明：** 集成式桌面应用尚未发布二进制文件或安装程序。目前公开可用的版本是 **CLI/TUI**（命令行工具及终端界面）。下文的下载和安装说明仅适用于当前版本，并不代表桌面应用已经发布。

## 我们正在构建的桌面体验

计划中的桌面体验将把项目选择、与智能体对话、diff 审查、逐项审批、任务历史和结果验证整合到一个界面中。浏览器与桌面自动化取决于平台支持和用户明确授予的权限。

本节描述的是产品方向，并非已发布桌面 GUI 的功能说明。

## 当前 CLI/TUI 已具备的能力

| 理解项目 | 在控制下修改 | 验证结果 |
| --- | --- | --- |
| 读取文件、搜索代码、加载项目说明并检查 Git 状态 | 审查 diff，并逐项批准或拒绝文件操作 | 运行获准的命令，查看实际输出和退出码 |
| 保存会话与持久上下文 | 将文件工具限制在已配置的工作区内 | 使用 Playwright 浏览器 QA 和已配置的桌面集成 |

### 自由选择模型与服务提供商

OVC 不要求你绑定单一 AI 厂商。它内置 OpenAI、Anthropic/Claude 和 Google Gemini 的原生适配器，也支持兼容 OpenAI API 的端点，包括 DeepSeek、Kimi、Qwen、Mistral 和本地模型运行环境。

`ovc --setup` 会列出可用的服务提供商选项和免费层预设，检测 Ollama 与 LM Studio，并可导入已在 OpenCode 中配置的凭据。各服务的可用性、配额与条款可能变化。

### 持久会话与证据

对话、上下文摘要、审计记录和用量保存在本地 SQLite 数据库中。结果应反映真实发生的情况：被拒绝的命令不能被视为已批准，退出码非零的进程也不能被报告为成功。

### 需经批准的自动化

OVC 包含基于 Playwright 的浏览器 QA，以及依赖操作系统的桌面集成。只启用当前任务所需的能力。自动化操作须遵守对应工具的授权与审批规则。

## 体验当前公开版本（CLI/TUI）

### Linux 和 macOS

安装脚本会下载与操作系统和 CPU 架构匹配的发行包，并验证 SHA-256 校验和。使用已发布的二进制文件不需要安装 Go。

```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
ovc --setup
ovc --workspace /path/to/your/project
```

如果系统找不到 `ovc`，请将 `~/.local/bin` 添加到 `PATH`，然后重新打开终端。

### Windows（PowerShell）

在仓库目录中运行：

```powershell
./scripts/install.ps1 -AddToPath
```

安装完成后关闭并重新打开 PowerShell。如果执行策略阻止脚本：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath
```

也可以从 [Releases](https://github.com/iagoluch/OpenVibeCoding/releases/latest) 下载适用于相应系统和架构的预编译发行包。

### 首次运行

运行 `ovc --setup` 配置服务提供商、凭据和模型，然后指定项目目录：

```sh
ovc --workspace /path/to/your/project
```

使用 `ovc --doctor` 检查配置和能力；此命令不会输出密钥。

### 从源码构建

需要 Go **1.26.2 或更高版本**：

```sh
go build -o ovc ./cmd/ovc
```

在 Linux/macOS 上，可使用 `bash scripts/install.sh ./ovc` 安装生成的二进制文件。在 Windows 上，构建 `ovc.exe` 后运行 `./scripts/install.ps1 -Binary ./ovc.exe -AddToPath`。

## 典型开发流程

```text
❯ 检查这个 parser，提出小范围修复并验证。

理解       阅读实现、项目说明和相关测试。
审查       修改前查看提议的 diff。
批准       接受或拒绝这项操作。
验证       运行获准的命令并检查真实结果。
```

以上仅用于说明工作流，并非真实执行记录，也不保证每次修改都会成功。

常用命令：

```text
/model <provider/model>    /harness inspect
/sessions                  /resume <id>
/context                   /cost
/changes                   /permissions
/capabilities              /help
/init · /undo · /doctor
```

项目中的 `AGENTS.md` 指令会加入工作上下文。运行 `/help` 查看已安装版本支持的命令。

## 安全：了解执行边界

- 常规文件工具被限制在配置的工作区内，并对敏感路径应用排除规则。
- 文件修改会先展示供审查，应用前需要逐项批准。
- 在 Linux 上，默认使用 Bubblewrap 执行隔离；如果所需沙箱不可用，程序会拒绝执行，而不会静默退回到不受隔离的模式。
- **内置进程沙箱仅支持 Linux。** 在 macOS 和 Windows 上，需要隔离的命令会被拒绝。显式选项 `--allow-unsandboxed-shell` 会启用具有当前主机用户权限的命令执行，这并不等同于沙箱隔离。
- 桌面集成依赖各平台的权限与能力。交叉编译并不能证明程序在目标系统上的原生运行行为。

启用 shell 或自动化前，请阅读[安全与执行限制](docs/zh-CN/security.md)。外部操作、后台分离进程及其他应用中的操作不保证可以撤销。

## 文档

| 文档 | 内容 |
| --- | --- |
| [使用指南](docs/zh-CN/guide.md) | 安装、配置、服务提供商和命令 |
| [架构](docs/zh-CN/architecture.md) | 模块、代理生命周期、持久化和平台职责 |
| [安全](docs/zh-CN/security.md) | 审批、执行边界、凭据与限制 |
| [自动化](docs/zh-CN/automation.md) | 浏览器工作流、桌面操作与适配器 |
| [测试与故障排查](docs/zh-CN/testing.md) | 自动化检查、环境验证和问题排查 |
| 技术参考 | [技术参考](docs/zh-CN/sources.md) |

## 参与贡献

- **试用并点亮 Star：** [⭐ OpenVibeCoding on GitHub](https://github.com/iagoluch/OpenVibeCoding/stargazers)。
- **报告问题：** [创建 Issue](https://github.com/iagoluch/OpenVibeCoding/issues)。
- **阅读代码并提出改进：** 欢迎提交 Pull Request；贡献前请先阅读仓库中的相关指南。

OpenVibeCoding 使用 [MIT 许可证](LICENSE)发布。

---

<p align="center"><sub>OpenVibeCoding — 从意图到经过验证的修改，让人工审批始终处于工作流中。</sub></p>
