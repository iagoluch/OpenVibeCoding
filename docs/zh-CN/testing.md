# 验证与故障排查

[Português](../pt-BR/testes.md) · [English](../testing.md) · [Español](../es/pruebas.md) · [简体中文](testing.md)

## 自动化检查

```sh
go test ./...
go test -race ./...
go vet ./...
bash scripts/package.sh
```

覆盖范围包括原生/兼容流式协议、工具片段/用量/错误/重定向、endpoint 验证、凭据回退权限与删除、SQLite 持久化/恢复、增量摘要保留与防重复、写入前远程审批、runtime 认证/重复实例防护/会话排他/导出、ignore/路径/符号链接边界、过期 diff、创建/移动/删除、命令退出状态/取消、slash 命令验证和 UI 关闭。

环境集成测试需要显式的 fixture：

```sh
# 先安装浏览器前置依赖：
ovc --setup-browser
OPENVIBECODING_TEST_PLAYWRIGHT=/应用数据路径/qa/node_modules/playwright go test ./internal/automation -run TestRealBrowserWorkflow
OPENVIBECODING_TEST_SANDBOX=1 go test ./internal/workspace -run TestSandboxBoundary
# 只能指向隔离的测试桌面，绝不要使用日常工作会话：
DISPLAY=:93 OPENVIBECODING_TEST_DESKTOP=1 go test ./internal/automation -run TestRealDesktop
```

真实浏览器测试会填写合成表单、点击按钮、检查可见结果、截屏并验证未授权来源会被拒绝。真实桌面测试会移动/点击/滚动、观察指针位置、截图，并在合成 xterm 进程中键入文本，再用 fixture 文件验证输入。Xvfb 需要 `-noreset`，避免客户端断开时服务器重置指针状态。

sandbox 测试会验证对所选项目的访问、对外部 fixture 文件的拒绝、对 `.env` 的屏蔽，以及工作区内实际写入。主机 namespace 策略可能阻止 Bubblewrap；此时应记录为不可用，而不是静默回退到主机执行。

## 本地证据与限制

开发期间，所有包均通过单元/集成测试和竞态检测。Linux 下真实的浏览器、Bubblewrap 和 X11 测试通过。两个终端客户端曾连接同一 daemon；关闭其中一个不会影响另一个。协议 fixture 覆盖 OpenAI、Anthropic 和 Gemini。打包流程为六个目标编译了 `ovc` 启动器。

最初一次 X11 fixture 测试失败，因为 Xvfb 在客户端断开后重置；使用 `-noreset` 修正后，真实鼠标/键盘测试通过。Playwright 将 Amazon Linux 识别为不支持的发行版，并下载 Ubuntu fallback；Chromium 工作流已在该环境中成功运行。

不声称已执行收费的外部服务请求，也不声称已在 macOS/Windows 上完成原生 GUI/密钥链执行。服务提供商适配器使用合成 HTTP/SSE 响应进行了测试。CI 配置中存在原生测试，不代表某次远程执行已经通过；请查看记录的执行结果。

## 使用真实服务提供商验收

1. 创建包含 README 和合成数据的一次性项目。用 `ovc --credential <provider>` 保存凭据，不要将凭据粘贴到项目或会话中。
2. 检查 `--doctor`，使用 `--workspace` 启动，然后选择 `/model provider/model`。
3. 要求检查 README 并创建 `hello.txt`。先拒绝一次，确认文件不存在；再批准新的提案并检查实际文件。
4. 批准一个无害测试命令，确认输出和故意返回的非零退出码均被准确报告。可用时使用隔离；主机模式需要显式选择。
5. 退出并恢复会话，检查对话记录、`/changes` 和保存的 `/cost`。
6. 用 Anthropic/Gemini 和兼容 endpoint 重复测试，确认恢复会话或切换模型后工具响应仍连贯。
7. 对支持图像的模型，启用浏览器/桌面，批准截图并确认模型引用了观察到的证据。恢复会话后，确认它会重新截图，而不会编造旧图像细节。

## 故障排查

- **Runtime 启动：** 在终端运行 `ovc --runtime serve` 查看设置错误。`--runtime status` 检查兼容认证。不要删除活动锁文件；所有权由 OS 锁决定。发生崩溃后，正常重新连接。
- **凭据/配置变更：** 停止空闲 runtime 并重新启动。模型/harness/授权的 slash 命令更改从下一轮生效。
- **密钥链不可用：** 解锁/配置操作系统凭据服务，或显式选择 `--credential-file`。Linux 需要 Secret Service 会话。不要降低文件权限。
- **服务提供商 HTTP/协议错误：** 检查账号、endpoint、模型和协议；错误正文会有意省略。对不支持 `stream_usage` 的兼容服务将其关闭。仅支持文本的模型应设置 `tools:false`，且不适合截图工作流。
- **上下文准备失败：** 原始消息仍已保存。审查/导出会话；若当前任务超出预算，请开始更小的任务。无效摘要会被拒绝。
- **浏览器不可用：** 运行 `--setup-browser`；检查 Node/npm 和 Chromium 系统库。下载需要网络；允许来源必须包含必要子资源。
- **桌面不可用：** 检查 `/capabilities`、DISPLAY/OS 会话和用户授予的权限。X11 需要 xdotool/ImageMagick/xclip。Windows 需要交互式桌面且现有 PowerShell 策略允许运行固定脚本。macOS 需要辅助功能/屏幕录制权限。
- **取消：** Esc/Ctrl+C 会取消动作/worker。之后检查目标；已完成的点击、提交、输入和外部效果无法靠取消撤回。

## 备份

导出空闲会话：`ovc --export ID > session.json`。创建完整可恢复备份时，先停止 runtime，再复制整个应用数据目录（包含仍存在的 SQLite 辅助文件）。先还原到单独的 `OPENVIBECODING_HOME` 中并检查。不要单独复制正在使用的 SQLite 数据库文件。
