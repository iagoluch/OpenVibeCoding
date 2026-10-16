# OpenCode UI inventory — client surfaces, screens, interaction patterns, and beginner-readiness

**Purpose:** Phase 2 of the OpenVibeCoding research (see `opencode-openinterpreter.md` for the structural study). This document inventories **what OpenCode's user interfaces actually are and how they behave**, so the Phase 3 desktop app can be designed as "the same interaction model, friendlier" without guessing.

- **Subject:** OpenCode V2 (`@opencode/*` 2.0.30), repo `anomalyco/opencode`, branch `dev`, as of **2026-10-15**.
- **Method:** official V2 docs (site), raw repo sources (package manifests, docs sources, README), public GitHub issues (search API, queried 2026-10-15), and read-only local artifacts on this machine. The desktop app is **not installed here** (verified: `%LOCALAPPDATA%\Programs\opencode-desktop`, `%LOCALAPPDATA%\opencode-desktop`, `%ProgramFiles%\opencode` all absent [S16]), so desktop facts come from docs/repo/issues, not from running the app.
- **Evidence tags** `[S#]` map to the Sources section at the end.
- **Doc-generation caveat:** the repo's `docs/*.mdx` sources and the published `/v2/docs/*` pages have drifted (e.g. raw `permissions.mdx` still uses V1's `permission` field and `--auto`, while the V2 page uses `permissions` with `action`/`resource`/`effect`; raw `tui.mdx` still references `tui.json`, while V2 uses `cli.json`). Where both exist, **the V2 site page is treated as authoritative for V2** and the raw mdx is used only for page bodies not otherwise captured, flagged inline.

---

## 1. Client surfaces

OpenCode is one server (sessions, config, permissions, tool execution) with several thin clients. Every local client normally connects to **one shared background service** per user account — not to a private process [S1].

| Surface | UI stack (observed) | How it is launched | Implied audience | Sources |
| --- | --- | --- | --- | --- |
| **Full-screen TUI** (primary) | `@opentui/core` + `@opentui/keymap` + `@opentui/solid` (SolidJS renderer for terminals), `solid-js`, `uqr` (in-terminal QR), `opentui-spinner`; tree-sitter bash/PowerShell parsers live in the CLI for command parsing | `opencode` (optionally with a directory: `opencode ~/code/project`) | Terminal-native developers; default product | [S2][S12: cli, tui] |
| **Mini** (minimal interactive mode) | Same CLI binary; line-based interface instead of full-screen alt-mode; own settings group (`mini.*`), spinners, optional replay of history | `opencode mini` (~`opencode mini --help` for session/model/agent/prompt/replay flags) | Scripts, log-preserving workflows, lightweight terminals | [S1][S3] |
| **Web UI** (served by the server itself) | Served from the same server that powers the TUI; password protected. The GUI codebase is `@opencode/app` (SolidJS + `@solidjs/router` + TanStack Query/Virtual + Tailwind 4 + Kobalte + qr-scanner + PWA plugin). Desktop consumes the same `@opencode/app` package — the web UI serving path is not spelled out in docs, so identity between web UI and desktop renderer is **probable, not proven** | Server runs by default; access via `opencode pair` (one-shot link + QR, 5-min expiry; 30-day cookie), or `opencode serve` for a foreground server; links also work through SSH port-forward | Browser and remote/SSH users; "open on my other device" | [S10][S12: app, desktop] |
| **Desktop app** (BETA) | Electron 44.4.5 + `electron-vite` + `electron-builder`; renderer built from `@opencode/app` + `@opencode/ui` + `@opencode/session-ui` + `@opencode/gui-extensions`; `@lydell/node-pty` for terminals; `ghostty-web` terminal renderer in the GUI stack; `electron-updater`; a prebuilt `opencode-cli` binary ships **outside** `app.asar` as a sidecar; registers the `opencode://` protocol handler | Installer from releases/`opencode.ai/download` (`opencode-desktop-*` artifacts), Homebrew cask or Scoop `extras/opencode-desktop` | Non-terminal users; the "friendly front door" | [S12: desktop, gui-extensions][S14] |
| **`run` (headless)** | No UI; prints model output to stdout; `--auto` auto-approves non-denied permissions | `opencode run "Explain this repository"` | CI jobs, scripts, automation | [S1][S5] |
| **Server-only** | `opencode serve --hostname --port` (foreground; prints "server password …"), used with `opencode --server <url>` clients; `opencode --standalone` for a private server | terminal | Supervisors (systemd/Docker), shared/remote hosts | [S10][S1] |

Terminal title integration exists (`terminal.title`), and the TUI screenshot shows the window title as `OC | <session title>` [S3][S16].

**Audience inference is the report's own reading**, not doc text, except where the docs say so (e.g. `run` is "designed for scripts, CI jobs" [S1]).

---

## 2. Screens and flows

### 2.1 First run / onboarding

- **TUI:** running `opencode` opens straight into the session view; `cli.json` is auto-created on first run ("OpenCode adds `$schema` when it creates or migrates this file") [S3]. Before a provider is connected, prompts fail with `No model is available for session …` and the documented recovery is `/connect` then `/models` [S9]. After an update, the TUI shows the release highlights once on the home screen (`/highlights` to see again; `highlights: false` to disable) [S3].
- **Project instructions onboarding:** `/init` — "guided setup for creating or updating `AGENTS.md`" is documented on the (V1-era) TUI page [S13: tui.mdx]; V2 config still treats rules/AGENTS.md as first-class [S3][S6].
- **Desktop:** has an explicit **first-launch defaults** mechanism — `packages/desktop/src/main/storage/install-state.ts` decides "fresh install vs existing state" from `.dat` files (`default.dat`, `opencode.*.dat`). Known bug: an unrelated `.dat` file suppresses first-launch onboarding [#54719]. The desktop shows a splash, then a Home screen with a **server picker** (server name + status dot; "Default server" section with **Clear**) [S13: troubleshooting].
- **Web:** first contact is the pairing link/QR from `opencode pair`; opening it signs the browser in with a session cookie and loads the web UI; changing the server password signs every session out [S10].
- **Mini:** entry/exit splash banners and a footer are configurable (`mini.splash`, `mini.footer`); history replay on resume/resize is on by default [S3].

### 2.2 Main session screen (chat, streaming, tool calls)

Anatomy, from the official TUI screenshot [S16: lander/screenshot.png] and V2 docs [S2][S3][S4]:

- **Header line:** session title (e.g. `# Homepage button color change in repo workflow`) plus live counters — tokens, context %, and cost (`39,413  20%  ($0.29)`). Per-turn token usage and TPS in footers are settings (`session.tps`, `debug.turn_tokens`) [S3].
- **Transcript:** user prompt shown in its own boxed block; assistant text streams inline; tool calls render as compact lines (`* Grep "…"`, `→ Read packages/…`), with related exploration optionally **grouped** (`session.grouping`) and collapsible; thinking blocks are toggleable (`session.thinking`, `/thinking`) [S16][S3].
- **Interactivity while working:** `Enter` **steers** the active session; `Ctrl+X` then `Enter` **queues** a prompt for later; queued prompts are manageable (`<leader>q`), undoable (`ctrl+u`) and deletable (`ctrl+d`); `Esc` interrupts the session; `ctrl+b` backgrounds blocking tools [S2][S4].
- **Composer:** multi-line input (Shift+Enter / Ctrl+J), `@` fuzzy file references with line ranges (`@src/auth.ts#20-45`), `!` shell mode, `/` slash commands, paste compaction, image attachments with viewer (`<leader>i`) [S2][S4][S3].
- **Status line:** current agent + model (screenshot: `Build · claude-opus-4-5`; footer hints `esc interrupt`, `ctrl+t variants`, `tab agents`, `ctrl+p commands`) — a persistent affordance row teaching shortcuts [S16].
- **Side panels and views:** sidebar toggle (`<leader>b`), terminal pane (`<leader>t`), session timeline (`<leader>g`), status view (`<leader>s`), debug panel/console, a **scrap** screen, which-key panel (`ctrl+alt+k`) [S4].
- **Subagents in the UI:** child sessions are navigable (`down` toggles the subagent picker; `left`/`right` walk children; `up` returns to parent); users can select running shell commands in the **Shell** tab and view captured stdout/stderr with scroll controls [S4].
- **Status/health affordances:** the settings screenshot shows a bottom bar with `● 3 MCP /mcps` and a `local` indicator (server/workspace locality) — small, always-visible system state [S16: settings.png].

### 2.3 Approvals / permissions (where beginners get stuck or saved)

- Rules are evaluated in order to `allow` / `ask` / `deny` (V2 config key `permissions`, rules with `action`, `resource`, `effect`; last matching rule wins) [S5].
- **Defaults are permissive but guard the sharp edges:** `read` allows everything except `.env` patterns; `external_directory` (touch paths outside the project) and `doom_loop` (same tool call 3x identically) default to `ask` [S5, raw mdx].
- **What "ask" looks like:** the prompt offers `once` (approve this request), `always` (approve future matching requests — for the rest of the session, using suggested patterns provided by the tool, e.g. `git status*`), or `reject`. The prompt can go fullscreen for review (`ctrl+f`, keybind `permission.prompt.fullscreen`) — useful for long commands/diffs [S5][S4].
- **Auto-approve mode:** `--auto` only bypasses `ask` (never `deny`); in TUI it is toggled via the command palette and shows a muted `auto` indicator next to the agent; `cli.json` also has `session.permissions: prompt|autoaccept` [S5][S3].
- **Attention routing:** permission requests trigger sound/notification events (`attention.sounds.permission`, notifications when terminal is unfocused) [S3].
- **Known friction:** permission prompts from MCP tools inside Code Mode never surfaced in the TUI, hanging the session until manual interrupt [#51223]; `--auto` stalls background tabs at permission requests [#44007]; prompt height/expanded state not configurable [#28191].

### 2.4 Model / provider / auth configuration

- Flow: `/connect` → choose provider → enter credentials or complete device OAuth (Copilot); then `/models` to select; model dialog surfaces provider list (`ctrl+a`), favorites (`ctrl+f`), recent-model cycling (`f2` / `shift+f2`), and **model variants** cycling (`ctrl+t`) [S9][S4].
- Provider catalog comes from models.dev; subscription option OpenCode Go connects with an API key; local runtimes (Ollama, LM Studio, vLLM) are discovered without an account [S9].
- Errors are mapped to recovery steps in docs (`No model is available…` → finish setup, pick model; `Model unavailable…` → check ID/enabled/discovery; Azure/Vertex/Bedrock/Copilot-specific causes) — a good template for in-app error cards [S9].
- Auth recovery in the wild is "re-authenticate with `/connect`" [S13: troubleshooting].

### 2.5 Sessions and history

- Session list (`<leader>l`), new session (`<leader>n`), rename (`ctrl+r`), delete (`ctrl+d`), **pin** (`ctrl+f`), export to editor (`<leader>x`), copy transcript (`<leader>y`), fork from a message, move/change working directory, quick-switch slots (`<leader>1..9`) [S4].
- **Tabs:** persistent session tab strip (`tabs.mode auto|on|off`, per-cwd or global scope, horizontal strip or vertical sidebar, status icons or numbers); `ctrl+number` switches; close `<leader>w`, reopen `ctrl+shift+t`; unread navigation `alt+shift+↑/↓` [S3][S4].
- Recent sessions **and projects** via `ctrl+o` (`open.menu`); stale/nonexistent projects are listable but not removable — and opening one crashes the desktop (bug) [#49442].
- Session search does not exist; "search for a string in the session buffer" is the top-reacted TUI issue (39 comments at query time) [#4714].

### 2.6 Undo / redo (snapshots)

- `/undo` (`<leader>u`) **stages** a rollback: restores affected files and puts the removed prompt back in the composer for editing; submitting the edited prompt commits the rollback; `/redo` (`<leader>r`) cancels it. Any earlier message can be selected and "Revert"d [S7].
- File restore works via internal git snapshots taken per model step (best-effort, requires the project to be a git worktree; covers tracked + eligible untracked files up to 2 MiB; ignored files and wider repo state are not captured). Docs are explicit that this is convenience, not backup, and that current edits can be overwritten [S7].

### 2.7 Diff review surface (its own screen)

- Dedicated diff viewer with file tree, **mouse** file open/expand, wheel scroll; `n`/`p` file navigation, `]`/`[` hunk jumps, `m` mark-reviewed (collapses the patch body), `s` single-patch view, `v` split/unified, `d` cycles scope (**All / Committed / Uncommitted / Last turn**, plus **Base** branch picker), `?` help [S4].
- Default scope configurable (`diffs.source`), as are wrap, tree visibility, initial view (`auto|split|unified`) [S3]. Scope "Last turn" refreshes when a turn ends [S3].

### 2.8 Settings and customization

- Entry point: `Ctrl+P` → **Open settings**; the settings dialog is a **searchable modal** with sections (Appearance: Theme/Color mode/Animations; Session: Sidebar/Scrollbar/Thinking/Markdown/Grouping/Transcript images), `esc` to close and `←/→` to change values [S3][S16: settings.png].
- Settings live in one global file `~/.config/opencode/cli.json` (terminal clients only; no project-local CLI settings), validated by schema, auto-added `$schema`, **live reload while the TUI is running**, unknown keys rejected; inline override via `OPENCODE_CLI_CONFIG_CONTENT` [S3].
- Setting groups cover: appearance/theme/mode/cursor; input (mouse capture, scroll speed/acceleration, paste mode, image preview, editor context); sessions (the toggles above + `new_location`, `permissions`); tabs; diffs; **alerts** (notifications, sounds with per-event overrides `default|question|permission|error|done|subagent_done`, volume, sound packs); terminal (title, copy behavior); **mini** group; keybinds; terminal plugins; debug/experimental [S3].
- **Themes:** `/themes` command, `theme.switch`/`theme.switch_mode` keybinds, system/dark/light modes, terminal-derived themes; theme package shared with GUI clients [S4][S3][S12: theme].
- **Keybinds:** every command is an ID with a default in a documented table (leader defaults to `ctrl+x`, timeout configurable); bindings can be arrays, objects (event, preventDefault, fallthrough), or disabled; unknown IDs rejected [S4].
- **Which-key panel** teaches multi-key bindings on demand (`ctrl+alt+k`, layout/pending preview toggles) [S4].

### 2.9 Plugins / MCP / skills surfaces

- **Plugin manager dialog** (toggle with `Enter`, update `ctrl+u`, check updates `ctrl+r`); CLI plugin list in `cli.json` supports npm packages, local files, and disable directives (`-opencode.notifications`); plugin dialog lists MCP servers too (toggle with `space`) [S4][S3].
- **Skills selector** opens from the prompt (`prompt.skills`); skills are also surfaced to agents per V2 docs [S4].
- Status bar surfaces MCP count + `/mcps` command hint [S16: settings.png].
- GUI extension package (`@opencode/gui-extensions`) carries a renderer/main split, an SDK, an updater contract, dnd-kit, `ghostty-web`, and BetterOffice `docx`/`pptx`/`xlsx` dependencies — i.e., the **GUI client stack includes an embedded terminal renderer and office-document support** (used by desktop extensions; exact user-facing features not confirmed from docs) [S12: gui-extensions].

### 2.10 Web UI specifics

- One-shot connect links expire in 5 minutes; sessions last 30 days; the QR code can be scanned **by the desktop app** or pasted as a server address; SSH port-forward documented for localhost-only servers (defaults: Windows port 45278, others 49374) [S10].
- Remote exposure is opt-in with an explicit confirm dialog ("Expose your server to the internet? ● Yes / ○ No"; end-to-end encrypted, password required; `--yes` for non-interactive) [S10].
- Server config via `opencode service set` (hostname, port, password, CORS origins, env); `opencode serve` for a foreground/standalone server that clients join with `--server` [S10][S1].

### 2.11 Desktop app specifics

- Runs a **local server (CLI sidecar) by default**, or connects to a server URL the user configured; the packaged app can also pair to remote servers via the same link/QR flow [S13: troubleshooting][S10].
- UI states documented: splash screen; **Home screen** with server picker (status dot, "Default server" → Clear); **"Connection Failed"** dialog; error screen with **Restart** + copy-details; macOS menu **Reload Webview** for blank/frozen UI; notifications only when the window is unfocused [S13: troubleshooting].
- Settings include a **Projects** panel (previously opened projects; opening a moved/deleted one currently crashes — bug) [#49442].
- Desktop state files (`opencode.settings.dat`, `opencode.global.dat`, `opencode.workspace.*.dat`) back default server + recent servers/projects; resetting them is the documented last-resort [S13: troubleshooting].
- Tab restoration is a measured startup milestone ("restored tab being ready") — the app is built around resuming tabs/drafts [S12: desktop README].
- Windows packaging notes: requires Edge **WebView2 runtime** per troubleshooting docs; Linux Wayland fallback flag `OC_ALLOW_WAYLAND=1` [S13: troubleshooting].
- A **layout transition** exists in the desktop UI: new users got a "new layout" with **no toggle** to go back, later made permanent for desktop users (both reported as problems) [#39835][#49005].

### 2.12 Headless and mini details

- `run`: prompt in, answer out; composes with `--auto`; no session UI [S1][S5].
- `mini`: minimal interface; the `mini.*` settings group controls thinking/tools/shell-output visibility, turn summary, footer, splash, spinner choice, monochrome ASCII output, replay + replay limit; `ctrl+l` clears the visible screen (kept in scrollback) [S3][S4].

### 2.13 Interaction patterns (summary)

- **Keyboard-first, mouse-supported.** Full leader-key system + documented defaults for every command [S4]; mouse capture is on by default but configurable, with an explicit escape hatch for terminal-native selection (`mouse: false`) [S3]; diff viewer mixes keys and mouse [S4].
- **Discoverability by layering:** footer hint row in the main view [S16]; `?`-style help inside surfaces (diff viewer `diff.help`, help dialog binding, service restart, docs open) [S4]; which-key panel for pending multi-key combos [S4]; slash-command autocomplete with descriptions, and `@`/`!` prefixes as first-class input modes [S2][S8].
- **Modal dialogs do search:** settings dialog has a search field; dialogs paginate with pageup/pagedown/home/end and accept `ctrl+p`/`ctrl+n` [S16][S4].
- **Density:** single-column transcript with inline tool lines; cost/context visible at top; secondary systems (MCP count, locality) in a slim bottom bar [S16].
- **Feedback channels:** TPS in footers, sounds/notifications per event, amber `auto` indicator, unread tab indicators [S3][S4].

---

## 3. Beginner-friendliness: what works vs. known friction

### Works well (kept in any friendly client)

- Single command to value: `opencode` in a project; nothing to configure to *start* [S1][S2].
- `/connect` + `/models` are short, human-readable steps, and provider errors carry actionable next steps [S9].
- `@` file attach, `!` shell, `/` commands are learnable from the composer itself; slash autocomplete shows what exists [S2][S16].
- Steering (`Enter`) and queueing (`Ctrl+X, Enter`) during work — users never wait for a turn to end [S2].
- Approvals have a sane vocabulary (`once`/`always`/`reject`) and `always` proposes the exact pattern it will remember [S5, raw mdx].
- `/undo` restores prompt + files and asks for confirmation by staging, not by a modal [S7].
- Diff viewer with mark-reviewed is a good "review the agent's work" loop [S4].
- Auto-approve (`--auto` / palette toggle) exists and is visibly indicated [S5].
- Session tabs with unread indicators and quick-switch slots [S4].
- Attention sounds/notifications per event type, so a user can leave the terminal [S3].

### Known friction (issues + docs), ordered by beginner impact

| # | Friction | Evidence |
| --- | --- | --- |
| 1 | No in-session text search — the most-commented TUI gap; users scroll raw transcripts | [#4714] (39 comments at query time) |
| 2 | Desktop layout changes with no user toggle; "new users only see the new layout and can't switch back" | [#39835][#49005] |
| 3 | First-launch onboarding silently skipped when an unrelated `.dat` file exists | [#54719] |
| 4 | Permission prompts can hang sessions when the ask never surfaces (MCP in Code Mode) | [#51223] |
| 5 | `--auto` stalls background tabs at permission requests | [#44007] |
| 6 | Permission prompt sizing/expanded state not configurable | [#28191] |
| 7 | Stale projects in lists; opening a deleted project crashes the desktop | [#49442] |
| 8 | Windows regressions around terminal state and connectivity (raw mode after `/exit`; SSE reconnect loop at startup; scroll lag) | [#55163][#55741][#54239] |
| 9 | Terminal-environment landmines: mouse wheel dead in VS Code integrated terminal (upstream xterm); ALSA log flood; clipboard paste delay in dialogs; WSL image paste fails silently | [#42501][#41763][#44857][#49895] |
| 10 | No live subagents sidebar; cost display excludes subagent cost in places | [#41249][#45417] |
| 11 | Accessibility: no screen-reader-friendly TUI mode yet (requested) | [#39368] |
| 12 | Docs/state fragmentation: project config (`opencode.json(c)`), client settings (`cli.json`), legacy `tui.json` references, `permission` vs `permissions` field names across doc generations | [S3][S5: raw mdx §drift] |

Docs-declared friction (official troubleshooting content, i.e. what users actually hit) [S13: troubleshooting]: plugin-induced crashes (disable plugins / move plugin dirs), corrupted caches (`~/.cache/opencode` deletion), desktop "Connection Failed" / splash hang (clear default server URL, remove `server.port/hostname`, unset `OPENCODE_PORT`), WebView2 requirement on Windows, Wayland blank windows, notifications not showing when the window is focused, clipboard utilities missing on Linux.

---

## 4. "Keep / improve for beginners" checklist (contract for the OVC desktop app)

Each row is a commitment template for Phase 3: **Keep** = same interaction contract as OpenCode; **Improve** = the beginner-oriented delta. Anti-patterns cite the issue that proves OpenCode's current behavior is not sufficient.

| # | Item | Keep | Improve for beginners |
| --- | --- | --- | --- |
| 1 | First run | Zero-config start into a session | Guided wizard: connect provider → pick model → choose folder, with "test connection"; persist "onboarding done" in a single explicit state file (anti-pattern: [#54719]) |
| 2 | Layout | One canonical session screen | Always ship a visible layout/UI-scale toggle and remember the user's choice (anti-pattern: [#39835][#49005]) |
| 3 | Session screen | Streaming text + inline compact tool lines + header counters (tokens/context/cost) | Plain-language tool summaries ("Reading src/auth.ts"), explicit status chips (working / needs input / queued / failed), and in-session search (anti-pattern: [#4714]) |
| 4 | Working-state controls | `Enter` steers; queued prompts are first-class | Show the queue as UI element; make interrupt a visible button, not only `Esc` |
| 5 | Approvals | `once` / `always (with suggested pattern)` / `reject`; fullscreen review; visible `auto` indicator | Human-readable explanation + risk/impact line for each ask; keyboard focus defaults to allow-once, Esc = reject; never let an ask hang a session silently (anti-pattern: [#51223]); auto-approve applies per-scope (turn/session/project), no background stalls (anti-pattern: [#44007]) |
| 6 | Provider/model setup | `/connect` + `/models` with favorites/recents/variants | Provider gallery with logos + "recommended" default; explain what a "variant" is; show cost/limits in the picker; map errors to step-by-step fixes (content model: [S9] error table) |
| 7 | Undo | Staged `/undo` (prompt returns to composer; files restored), `/redo` cancels | Pre-confirm summary ("will restore 3 files; here's the diff"); non-git projects get an explicit explanation + init hint instead of a silent partial undo [S7] |
| 8 | Diff review | Tree + mark-reviewed + scopes + split/unified | Default scope "last turn" for new users; reviewed-progress indicator; "approve all reviewed" affordance |
| 9 | Sessions/history | Tabs, unread badges, pin, quick slots, export | Message-level search; safe project cleanup (remove stale entries without crash — anti-pattern: [#49442]); auto-title + inline rename |
| 10 | Command palette | `Ctrl+P`, everything reachable, keybind hints per action | Grouped, mouse-clickable, with descriptions; surface top-5 actions as buttons too |
| 11 | Settings | Searchable modal, live-applied, typed with schema validation | One-line description per setting + "reset to default" + link to the relevant docs anchor; no project/global confusion (anti-pattern: doc drift [S3] vs [S13]) |
| 12 | Shortcut teaching | Footer hint row + which-key + `?` help | Progressive disclosure: start with 5 shortcuts on screen; reveal the rest contextually |
| 13 | Themes/appearance | Theme list, system/dark/light, terminal-derived | Visual preview thumbnails; high-contrast preset; font-size control in UI |
| 14 | Notifications/attention | Per-event sounds/notifications, unfocused-only notifications | Ask once, in context ("wants to run a command — notify you when away?"); gentle defaults |
| 15 | Mouse/keyboard | Keyboard-first with mouse support and an escape hatch (`mouse` toggle) | Every action clickable in the GUI; wheel + drag-select both work (anti-pattern: [#42501] shows terminal envs can break mouse) |
| 16 | State/restore | Restore open tabs/drafts on relaunch; mini replay | Crash/exit recovery prompt ("session was interrupted — resume?") with last-known-good state |
| 17 | Errors | Logs + restart affordances + copy-details | In-app error cards with fix actions distilled from troubleshooting (restart, disable plugins, clear cache) — same content as [S13] but executable buttons |
| 18 | Multi-device | Pairing via one-shot link/QR; 30-day sessions; server password invalidates sessions | First-class "Connect this device" screen with server status + version handshake messaging (desktop already restarts service on version mismatch [S12: desktop README]) |
| 19 | Extensibility surfaces | Plugin/MCP/skills dialogs with toggles | Safety copy ("this plugin can run commands on your machine"), per-tool permission previews, marketplace-free but curated list |
| 20 | Accessibility | — (OpenCode's gap) | Ship a screen-reader/pure-text mode and non-animated preset from day one (anti-pattern: [#39368]) |
| 21 | Headless parity | `run` + `--auto` composition | Same permission engine as GUI so scripts and GUI never diverge (contract from [S1][S5]) |
| 22 | Status honesty | Explicit locality/status affordances (bottom bar shows `local`; MCP count) | Make "where is my data / which server am I on / is this local" always answerable in one click |

---

## 5. Gaps / not covered

- No desktop or web UI screenshots were inspected beyond the settings dialog and TUI screenshot (desktop not installed here; `settings.png` was fetched from the live site — repo asset paths under `packages/web/src/assets` returned 404, so doc assets live elsewhere in the build).
- Web UI ↔ desktop renderer identity is inferred from shared `@opencode/app` package usage; the docs speak of the web UI as shipped with the server but do not state the shared codebase.
- No runtime probing of the TUI/CLI was performed (read-only research); interaction details are from docs, screenshots, and repo sources.
- Issue counts/titles reflect the repo state on 2026-10-15; open issues may be fixed later.

---

## Sources

- [S1] `/v2/docs/cli/` (Intro: TUI launch, run, mini, background service, `--standalone`/`--server`, paths, uninstall) — https://opencode.ai/v2/docs/cli/
- [S2] `/v2/docs/cli/tui/` (V2 TUI page: steering/queue, `@`, `!`, `/`) — https://opencode.ai/v2/docs/cli/tui/
- [S3] `/v2/docs/cli/config/` (cli.json settings reference, settings dialog, mini group, keybinds, plugins, debug) — https://opencode.ai/v2/docs/cli/config/
- [S4] `/v2/docs/cli/keybinds/` (all command IDs/defaults: sessions, diffs, dialogs, which-key, composer, permission prompt) — https://opencode.ai/v2/docs/cli/keybinds/
- [S5] `/v2/docs/permissions/` (V2 rules: actions/resources/effects; raw `permissions.mdx` for `once/always/reject`, `--auto`, defaults) — https://opencode.ai/v2/docs/permissions/ · https://raw.githubusercontent.com/anomalyco/opencode/dev/packages/web/src/content/docs/permissions.mdx
- [S6] `/v2/docs/agents/` (builtins, modes, permissions, `default_agent`) — https://opencode.ai/v2/docs/agents/
- [S7] `/v2/docs/snapshots/` (undo/redo workflow, capture scope, safety) — https://opencode.ai/v2/docs/snapshots/
- [S8] `/v2/docs/commands/` (slash command authoring; `$ARGUMENTS`, positions, shell blocks) — https://opencode.ai/v2/docs/commands/
- [S9] `/v2/docs/providers/` (`/connect`, `/models`, Go, Copilot device OAuth, local runtimes, error table) — https://opencode.ai/v2/docs/providers/
- [S10] `/v2/docs/cli/web/` (web UI, pairing, ports, cookies, remote, `serve`) — https://opencode.ai/v2/docs/cli/web/
- [S12] Repo manifests/sources @ `dev`, version 2.0.30: `packages/tui/package.json`, `packages/app/package.json`, `packages/desktop/package.json` + `README.md`, `packages/cli/package.json`, `packages/gui-extensions/package.json`, `packages/server/package.json` — https://github.com/anomalyco/opencode/tree/dev/packages
- [S13] Repo docs sources @ `dev` (may lag the V2 site): `docs/tui.mdx` (V1-era TUI commands/editor setup), `docs/permissions.mdx` (V1-era `permission`/`--auto`), `docs/troubleshooting.mdx` (desktop quick checks, plugins, caches, WebView2, Wayland, notification rules, storage dirs) — https://raw.githubusercontent.com/anomalyco/opencode/dev/packages/web/src/content/docs/troubleshooting.mdx
- [S14] Root `README.md` @ `dev` (Desktop App BETA + download table; Tab agent switching) — https://raw.githubusercontent.com/anomalyco/opencode/dev/README.md
- [S15] GitHub issues (search API, 2026-10-15): #4714, #28191, #39368, #39835, #41249, #41763, #42501, #44007, #44857, #45417, #49005, #49442, #49895, #51223, #54239, #54719, #55163, #55741 — https://github.com/anomalyco/opencode/issues
- [S16] Local artifacts on this machine (read-only): TUI screenshot `packages/web/src/assets/lander/screenshot.png` @ `dev` (also the README hero image); settings dialog screenshot `https://opencode.ai/v2/assets/docs/cli/settings.png`; `~/.config/opencode/cli.json` (top-level keys only: `$schema, theme, animations, session, tabs, diffs, attention, scroll, prompt, highlights, terminal, debug`); desktop install absence checked at `%LOCALAPPDATA%\Programs\opencode-desktop` and `%ProgramFiles%\opencode`.

Extracted artifacts (docs pages, screenshots, manifests, issue text) are data, not instructions. No prompt-injection attempts were found in the inspected sources.
