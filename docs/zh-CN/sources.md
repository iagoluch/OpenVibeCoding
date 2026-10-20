# 官方实现参考

查阅日期：2026-10-15。使用 Bubble Tea v2 的 View、textarea 和 viewport API 前，也通过 `go doc` 检查了已安装模块的 API。

- [Go 1.26 发布说明](https://go.dev/doc/go1.26)：工具链和平台支持。`os.Root` API 可防止部分路径逃逸，但不提供进程隔离。
- [Bubble Tea v2](https://github.com/charmbracelet/bubbletea)：事件/update/view 架构和 v2 模块路径。[Bubbles](https://pkg.go.dev/charm.land/bubbles/v2) 提供 textarea 和 viewport。
- [OpenAI Chat Completions create](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create)：流式 delta、工具调用和可选用量事件。
- [modernc SQLite](https://pkg.go.dev/modernc.org/sqlite)：database/sql 驱动和纯 Go 构建路径。
- [Playwright 入门](https://playwright.dev/docs/intro)：浏览器 QA 和支持的操作系统要求。

## 补充实现参考

- [Anthropic Messages streaming](https://platform.claude.com/docs/en/api/messages-streaming)：内容块开始/delta、部分工具 JSON、用量和消息结束事件。
- [Gemini generate/stream content](https://ai.google.dev/api/generate-content)：原生 parts、函数调用/结果及 thought signature。
- [Go keyring 库](https://github.com/zalando/go-keyring)：macOS/Windows/Secret Service API、Get/Set/Delete 和系统前置条件。
- [Bubblewrap](https://github.com/containers/bubblewrap)：命名空间、只读挂载、策略责任与限制。
- [Playwright Browser API](https://playwright.dev/docs/api/class-browser)：browser/context 生命周期。
- [xdotool](https://github.com/jordansissel/xdotool)：X11 输入/窗口 API 及 Wayland 限制。
- [Apple UI scripting](https://developer.apple.com/library/archive/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/AutomatetheUserInterface.html) 与 [CoreGraphics CGEvent](https://developer.apple.com/documentation/coregraphics/cgevent)：用户授权的辅助功能、事件、指针/滚动字段和事件发送。
- [Microsoft 桌面输入](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput)：桌面输入限制与完整性级别边界。
