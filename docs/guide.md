# OpenVibeCoding

A terminal coding agent written in Go with Bubble Tea. It can inspect projects, propose reviewed file changes, run approved commands, manage persistent task context, test websites in Chromium, and perform approved desktop actions through platform backends.

## Install and launch

Install the latest [release](https://github.com/iagoluch/OpenVibeCoding/releases/latest) with the install script — it downloads the archive for your OS/CPU and verifies SHA-256 — or grab the archive manually:

```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
# Add ~/.local/bin to PATH if needed:
export PATH="$HOME/.local/bin:$PATH"
ovc --workspace /path/to/project
```

Windows: `./scripts/install.ps1 -AddToPath`, then reopen the terminal (if your execution policy blocks scripts: `powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath`). Pin a specific release with `-Version v0.1.0` (PowerShell) or `OPENVIBECODING_VERSION=v0.1.0` (shell). Building from source needs **Go 1.26.2+**: `go build -o ovc ./cmd/ovc`, then `bash scripts/install.sh ./ovc` (`./scripts/install.ps1 -Binary ./ovc.exe -AddToPath` on Windows).

`bash scripts/package.sh` creates Linux/macOS/Windows × amd64/arm64 archives containing `ovc`, plus `SHA256SUMS`. Building needs Go; packaging also needs Python 3. Running a release binary does not need Go. Browser QA additionally needs Node and Chromium; desktop prerequisites are listed below.

## Credentials and first run

The first run writes a default configuration. `ovc --setup` is the guided path: it detects Ollama and LM Studio servers, lists free-tier providers with the page that issues each key, can import credentials already configured in OpenCode, stores the key in the OS keychain and makes one real request before activating. When the preferred model is retired, rate limited or busy, the wizard tries the provider's next model instead of failing the first run.

Store a credential directly through hidden terminal input when you already know the provider:

```sh
ovc --credential openai
ovc --credential anthropic
ovc --credential gemini
```

The default credential backend is the OS keychain: macOS Keychain, Windows Credential Manager, or Linux Secret Service. If unavailable, explicitly choose `--credential-file` to use a local file with restrictive permissions. There is no silent plaintext fallback. Environment variables such as `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` and `GEMINI_API_KEY` override saved credentials. Never paste secrets into conversations or endpoint URLs.

Use `ovc --doctor` to check configuration and capabilities without printing keys. If credentials or saved configuration change while the runtime is running, stop the idle runtime with `ovc --runtime stop`, then reconnect to load them. `/model` and `/harness` switches take effect immediately without a restart.

## Providers

| Adapter | Configuration |
| --- | --- |
| OpenAI | Native Chat Completions streaming, tools, optional usage |
| Anthropic Claude | Native Messages API streaming, tool use/results, usage |
| Google Gemini | Native streamGenerateContent, function calls/results, image evidence, thought-signature preservation |
| DeepSeek, Kimi, Qwen, Mistral | Configured OpenAI-compatible endpoints |
| Free-tier presets | OpenRouter (`:free` models), Groq, Cerebras, NVIDIA NIM, Z.AI and Google AI Studio, listed by `ovc --setup` with the key page for each |
| Local/custom providers | Register any compatible endpoint and its models; loopback HTTP is allowed, and Ollama/LM Studio are detected by `ovc --setup` |

Models, context budgets, tool capability, output limits, temperature and prices live in the provider registry. Defaults are conservative starting configurations; verify model availability and regional endpoints with your provider. Routes that are genuinely free to use (OpenRouter `:free`, NVIDIA NIM credits, Z.AI flash, local servers) report `$0`; paid platforms show list prices per million tokens; when no reliable price is known the cost is reported as unknown rather than guessed. Requests have timeouts, bounded responses, cancellation, HTTP 429/5xx retries and redacted errors. Redirects are rejected to protect credentials. HTTP is accepted only on loopback.

Provider presets refresh with each release: `config.json` stores only your own changes, so entries you never edited pick up new model IDs, contexts and prices automatically, while edited entries are preserved exactly as written.

## Development workflow

1. Ask the agent to inspect the project and plan the change. File tools respect protected names and `.gitignore` patterns, bound reads/searches, and reject paths outside the workspace.
2. Review the proposed unified diff. **Ctrl+Y** approves that operation, **Ctrl+N** rejects it. File replacement, creation, movement and deletion are supported. Existing file changes invalidate stale proposals.
3. Approve test/build commands. On supported Linux systems they run in Bubblewrap with workspace access, read-only OS runtime files, and no network by default. If isolation cannot be enforced, execution fails closed.
4. Inspect actual stdout, stderr, exit status and `/changes`. Resume the saved session later.

`--allow-unsandboxed-shell` is an explicit alternative for host commands on any platform. Every command still requires approval, but it runs with your user privileges and can access host files/network. This is not an OS sandbox. See [security guarantees](security.md).

### Controls

Enter sends; Ctrl+J or Alt+Enter inserts a newline; Alt+Up/Down recalls input; Tab completes slash commands; PgUp/PgDn scrolls. Esc or Ctrl+C cancels the current operation. Ctrl+C while idle exits the client. File review is per operation/file, with full unified diffs; hunk-level selection is not provided.

| Commands | Purpose |
| --- | --- |
| `/help`, `/status`, `/config` | Commands, session/runtime state and effective settings |
| `/model [provider/model]`, `/provider [name]` | Select registered provider/model |
| `/harness [review\|inspect]` | Review permits approved operations; inspect permits project reads only |
| `/sessions`, `/resume <id>`, `/new` | Persistent session navigation |
| `/rename <title>`, `/delete <other-session-id>` | Manage sessions |
| `/context`, `/cost`, `/changes` | Context state, saved usage/cost, observed file changes |
| `/capabilities` | Discover installed automation and sandbox prerequisites |
| `/permissions [browser\|desktop\|clipboard\|network\|install on\|off]` | Inspect or explicitly change grants for this client |
| `/clear`, `/exit` | Clear the view without deleting data; exit |

Grants do not remove operation approvals. Network and install default off. The package-install tool requires its separate grant. A permitted shell is a general programming environment: application-level tool categories do not prevent equivalent operations inside an approved shell. OS isolation enforces its filesystem/network boundary; host mode does not.

## Shared runtime and sessions

The first terminal starts one local runtime; subsequent terminals reuse it. The daemon owns SQLite, provider configuration, audit records and task coordination. IPC uses authenticated loopback HTTP on all three operating systems, with a random token in a private metadata file and an OS lock preventing duplicate daemons. Browsers cannot invoke it via cross-origin requests.

```sh
ovc --runtime start
ovc --runtime status
ovc --runtime stop
ovc --resume SESSION_ID --workspace /path/to/project
ovc --export SESSION_ID > session.json
```

Each client has independent identity and approvals. Turns are serialized per session **and workspace**. Closing one client cancels its current request and leaves other clients' tasks intact. The daemon stays available until explicitly stopped; stopping with active tasks is refused. OS termination cancels active tasks before closing storage. Stale metadata from a crash is replaced after obtaining the released OS lock. A protocol mismatch requires stopping the older runtime.

SQLite stores original messages, tool outcomes, approvals, task lifecycle, configuration changes, summaries and token/cost usage. Migrations are transactional. Recovery marks an interrupted tool outcome as uncertain and never replays a mutation automatically. Exports include session history, summary, usage and audit records; running sessions must finish or be cancelled before export. Screenshots are transient and are not stored in the session database.

Context preparation incrementally summarizes older complete exchanges into structured task memory while retaining recent messages. Source IDs support selective retrieval through `recall`, with offsets for large outputs. Original history is preserved. Token estimates are conservative byte-based estimates; exact usage comes from providers when available. Summaries can be imperfect: original evidence remains authoritative, and past approvals never authorize new work. An oversized current user task fails with a useful error rather than discarding it silently.

## Browser QA

```sh
ovc --setup-browser
# In the TUI:
/permissions browser on
```

Setup installs pinned Playwright 1.58.2 and Chromium into the app's QA directory. Review and approve a workflow specifying allowed origins and steps: `open`, `fill`, `click`, `press`, `select`, `expect_text`, `expect_url`, `screenshot`. Each workflow gets a fresh browser context; unauthorized origins, service workers, WebSockets and downloads are blocked. Results contain observed DOM text, title/URL, assertions, console messages, page errors, failed requests and optional screenshots. Screenshots can be inspected by image-capable models during the turn and are discarded afterward. Use synthetic data and dedicated test environments.

## Desktop computer use

Enable `/permissions desktop on`; clipboard needs `/permissions clipboard on` separately. Every action is reviewed. Supported actions include inspection, screenshots, pointer movement/clicks, scrolling, typing/keys, application launch/focus/close and clipboard access, subject to backend support.

| Platform | Backend and boundaries |
| --- | --- |
| Linux X11 | `xdotool`, ImageMagick `import` for screenshots, `xclip` for clipboard; requires an active `DISPLAY`. Real mouse/keyboard/screenshot tests passed in Xvfb. Wayland is explicitly unsupported by this backend. |
| macOS | AppleScript/System Events, `screencapture`, and CoreGraphics through JavaScript for Automation for pointer/scroll actions. Requires user-granted Accessibility/Screen Recording. |
| Windows | PowerShell, Win32 desktop APIs, Windows Forms input/clipboard and screen capture. Requires an interactive desktop; cannot control UAC/secure desktops or bypass OS permissions. |

`automation.applications` is a launch/focus allowlist; it is not process isolation for global mouse/keyboard actions. Review the visible target before approving input. Native gestures affect the current visible desktop. External native adapters can be configured with `automation.desktop_backend` as an executable/argument array using the documented JSON action protocol. See [automation](automation.md).

## Configuration and maintenance

See [config.example.json](../config.example.json). Data lives under the platform's `os.UserConfigDir()/openvibecoding` (`~/.config`, macOS Application Support, or Windows AppData). `OPENVIBECODING_HOME` overrides the data directory. `OPENVIBECODING_PROVIDER`/`OPENVIBECODING_MODEL` override defaults. Provider secrets are excluded from JSON configuration/export.

The directory contains `config.json`, SQLite session files, `runtime.json`/`runtime.lock`, metadata-only `runtime.log`, optional `credentials.json`, and `qa/` dependencies. Unix files/directories are restricted; Windows relies on the user's profile ACLs. `--debug` enables diagnostic runtime metadata logging without request bodies or secrets. Logs do not contain prompts/screenshots. Stop the runtime before copying the whole data directory for backup or upgrading binaries.

Upgrade by running the installer with a newer local binary or release version (`OPENVIBECODING_VERSION=v…` / PowerShell `-Version v…`). Configuration and sessions are preserved. Stop the runtime, then use `scripts/uninstall.sh` or `scripts/uninstall.ps1` to remove the launcher. Remove data separately only if you intend to erase sessions and credentials. Fish users can add PATH with `fish_add_path ~/.local/bin`; Bash/Zsh users should update their appropriate shell startup file. PowerShell installation does not change execution policy or request administrator access; if your policy blocks scripts, run `powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath`.

## Verification

See [testing and evidence](testing.md), [architecture](architecture.md), and [official references](sources.md). Local checks cover real Linux sandbox, browser and desktop behavior, provider protocol fixtures, runtime concurrency/authentication/approval, persistence and context recovery. Native macOS/Windows execution, OS keychain integration on those hosts, and billable live-provider requests still require those environments/accounts. Cross-compilation alone does not certify native behavior.
