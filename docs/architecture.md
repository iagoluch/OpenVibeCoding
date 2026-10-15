# Architecture

## Modules

| Module | Responsibility |
| --- | --- |
| `cmd/interpreter` | CLI, first-run configuration, hidden credential input, runtime lifecycle, diagnostics/export/browser setup |
| `internal/config` | Model/provider registry, protocol selection, request parameters, permissions and credential-safe configuration |
| `internal/credentials` | OS keychain plus explicit restricted-file fallback |
| `internal/provider` | OpenAI-compatible, native Anthropic and native Gemini streaming adapters; tools, images, usage, errors/retries |
| `internal/store` | SQLite migrations, sessions, original exchanges, audit records, summaries, usage and exports |
| `internal/runtime` | Authenticated local IPC, OS ownership lock, daemon reuse, per-session/workspace coordination and approval routing |
| `internal/agent` | Bounded model/tool loop, planning instructions, harness policy, approvals, incremental context and evidence retrieval |
| `internal/workspace` | Confined file tools, ignore rules, search, Git inspection, reviewed writes/moves/deletes, shell and Linux sandbox |
| `internal/automation` | Embedded Playwright worker, native platform backends, screenshots, capability checks and external adapter contract |
| `internal/tui` | Bubble Tea streaming transcript, command input/history, model/harness switches, permissions, reviews and session navigation |
| `scripts` | Build, archive/checksum generation, binary download/local installation, upgrades and uninstall |

## Turn lifecycle

A client connects to the existing runtime or starts one. It selects a persisted session and provider/model/harness. The runtime validates selection and owns the workspace/session for the turn. Original messages are saved before requests; context preparation incrementally summarizes older exchanges without deleting them. Provider tool calls are assembled and validated. Read tools execute within their boundary; operations needing approval are streamed to the owning client and remain paused until that client answers. The audit entry precedes execution. Observed outputs and usage are persisted. Cancellation propagates through the service to providers/processes/workers.

The TUI continues handling resize, scrolling, cancellation and review while requests run. Tool workflows are bounded to sixteen model requests per turn. Browser/native steps are actual worker/API operations; unsupported environments return errors. A screenshot is stripped from durable tool output and attached to a later model request only in transient memory.

## Persistence and recovery

SQLite is authoritative. Schema 1 introduced sessions/messages/audit; schema 2 adds summaries and usage. Migrations are transactional. WAL and a single database connection serialize writes inside the service; the daemon lock prevents competing runtimes. Exports require an idle session and contain history, audit, summary and cost evidence. An interrupted tool exchange receives an explicit uncertain result on recovery; actions are never replayed blindly.

## Platform boundaries and verification

The built-in shell sandbox is Linux Bubblewrap. On other platforms required confinement is unavailable and fails closed, with an explicit optional host mode. Browser automation runs through Node/Playwright on supported hosts. Native control uses X11 tools on Linux, AppleScript/CoreGraphics on macOS, and PowerShell/Win32 on Windows, with external JSON adapters for additional integrations. Native desktop APIs and OS keychain behavior still require validation on the actual target host with user-granted permissions.

This repository includes unit/integration/security tests and CI for native OS execution. Local Linux tests verify real browser, sandbox and X11 operations. Cross-builds are separate evidence and do not establish native macOS/Windows runtime behavior. Account-backed provider acceptance requires configured credentials. No public release or remote CI success is inferred from workflow files.
