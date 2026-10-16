# Official implementation references

[Português](pt-BR/referencias.md) · [English](sources.md) · [Español](es/referencias.md) · [简体中文](zh-CN/sources.md)

Consulted on 2026-10-15. Installed module APIs were also checked with `go doc` before using Bubble Tea's v2 `View`, textarea and viewport APIs.

- [Go 1.26 release notes](https://go.dev/doc/go1.26): toolchain and platform support. Go's `os.Root` APIs provide traversal-resistant filesystem operations; they are not process isolation.
- [Bubble Tea v2](https://github.com/charmbracelet/bubbletea): event/update/view architecture and v2 module path. [Bubbles](https://pkg.go.dev/charm.land/bubbles/v2) supplies textarea and viewport.
- [OpenAI Chat Completions create](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create): streaming deltas, tool calls, optional usage events. The older platform.openai.com URL returned HTTP 403; the developers.openai.com reference was retrieved successfully.
- [modernc SQLite](https://pkg.go.dev/modernc.org/sqlite): database/sql driver and pure-Go build path.
- [Playwright introduction](https://playwright.dev/docs/intro): browser QA and supported OS requirements. The optional worker uses the installed Playwright browser APIs.

Native integrations are implemented with platform capability checks. Linux X11 was exercised locally; macOS and Windows native execution still requires validation on those hosts.

## Completion implementation references

- [Anthropic streaming Messages](https://platform.claude.com/docs/en/api/messages-streaming): content block starts/deltas, partial tool JSON, usage and message completion.
- [Gemini generate/stream content](https://ai.google.dev/api/generate-content): native parts, function calls/results and thought signatures.
- [Go keyring library](https://github.com/zalando/go-keyring): macOS/Windows/Secret Service APIs, Get/Set/Delete and OS prerequisites.
- [Bubblewrap](https://github.com/containers/bubblewrap): namespaces, read-only binds, explicit policy responsibility and limitations.
- [Playwright Browser API](https://playwright.dev/docs/api/class-browser): browser/context lifecycle; installed Playwright API was exercised by the actual synthetic-site workflow.
- [xdotool](https://github.com/jordansissel/xdotool): X11 input/window APIs and Wayland limitation.
- [Apple UI scripting](https://developer.apple.com/library/archive/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/AutomatetheUserInterface.html) and [CoreGraphics CGEvent](https://developer.apple.com/documentation/coregraphics/cgevent): user-granted accessibility, events, pointer/scroll fields and posting.
- [Microsoft desktop input](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput): desktop input restrictions and integrity-level boundaries. Windows implementation uses existing PowerShell/Windows Forms and Win32 calls without privilege elevation.
