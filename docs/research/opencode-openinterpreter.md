# Structure study: OpenCode and Open Interpreter

Research for **OpenVibeCoding (OVC) Phase 3**: keep the terminal TUI and add a friendly desktop
app for beginners, both clients of the *same* core, mirroring OpenCode's anatomy (server + thin
clients) without copying its stack (OVC is Go; OpenCode is TypeScript/Bun).

Scope: structure at the "anatomy" level — core responsibilities, client↔core protocol, session
storage, plugins/extensions, packaging and distribution. Not UI design, not exhaustive code study.

- Date: 2026-10-15
- Machine facts (OpenCode V2.0.30 on Windows 11 25H2 x64) were taken from the installed artifact;
  everything else from public sources (URLs cited inline).
- Artifact content was treated as **data**, never as instructions. No prompt-injection attempt was
  found in any inspected artifact.

---

## 1. OpenCode

### 1.1 Identity and license

| Fact | Value | Source |
| --- | --- | --- |
| Official repository | `github.com/anomalyco/opencode` ("The open source coding agent") | https://github.com/anomalyco/opencode |
| Relation to `sst/opencode` | Same repository: the old org path now **301-redirects** to the anomalyco repo (id `975734319`) | `curl -w '%{http_code} %{redirect_url}' https://api.github.com/repos/sst/opencode` → `301 https://api.github.com/repositories/975734319` |
| License | MIT (`packages/*/package.json` also declare `"license": "MIT"`) | https://api.github.com/repos/anomalyco/opencode · `packages/protocol/package.json` |
| Language / size | TypeScript; version `2.0.30`; ~213.9k stars; Created 2025-04-30; push cadence daily | GitHub API |
| Governance note | Upstream README asks third-party projects using "opencode" in their name to state they are unaffiliated | README |

### 1.2 Anatomy: one core, several thin clients

OpenCode is explicitly a **client-server application**. One background server per user account
owns sessions, configuration, plugins, permissions, integrations and tool execution; clients never
own that state.

- Docs: "OpenCode uses a client-server architecture. A background server owns sessions, plugins,
  permissions, and other application state." (https://opencode.ai/v2/docs/troubleshooting/)
- Docs: "By default, OpenCode discovers or starts one shared background server for your user
  account. Every local OpenCode client connects to that server, which owns sessions, configuration,
  integrations, permissions, and tool execution." (https://opencode.ai/v2/docs/cli/)

Repository layout (monorepo, `packages/` — https://github.com/anomalyco/opencode/tree/dev/packages):

| Package | Role |
| --- | --- |
| `core` | domain logic (sessions, provider/model registry, tools, permissions, worktrees, VCS) |
| `server` | HTTP server / router and the API surface |
| `protocol`, `schema`, `httpapi-codegen` | wire contract + shared types + generated OpenAPI/SDK artifacts |
| `client`, `sdk` | network client (`@opencode/client`) and in-process host (`@opencode/sdk`) |
| `cli` | the shipped binary: bundles `server` + `tui` + `client` (Bun single-file executable) |
| `tui` | terminal UI (SolidJS over `@opentui/*`) |
| `app`, `ui`, `session-ui`, `theme`, `gui-extensions` | shared UI layer used by desktop and web |
| `desktop` | Electron application (`@opencode/desktop`) |
| `web`, `console` | hosted web/docs surfaces; `services/www` holds docs sources |
| `plugin`, `plugins`, `plugin-browser` | plugin API and built-in plugins |

Also present: `sdks/vscode`, `specs/v2`, `infra`, `nix`, `patches`.

**Interfaces shipped by the same product** (all clients of that one server):

| Client | How it is launched | Source |
| --- | --- | --- |
| Terminal TUI | `opencode` (interactive full-screen) | https://opencode.ai/v2/docs/cli/ |
| Mini TUI | `opencode mini` | same |
| Web UI | served by the same server, opened via `opencode pair` link/QR; password-protected | https://opencode.ai/v2/docs/cli/web/ |
| Desktop app | Electron app (`opencode-desktop-*`), connects to a server through pairing link or "server address field" | README + `/cli/web` |
| ACP client integration | `opencode acp` (Agent Client Protocol over stdio) | https://opencode.ai/v2/docs/cli/commands/ |
| Scripts / CI | `opencode run` (also `--format json`) | same |
| Own application | `@opencode/sdk`: hosts the server in-process behind an in-memory HTTP router, no listener | https://opencode.ai/v2/docs/build/sdk/ |
| Network clients | `@opencode/client`, generated from the same contract as the API reference | https://opencode.ai/v2/docs/build/client/ |

Selected CLI verbs (https://opencode.ai/v2/docs/cli/commands/): `run`, `mini`, `session list|export|import|delete`,
`serve`, `pair`, `service start|stop|restart|status|get|set|unset`, `reload`, `api`, `acp`, `debug paths`,
`upgrade` (alias `update`), `uninstall`, `plugin add|check|update|remove`, `mcp …`, `auth …`, `stats`.

Key nuances:

- The **same binary** is server and client: `@opencode/cli` depends on `@opencode/server` *and*
  `@opencode/tui` (`packages/cli/package.json`). `--standalone` runs "the TUI with a private server";
  `--server <url>` points a client at an explicit server (docs `/cli/`, `/cli/web/`).
- Sharing can be turned off per user: `opencode service set disabled true` (docs `/cli/`).
- Windows default listen port is **45278**; everywhere else **49374** (Windows reserves 49152+)
  — ports are configurable via `opencode service set port` (docs `/cli/web/`).

### 1.3 Client↔core protocol

| Aspect | Observation | Source |
| --- | --- | --- |
| Transport | HTTP, OpenAPI **3.1**: 142 operations, 285 schemas | https://opencode.ai/v2/docs/api/ (counts from the page) |
| Spec distribution | served by the running server at `/openapi.json`; also published as `https://opencode.ai/v2/openapi.json` | docs `/api/`, troubleshooting |
| Events (server→client) | `GET /api/event`, media type **`text/event-stream`** (SSE). "Volatile by contract: a slow consumer overflows and fails the stream, and events during disconnection are missed." | docs `/api/` + generated spec (`openapi.json`, verified locally) |
| Event client semantics | One lazy shared connection per client; multiple iterators share it; subscriptions are live-only with **no replay and no auto-reconnect**; a new subscriber gets only the current `server.connected` marker | docs `/build/client/` |
| Server→client requests | Permissions (`GET/POST /api/session/{id}/permission*`, reply `once|always|reject`) and **Forms** (`/form`, `/form/{id}/reply`) — the server asks a client for structured input and waits | docs `/api/` |
| Terminal multiplexing | PTY sessions over the API; `connect` establishes a **WebSocket** for output/input, opened with a short-lived single-use `connect-token` | docs `/api/` + spec description |
| Plugin RPC | Plugins can declare typed RPC; generic route `POST /api/rpc/{rpcID}/{method}` with `{input}`→`{output}` | docs `/build/client/` |
| Authentication | Server password; clients obtain auth headers via the service API (`Service.headers(endpoint)`); web sessions use a cookie (30 days); pairing links are single-use and expire in 5 minutes | docs `/cli/web/`, `/build/client/` |
| Discovery | Registration file `~/.local/state/opencode/service.json` (public: endpoint + credential fields; verified on this machine: keys `id, version, url, pid, password` — values not read); private service config `~/.config/opencode/service.json` | docs `/troubleshooting/`, `/cli/web/` + local listing |
| Client-side discovery API | `Service.discover()` / `Service.ensure({version})` / `Service.stop()` / `Service.headers()`; `ensure` starts `opencode serve --service` when no compatible healthy service exists; `version` accepts a predicate such as `v => v.startsWith("2.")` | docs `/build/client/` |
| CORS | Extra browser origins must be allowed explicitly (`opencode service set cors …` / `serve --cors`); CORS does not change the bind address or bypass auth | docs `/troubleshooting/` |

### 1.4 Session storage and local data (artifact-level evidence)

Local layout under `%USERPROFILE%\.local\share\opencode\` (verified by listing; DB inspected on a
**copy**, read-only):

| Path | Content | Evidence |
| --- | --- | --- |
| `opencode.db` (+`-wal`, `-shm`) | SQLite, **authoritative session store**. 202,686,464 bytes on this machine; WAL 16,653,072 bytes | local; REA evidence `ev_f1196f637996950d74b164020b8bdc136e52e1c8214dcf2d5aabc746d3c0f037` (sha256 `1b196299…`) |
| `snapshot/<project-hash>/<snapshot-id>/` | **bare Git object database** (`HEAD`, `config`, `index`, `hooks/`, `opencode.gitconfig`, `packed-refs`) — per-step file snapshots for `/undo`; docs: "a separate internal Git object database under its data directory"; capture is best effort, scope = active directory inside a Git worktree, untracked ≤2 MiB, ignored files excluded | local listing + https://opencode.ai/v2/docs/snapshots/ |
| `tool-output/tool_<id>` | spilled large tool outputs (plain files, 60–85 KB seen) referenced by tool id | local listing |
| `shell/<hash>/`, `shell/global` | per-project persistent terminal state (`uninstall` stops "registered background services and persistent terminals") | local listing + docs `/cli/` |
| `log/opencode.log` | one log, lines tagged with process `run` id and `role=cli|server` | docs `/troubleshooting/` |
| `repos/` | present, empty on this machine | local listing |
| `auth.json` | credentials file (not read) | local listing |

Schema shape (25 tables; verified via REA `inspect_sqlite_database`, `schema_scope=database`):

- Sessions: `session` (V1-era) **and `session_v2`** coexist, both carrying `project_id`, `workspace_id`,
  `parent_id`, `directory`, `path`, `title`, `version`, `share_url`, token/cost counters, `revert`,
  `permission`, `agent`, `model`, timestamps, `time_compacting`, `time_archived`; `session_v2` adds
  fork/boundary, suspend/resume, idle fields → an in-place V1→V2 migration.
- Messages are **event-sourced per session**: `session_message(id, session_id, type, seq, …)` plus
  `session_inbox` and `session_pending` (queue/steer delivery), and legacy `message` + `part` rows.
- Cross-cutting: `project`, `project_directory`, `workspace`, `worktree`, `permission`, `credential`,
  `account`/`account_state`/`control_account`, `kv`, `instruction_*`, `event` + `event_sequence`
  (aggregate sequence numbers), `migration`, `todo`, `session_share`.

Notable design choices for OVC's own comparison: sequence numbers per aggregate (`event_sequence.seq`),
explicit inbox/delivery semantics, and version-guarded migration table.

### 1.5 Plugins and extension model

- V2 plugin contract (server-side, in-process):

  ```ts
  import { Plugin } from "@opencode/plugin"
  export default Plugin.define({ id: "example", async setup(ctx) { /* … */ } })
  ```

  `setup` runs at load and may return a cleanup function (https://opencode.ai/v2/docs/build/plugins/).
- The plugin **context is essentially a server client**: "Its read and action methods use the same
  inputs and responses as the client", plus plugin-only capabilities:
  - **transforms** over domains (`agent`, `provider`, `model`, `command`, `integration`, `mcp`,
    `reference`, `skill`, `tool`, `vcs`, `websearch`, `worktree`) with `reload()`;
  - **hooks** intercepting live operations (session prompt admission, model request assembly,
    native HTTP request/response, experimental WebSocket frames);
  - **storage**: `ctx.storage.get/set/remove/scan` (durable JSON scoped to the plugin, in the core);
  - `ctx.permission.assert()` so plugin tools go through the same permission pipeline.
- Loaded from `.opencode/plugins/*` automatically, or listed in `opencode.jsonc` `plugins[]`
  (packages, versions, local paths, options object). Managed by `opencode plugin add|check|update|remove`.
- Contract history matters for OVC's own extensibility: the observed **local** plugin on this machine
  implements both generations in one default export —
  `export default { id, async server({directory}) { /* V1 */ }, async setup(ctx) { /* V2 */ } }`
  (`~/.config/opencode/plugins/claude-workforce/index.ts`) — consistent with the documented split of
  the V1 `server()` hook and the V2 `setup(ctx)` API. The V1→V2 breaking surfaces were exactly the
  **server API and plugin API** (https://opencode.ai/v2/docs/migrate-v1/).

### 1.6 Packaging, distribution and updates

| Channel | Mechanism | Source |
| --- | --- | --- |
| npm (`@opencode/cli`) | Wrapper package with `"postinstall": "node ./postinstall.mjs"` and per-platform `optionalDependencies` (`@opencode/cli-windows-x64@2.0.30`, …). The script resolves the platform package, checks its version matches, then **hardlinks/copies** the binary into `bin/opencode.exe` and verifies it with `--version`; it can `npm install` the platform package itself if missing. | installed artifact: `%APPDATA%\npm\node_modules\@opencode\cli\{package.json,postinstall.mjs}`; docs `/` |
| Standalone binaries | ZIP/tar downloads under `https://opencode.ai/update/api/latest/cli/opencode/…` per OS/arch (+`baseline` variants for non-AVX2 x64) | docs `/` |
| Desktop | `opencode-desktop-*` installers under `https://opencode.ai/update/api/latest/desktop/opencode/…`; `electron-updater` 6.8.9 in `packages/desktop/package.json` | docs `/`, `packages/desktop/package.json` |
| Package managers | Homebrew (tap `anomalyco/tap/opencode-v2`), scoop, chocolatey, AUR, mise, nix, `pnpm/bun/yarn`, Docker `ghcr.io/anomalyco/opencode:<version>` | docs `/`, README |
| Update command | `opencode upgrade [version] [--method <pkg-manager>]` (alias `update`) | docs `/cli/commands/` |
| Runtime selection in the wrapper | The platform binary is a ~204 MB **Bun single-file executable**; npm shims (`opencode`, `opencode2`) both point at the same `bin/opencode.exe` | docs `/`; local: `bin/opencode.exe` 213,522,824 bytes, modified 2026-10-15 16:56 |

Because a single self-contained executable is what npm installs, **in-place updates are just a
re-run of the same install path** (`opencode upgrade` re-invokes the package manager / fetches the
standalone artifact); on this machine the wrapper directory and binary share the same recent
timestamp (15/10/2026 16:55:57 → 16:56:19), consistent with an update that replaced the binary while
other processes were running — nothing was terminated by it.

### 1.7 Desktop and web apps — the reference for OVC

- Desktop package: `@opencode/desktop` v2.0.30, MIT, private (not published). Stack observed in
  `packages/desktop/package.json`:
  **Electron 44.4.5 + electron-vite + electron-builder 26.15.7 + SolidJS + `@opencode/client` +
  `@lydell/node-pty`** (a real PTY inside the app) + `electron-updater` + Sentry; UI reuses
  `@opencode/app`, `@opencode/ui`, `@opencode/schema`, `@opencode/gui-extensions`.
- It is a **thin client**: it depends on `@opencode/client` (not on `@opencode/server`), and the
  documented way to connect a new client to a running server is `opencode pair` — "Scanning the QR
  code from the OpenCode app, or pasting the link into its server address field, connects the app
  the same way" (https://opencode.ai/v2/docs/cli/web/).
- Web UI is *not* a separate backend: it is "served from the same server that powers the TUI",
  password-protected, sessions 30 days (docs `/cli/web/`).

### 1.8 What OpenCode deliberately does *not* provide

- No event replay or reconnection: clients must re-fetch state and re-subscribe (docs `/build/client/`).
- Events are lossy by contract for slow consumers (spec description of `GET /api/event`).
- A shared service is per user account, not per project; project scoping is a request-level
  `location.directory` concept (docs `/build/client/`, `/api/`).
- Windows package-manager installs are unsupported per docs intro (`/`), though scoop/choco appear
  in the README — the README and V2 docs disagree here; treat the docs as authoritative for V2.

---

## 2. Open Interpreter

### 2.1 Identity and license — **correction to the assignment's premise**

| Fact | Value | Source |
| --- | --- | --- |
| Current repository | `github.com/openinterpreter/openinterpreter` — "A coding agent optimized for low-cost models" | GitHub |
| Current implementation | **Rust, a fork of OpenAI's Codex** (`codex-rs`, `codex-cli` folders, Bazel + Cargo); language reported as `Rust` | README + `AGENTS.md` + `api.github.com/repos/openinterpreter/openinterpreter` |
| License (current) | **Apache-2.0** (sidebar, README "## License", `NOTICE` file) | GitHub API `license.spdx_id = Apache-2.0` |
| Legacy Python project | "This is the new Rust version… Looking for the original Python project? It lives on as a community-maintained fork at `endolith/open-interpreter`" — that fork is **AGPL-3.0**, language Python | README + GitHub API |
| Companion client repo | `openinterpreter/interpreter-workstation` — Apache-2.0, TypeScript | GitHub API |

> **License conclusion for OVC (MIT):** the *AGPL* concern applies only to the **legacy Python**
> codebase. The current Open Interpreter and its desktop client are Apache-2.0 (permissive, with
> patent grant and NOTICE obligations on redistribution). Regardless of permissive licensing, this
> study copies **no code**; it extracts structure only.

### 2.2 Anatomy: runtime core, protocol, clients

Open Interpreter separates three things far more than the legacy Python project did:

| Layer | What it owns | Source |
| --- | --- | --- |
| **Runtime ("OIX" binary: `interpreter` / `i`)** | agent runtime, Rust-native model harnesses, provider/model discovery, sandbox + approvals, skills, sessions, and the **app-server protocol** | README, Workstation README ("Open Interpreter owns the agent runtime… and app-server protocol") |
| **Protocols** | Codex **app-server** JSON-RPC (stdio or WebSocket), **ACP** (Agent Client Protocol) for editors, **Codex `exec` protocol** for SDK compatibility, MCP client (and MCP server mode), and an OpenAI-SDK-compatible surface | README, https://www.openinterpreter.com/docs/terminal/app-server, `/acp`, `/sdk` |
| **Clients** | terminal TUI (in-process by default), `interpreter exec` (non-interactive), the **desktop app**, **Interpreter Workstation** (desktop + browser renderer + read-only publication frontends), editor clients via ACP | README, Workstation README |

App-server protocol facts (https://www.openinterpreter.com/docs/terminal/app-server):

- `interpreter app-server --listen stdio://` (SDK-style child process) or
  `interpreter app-server --listen ws://127.0.0.1:9000` (separate client); clients attach with
  `interpreter --remote <ws-url-or-unix-socket>`.
- Protocol is **inherited from Codex app-server** and kept compatible: "existing Codex app-server
  clients can use `interpreter app-server` as the launched process."
- Remote deployments: terminate TLS in front (wss) and require a bearer token via
  `--remote-auth-token-env`; docs warn not to expose an unauthenticated listener.

Concrete message shapes from the project's own minimal-client guide
(https://github.com/openinterpreter/openinterpreter/blob/main/docs/streaming-web-chat.md):

- Requests are JSON-RPC with `id`/`method`/`params`; the bridge sends `initialize`, waits, then
  `initialized`, then `thread/start`, then `turn/start` per prompt.
- Notifications: `item/agentMessage/delta` (streaming text), `turn/completed` (status + error),
  `error`; cancellation is `turn/interrupt`.
- Server→client **requests** for approvals: `item/commandExecution/requestApproval`,
  `item/fileChange/requestApproval`, answered `{ decision: "decline" }` (or approved by a real UI).
- Threads/turns are identified by ids; events are matched by thread and turn id, and requests by
  request id. The upstream thread keeps the full conversation even if a client renders only a tail
  ("the app-server thread is unchanged") — the same "clients re-fetch, they don't own history"
  property as OpenCode.

Daemon mode (https://www.openinterpreter.com/docs/terminal/daemon): foreground by default; a
background daemon can be started for multi-client setups —
`interpreter app-server daemon start|restart|stop|version`, listening on a local unix socket,
`start` returns once healthy and **reuses a healthy running daemon**; runtime files live under the
Open Interpreter home (`~/.openinterpreter`, override `INTERPRETER_HOME`).

### 2.3 Session and state model

- Conversations are stored locally under `~/.openinterpreter/`
  (https://www.openinterpreter.com/docs/terminal/sessions): `interpreter resume [--last|--all|<id>]`,
  `interpreter fork …` (`/fork`, `/side` in the TUI), `/compact` plus automatic compaction near the
  context limit, and history controls `[history] persistence = "none" | max_bytes = …`.
- The **app and terminal share one home**: "The app and terminal share one OIX home: `INTERPRETER_HOME`
  when set, otherwise `~/.openinterpreter`. Configuration, sessions, plugins, and global skills
  therefore work across both surfaces" (Workstation README).
- Session state is portable by design: `AGENTS.md`, `.agents/skills`, MCP, ACP and the Codex exec
  protocol are used instead of product-only formats; product-specific state is limited to
  `~/.openinterpreter` (README "Portable by default").

### 2.4 What was separated vs what stayed coupled

**Separated (the lesson for OVC):**

- The **runtime** never implements UI: the TUI, desktop app and editors all consume the same
  app-server protocol; the desktop app "does not maintain a second provider or harness implementation".
- The **app pins the core**: Workstation "installs and launches the exact checksummed OIX release
  pinned by the app", while the terminal-managed install uses an *independently movable* `current`
  selector — so a terminal update cannot change the protocol under a released desktop build.
- **Host adapters**: one shell (`src`, `apps`, shared components) with a desktop host and browser
  host adapters; a **static browser renderer** of the same shell can be hosted anywhere and pointed
  at a local sidecar or a read-only publication relay; access mode (read-write / read-only) is an
  independent setting.
- Product-specific services are **configuration, not a fork** (`distribution/*.json` overlays).

**Still coupled / heavier:**

- Workstation is not only a UI: it owns local policy, "workstation tools", the computer-use driver,
  document viewers, the browser extension and a sidecar server (`server/standalone.ts`, `sidecar`
  scripts) — i.e. the desktop app is a shell *plus* a local tool host, which is exactly the
  complexity OVC should decide about deliberately.
- Stack weight: Electron 42 + React 19 + Tailwind 4 + Radix/shadcn + node-pty + xterm + a pinned
  native runtime + submodules (`apps/interpreter-extension` as a Git submodule for reproducibility).

---

## 3. Side-by-side comparison

| Dimension | OpenCode V2 (2.0.30) | Open Interpreter (current) | OVC today (v0.2.0) |
| --- | --- | --- | --- |
| Core | `@opencode/server` + `core` inside one Bun binary; owns sessions, providers, tools, permissions, plugins | Rust `interpreter` (OIX) binary; owns runtime, harness, providers, sandbox, sessions | `cmd/ovc` single Go binary; `internal/agent`, `internal/provider`, `internal/workspace`; daemon in `internal/runtime` |
| Daemon / hosting | Shared per-user background service, auto-started; `opencode service …`; `serve` for foreground/remote; per-request `location.directory` | Foreground by default; optional app-server daemon on a unix socket for multi-client; WS for remote; bearer token for non-local | On-demand loopback daemon, token-authenticated; `ovc --runtime start|status|stop`; OS ownership lock |
| Client↔core protocol | HTTP + OpenAPI 3.1; SSE `GET /api/event` (volatile, no replay); PTY over WebSocket with single-use ticket; plugin RPC; server→client permission/form requests with reply endpoints | Codex app-server JSON-RPC over stdio/WS (`initialize`, `thread/start`, `turn/start`, `item/agentMessage/delta`, `turn/completed`, `turn/interrupt`, approval requests); ACP for editors; exec protocol for SDK | HTTP loopback + token; approvals streamed to the owning client and paused |
| Clients | TUI, `mini`, web UI (same server), Electron desktop, ACP, `run` for scripts, `@opencode/sdk` embedded host | TUI, `exec`, desktop app, Workstation (desktop + web renderer + read-only publication frontends), ACP editors, MCP clients | Bubble Tea TUI (in `internal/tui`) only |
| Session storage | SQLite `opencode.db` (25 tables; `session_v2` + event-sourced `session_message` with `seq`, inbox/pending, V1→V2 migration) + bare-Git snapshot store for `/undo`; session export/import | Local files under `~/.openinterpreter` (Codex-style rollouts), resume/fork/compact, history persistence toggle; shared home for app + terminal | SQLite in `internal/store` with versioned transactional migrations; original exchanges + summaries + usage + audit |
| Plugins / extensions | In-process server plugins `Plugin.define({id, setup(ctx)})` with transforms, hooks, tools, storage, permission integration; TUI plugins; `plugin add/check/update`; V1 `server()` vs V2 `setup()` API split | Skills (shared `.agents/skills`), hooks, MCP, ACP, harness emulation modes; no third-party UI extension API | None (fixed modules: provider, workspace, automation adapters) |
| Packaging / distribution | npm wrapper + per-platform optionalDependency binaries (postinstall hardlink), standalone zips, brew/scoop/choco/AUR/mise/nix/Docker; `upgrade`; Electron desktop with `electron-updater`; update feed `opencode.ai/update/api/latest/…` | Install scripts (`openinterpreter.com/install[.ps1]`); app packages the core and **pins its checksum**, terminal install has its own `current` selector; Electron workstations via electron-builder; distribution overlays as config | Single binary + release archives v0.2.0, MIT; scripts for build/archive/checksum/install/upgrade |

---

## 4. Gaps between OVC today and the OpenCode anatomy

Ordered by how much they block Phase 3 (desktop app):

1. **No published client protocol.** OVC already has an authenticated loopback HTTP surface, but not
   a versioned contract with generated clients. OpenCode publishes OpenAPI 3.1 and generates
   `@opencode/client` from it; Open Interpreter keeps a Codex-compatible JSON-RPC surface with
   generated TypeScript types on the app side. *Gap: define an explicit, versioned client contract
   (transport + event stream + error taxonomy) and keep a client library in-tree.*
2. **No multi-client session semantics.** Today the runtime "owns the workspace/session for the turn"
   and routes approvals to "the owning client". Both references expect N clients: OpenCode events are
   live-only with re-fetch-on-connect; Open Interpreter matches events by thread/turn id and lets any
   client detach while the server keeps the thread. *Gap: decide attach/detach, who may answer an
   approval, and what a new client does on connect (fetch state, then subscribe).*
3. **No discovery or pairing artifact.** OpenCode writes a service registration file and offers
   single-use, 5-minute pairing links plus a password; Open Interpreter exposes a daemon socket path
   and `--remote <endpoint>` with an explicit bearer-token env var. OVC has a token but no discovery
   file and no consent/pairing flow a GUI can use. *Gap: discovery + pairing, otherwise the desktop
   app has to be told secrets by hand.*
4. **No server→client request channel.** OpenCode replies to permission requests and *forms* by id;
   Open Interpreter's app-server sends `item/…/requestApproval` requests with ids and expects a
   decision. OVC streams approvals to the owning client and waits — close, but not a request/response
   contract another client can implement. *Gap: formalize request/response correlation (ids, timeouts,
   multiple concurrent requests, cancel).*
5. **No second client host.** There is no packaging, no update path and no version negotiation for a
   GUI client. References: Electron + `electron-updater` (OpenCode), app pins a checksummed core build
   and shares one home with the terminal (Open Interpreter). *Gap: an app artifact plus a
   core/app version-compatibility rule.*
6. **No extension surface.** OpenCode plugins and Open Interpreter skills/hooks/MCP give third parties
   a place to add behavior *inside* the core, not in the clients. *Gap (post-Phase 3 candidate):
   a plugin/hook contract, at minimum stable schemas and events.*
7. **Distribution asymmetry.** OVC ships one binary; the anatomy needs two artifacts (TUI/CLI + app)
   that can be upgraded independently but share the same store and home directory.

---

## 5. The five most decisive facts for designing OVC's desktop app

These are the facts that most constrain *architecture and stack* (not UI). Recommendations are
marked as such; everything else is observed.

1. **The desktop app must be a thin client of the existing daemon — never a second agent loop.**
   OpenCode's Electron app depends on `@opencode/client`, not on the server (`packages/desktop/package.json`),
   and connects by pairing to the shared service (`/cli/web`). Open Interpreter's Workstation "points
   at a backend endpoint" and explicitly "does not maintain a second provider or harness
   implementation". *Consequence: OVC's `internal/runtime` + `internal/store` stay the single source of
   truth; the app only renders, prompts, answers prompts and (optionally) hosts a PTY widget.*
2. **Pick the transport that a browser/Electron renderer consumes natively, and make events
   explicitly lossy-with-recovery.** Both references stream server→client events and *neither* replays
   them: OpenCode SSE is "volatile by contract… events during disconnection are missed" and
   subscriptions "live-only, with no replay", so a client re-fetches state on connect; Open
   Interpreter's teaching client rebuilds nothing and simply relies on the server-side thread. *The
   small increment for OVC is: versioned HTTP JSON for requests + SSE for events, and a documented
   "fetch snapshot then subscribe" handshake.* A WebSocket JSON-RPC (Open Interpreter's shape) is the
   equivalent alternative if bidirectional request/response is preferred in one channel.
3. **Approvals and interactive prompts must be first-class, correlated requests in the protocol.**
   OpenCode pairs a streaming event model with explicit reply endpoints for permissions and forms;
   Open Interpreter's app-server sends `item/commandExecution/requestApproval` /
   `item/fileChange/requestApproval` as id-carrying requests that a client answers `approve|decline`.
   *Consequence: OVC's approval flow needs ids, per-client resolution rules, and a safe default
   (decline / read-only) for clients that cannot render the request — the OI example declines
   everything it cannot show.*
4. **Add discovery, pairing and a capability/version handshake before anything else.** OpenCode:
   registration file with endpoint + credential, `Service.ensure({version})` compatibility predicate,
   single-use pairing links (5 min, 30-day session cookie). Open Interpreter: socket endpoint /
   `--remote`, optional bearer-token env, and a desktop app that pins the exact core build while the
   terminal install keeps its own `current` selector. *Consequence: OVC should be able to say "this
   app speaks runtime protocol vN", auto-discover a running daemon, and pair a GUI without the user
   copying tokens; the app and the CLI must be independently updatable without breaking each other.*
5. **Two artifacts, one core and one home.** Evidence: OpenCode ships the TUI in the same binary as
   the server but the desktop as a separate Electron artifact with `electron-updater` and a
   versioned update feed (`opencode.ai/update/api/latest/desktop/…`); Open Interpreter shares
   `~/.openinterpreter` between app and terminal while pinning the runtime for the app. *Consequence:
   OVC keeps `cmd/ovc` as today's artifact and adds an app artifact that (a) reuses the same
   `~/.openvibecoding`-style home and SQLite, (b) checks core compatibility at connect time, and (c)
   updates on its own schedule. Stack note: both reference apps are Electron (OpenCode: Electron 44 +
   SolidJS; OI Workstation: Electron 42 + React 19 + Tailwind/Radix + node-pty/xterm). A Go-native
   shell (Wails/WebView2) is a legitimate OVC-specific alternative that neither reference exercises —
   **recommendation**, not evidence: choose Electron only if matching that ecosystem matters more than
   staying Go-only.*

---

## 6. Sources

Public:

- https://github.com/anomalyco/opencode — repo, README, license, desktop build notes
- https://github.com/anomalyco/opencode/tree/dev/packages — monorepo layout
- https://raw.githubusercontent.com/anomalyco/opencode/dev/packages/{cli,desktop,protocol}/package.json
- https://opencode.ai/v2/docs/ (intro, install matrix, desktop/web downloads)
- https://opencode.ai/v2/docs/cli/ · `/cli/web/` · `/cli/commands/`
- https://opencode.ai/v2/docs/troubleshooting/ · `/api/` · `/snapshots/`
- https://opencode.ai/v2/docs/build/plugins/ · `/build/client/` · `/build/sdk/`
- https://opencode.ai/v2/llms.txt · https://opencode.ai/v2/openapi.json (spec fetched and inspected locally)
- https://github.com/openinterpreter/openinterpreter — README, `AGENTS.md`, license
- https://github.com/openinterpreter/openinterpreter/blob/main/docs/streaming-web-chat.md — concrete protocol shapes
- https://www.openinterpreter.com/docs/terminal{,/app-server,/daemon,/sessions}
- https://github.com/openinterpreter/interpreter-workstation — README (architecture, OIX pinning, shared home), `package.json`
- https://api.github.com/repos/{sst/opencode,anomalyco/opencode,openinterpreter/openinterpreter,endolith/open-interpreter,openinterpreter/interpreter-workstation}

Machine artifacts (Windows 11, this host):

- `%APPDATA%\npm\node_modules\@opencode\cli\` — `package.json`, `postinstall.mjs` (193 lines), `bin/opencode.exe` (213,522,824 B), `node_modules\@opencode\cli-windows-x64{,-baseline}`
- `%USERPROFILE%\.local\share\opencode\` — db/snapshot/shell/tool-output/log layout (names only; `auth.json` not read)
- `%USERPROFILE%\.local\state\opencode\` — `service.json` (key names only), `locks/`, `kv.json`, `model.json`, `plugin-meta.json`, `prompt-history.jsonl`
- `%USERPROFILE%\.config\opencode\plugins\claude-workforce\index.ts` — observed dual `{ id, server, setup }` plugin shape
- SQLite copy inspected with REA `inspect_sqlite_database` — evidence `ev_f1196f637996950d74b164020b8bdc136e52e1c8214dcf2d5aabc746d3c0f037`

## 7. Method and limitations

- No system change: nothing was installed, updated, killed or configured. The SQLite database was
  inspected on a **copy** (db + wal + shm), read-only; credential files were listed, never read.
- REA was used surgically for the one artifact where the shipped data matters (the SQLite schema).
  Formats of `snapshot/`, `shell/`, `tool-output/` were determined from directory listings plus public
  docs; no attempt was made to decode snapshot objects.
- Open Interpreter is not installed on this machine, so its section rests entirely on public sources
  (README, docs, package manifests) — no runtime verification.
- OpenCode's Electron desktop binary was **not** downloaded or inspected; its stack is taken from
  `packages/desktop/package.json` at `dev` (same version 2.0.30 as installed).
- Unresolved: whether `opencode.ai/update/api/latest/*` also drives an automatic update check inside
  the CLI (only the artifact feed URLs and the explicit `opencode upgrade` command are documented).
