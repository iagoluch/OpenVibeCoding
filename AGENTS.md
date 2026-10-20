# OpenVibeCoding — project instructions

Terminal coding agent (`ovc`) in Go with Bubble Tea v2. Module `github.com/iagoluch/OpenVibeCoding`, Go 1.27.2+ (the stdlib of release binaries must be free of the GO-2026-6603..6617 vulnerabilities, fixed in Go 1.27.2).

## Build and test

```sh
go build ./...                      # compile everything
go build -o ovc.exe ./cmd/ovc       # local binary (Windows: ovc.exe, Unix: ovc)
go test ./...                       # unit/integration tests
go vet ./...
bash scripts/package.sh             # release archives + SHA256SUMS (needs Python 3)
```

- CI (`.github/workflows/ci.yml`) runs tests, vet and cross-builds on Ubuntu, macOS and Windows; keep it green.
- `go test -race ./...` needs CGO (a C toolchain); CI covers it where available.

## Layout

| Path | Responsibility |
| :--- | :--- |
| `cmd/ovc/` | CLI entry, flags, wiring |
| `internal/agent/` | Model/tool loop, approvals, context compaction |
| `internal/provider/` | Streaming adapters: OpenAI-compatible, Anthropic, Gemini |
| `internal/runtime/` | Authenticated loopback daemon (owns SQLite) |
| `internal/store/` | SQLite schema and migrations (bump `user_version` on change) |
| `internal/workspace/` | Confined file tools, shell sandbox (bubblewrap on Linux) |
| `internal/tui/` | Bubble Tea interface, slash commands |
| `internal/webui/` | Embedded browser client served by the daemon (page, assets, no service worker) |
| `internal/desktop/` | Windows desktop shell behind the `desktop` build tag: Wails window = reverse proxy to the daemon, confined folder-picker bridge, pt-BR tray, respawn monitor |
| `internal/automation/` | Playwright browser worker, desktop backends |
| `internal/config/` | Provider registry and permissions |
| `docs/` | User guide, architecture, security, automation, testing |

## Rules

- Security invariants are non-negotiable: per-operation approvals, audit before execution, secret redaction before display/persistence, `os.Root` confinement, fail-closed sandboxing. Never weaken them silently.
- Keep docs (`docs/guide.md`, `docs/security.md`) in sync with behavior changes.
- Stdlib first; discuss new dependencies before adding them.
- Match existing patterns and naming; keep changes reviewable.
