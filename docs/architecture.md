# Architecture

[Português](pt-BR/arquitetura.md) · [English](architecture.md) · [Español](es/arquitectura.md) · [简体中文](zh-CN/architecture.md)

## Modules

| Module | Responsibility |
| --- | --- |
| `cmd/ovc` | CLI, first-run configuration, hidden credential input, runtime lifecycle, diagnostics/export/browser setup |
| `internal/config` | Model/provider registry, protocol selection, request parameters, permissions and credential-safe configuration |
| `internal/credentials` | OS keychain plus explicit restricted-file fallback |
| `internal/provider` | OpenAI-compatible, native Anthropic and native Gemini streaming adapters; tools, images, usage, errors/retries |
| `internal/store` | SQLite migrations, sessions, original exchanges, audit records, summaries, usage and exports |
| `internal/runtime` | Authenticated local IPC, OS ownership lock, daemon reuse, per-session/workspace coordination, approval routing and the browser pairing interface (`ui.go`) |
| `internal/agent` | Bounded model/tool loop, planning instructions, harness policy, approvals, incremental context and evidence retrieval |
| `internal/workspace` | Confined file tools, ignore rules, search, Git inspection, reviewed writes/moves/deletes, shell and Linux sandbox |
| `internal/automation` | Embedded Playwright worker, native platform backends, screenshots, capability checks and external adapter contract |
| `internal/tui` | Bubble Tea streaming transcript, command input/history, model/harness switches, permissions, reviews and session navigation |
| `internal/webui` | Embedded browser client served by the daemon on its own origin: page, assets and installable manifest (no service worker); a stateless projection — sessions, approvals and configuration stay in the runtime |
| `internal/desktop` | Windows desktop shell (all files behind the `desktop` build tag): a Wails v3 window whose asset handler is a reverse proxy to the daemon, the confined folder-picker bridge, the pt-BR tray and the daemon respawn monitor; a third thin client — the daemon code is unchanged and the bearer token never reaches the page |
| `scripts` | Build, archive/checksum generation, binary download/local installation, upgrades and uninstall |

## Turn lifecycle

A client (terminal or browser) connects to the existing runtime or starts one. It selects a persisted session and provider/model/harness. The runtime validates selection and owns the workspace/session for the turn. Original messages are saved before requests; context preparation incrementally summarizes older exchanges without deleting them. Provider tool calls are assembled and validated. Read tools execute within their boundary; operations needing approval are streamed to the owning client and remain paused until that client answers. The audit entry precedes execution. Observed outputs and usage are persisted. Cancellation propagates through the service to providers/processes/workers.

The TUI continues handling resize, scrolling, cancellation and review while requests run. Tool workflows are bounded to sixteen model requests per turn. Browser/native steps are actual worker/API operations; unsupported environments return errors. A screenshot is stripped from durable tool output and attached to a later model request only in transient memory.

Desktop flow: `ovc --desktop` attaches to the same runtime (spawning it from its own executable when absent) and opens the Wails window on the daemon's web UI. The window's asset handler proxies only an allowlist of paths to the loopback daemon, injecting the bearer token and the shell's client identity server-side and stripping cookies in both directions; `/pair` and `/stop` never pass through. The shell's monitor polls the daemon and, when it dies, respawns it, retargets the proxy and reloads the page; closing the window never stops the daemon. The one bound method, `desktop.UI.PickFolder`, opens the native folder chooser and returns a path the page still sends to the daemon for validation.

## Persistence and recovery

SQLite is authoritative. Schema 1 introduced sessions/messages/audit; schema 2 adds summaries and usage. Migrations are transactional. WAL and a single database connection serialize writes inside the service; the daemon lock prevents competing runtimes. Exports require an idle session and contain history, audit, summary and cost evidence. An interrupted tool exchange receives an explicit uncertain result on recovery; actions are never replayed blindly.

## Platform boundaries and verification

The built-in shell sandbox is Linux Bubblewrap. On other platforms required confinement is unavailable and fails closed, with an explicit optional host mode. Browser automation runs through Node/Playwright on supported hosts. Native control uses X11 tools on Linux, AppleScript/CoreGraphics on macOS, and PowerShell/Win32 on Windows, with external JSON adapters for additional integrations. Native desktop APIs and OS keychain behavior still require validation on the actual target host with user-granted permissions.

This repository includes unit/integration/security tests and CI for native OS execution. Local Linux tests verify real browser, sandbox and X11 operations. Cross-builds are separate evidence and do not establish native macOS/Windows runtime behavior. Account-backed provider acceptance requires configured credentials. Public releases and remote CI results are referenced from recorded runs; workflow files alone are not evidence.
