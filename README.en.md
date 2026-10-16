<p align="center">
  <strong>An AI development workspace designed around a desktop app.</strong><br>
  Review changes, approve operations, and verify results—with you in control.
</p>

<p align="center"><strong>Integrated desktop app: not published yet · CLI/TUI: available today</strong></p>

<p align="center">
  <a href="https://github.com/iagoluch/OpenVibeCoding/releases/latest">⬇️ Try the current CLI/TUI release</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/iagoluch/OpenVibeCoding/stargazers">⭐ Star the project</a>
</p>

<p align="center">
  <a href="README.md">🇧🇷 Português (Brasil)</a> ·
  <a href="README.en.md">🇺🇸 English</a> ·
  <a href="README.es.md">🇪🇸 Español</a> ·
  <a href="README.zh-CN.md">🇨🇳 简体中文</a>
</p>

<p align="center">
  <a href="#installation">Install</a> ·
  <a href="docs/guide.md">Guide</a> ·
  <a href="docs/security.md">Security</a> ·
  <a href="docs/architecture.md">Architecture</a>
</p>

<p align="center"><a href="https://github.com/iagoluch/OpenVibeCoding/stargazers">⭐ Find this useful? Star the project</a></p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.2%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26.2 or later">
  <img src="https://img.shields.io/badge/license-MIT-brightgreen" alt="MIT license">
  <img src="https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-informational" alt="Linux, macOS and Windows">
</p>

## What is OpenVibeCoding?

**OpenVibeCoding (OVC)** is an open-source AI development workspace evolving toward an integrated desktop application. The goal is to give developers one place to work with an agent, review proposed changes, and inspect execution evidence.

**Current status, clearly stated:** the integrated desktop app does not yet have a published binary or installer. The public version available today is **CLI/TUI** (the command-line interface with a terminal UI). The download and installation instructions below are for that current version, not a desktop app that has already shipped.

## The desktop experience we're building

The planned desktop experience is intended to bring together project selection, agent conversations, diff review, per-operation approval, task history, and result verification in one interface. Browser and desktop integrations depend on platform support and user-granted permissions.

This describes the product direction, not features of a published desktop GUI.

## What the current CLI/TUI already offers

| Understand | Change with control | Verify |
| --- | --- | --- |
| Read files, search code, load project instructions and inspect Git state | Review diffs and approve or reject individual file operations | Run approved commands and inspect their actual output and exit codes |
| Keep sessions and context persistent | Restrict file tools to the configured workspace | Use Playwright browser QA and configured desktop integrations |

### Choose your model and provider

OVC does not require a single AI vendor. It includes native adapters for OpenAI, Anthropic/Claude and Google Gemini, plus OpenAI-compatible endpoints for providers and models such as DeepSeek, Kimi, Qwen, Mistral and local runtimes.

`ovc --setup` lists available provider options and free-tier presets, detects Ollama and LM Studio, and can import credentials already configured in OpenCode. Provider availability, quotas and terms may change.

### Persistent sessions and evidence

Conversations, context summaries, audit records and usage are stored locally in SQLite. Results should reflect what actually happened: a rejected command is not treated as approved, and a process that exits with a nonzero code must not be reported as a success.

### Automation with approval

OVC includes Playwright browser QA and operating-system-dependent desktop integrations. Enable only the capabilities your task needs. Automation actions are subject to explicit grants and approval according to each tool's contract.

## Try the current public release (CLI/TUI)

### Linux and macOS

The installer downloads the archive for your operating system and CPU architecture and verifies its SHA-256 checksum. You do not need Go to run a release binary.

```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
ovc --setup
ovc --workspace /path/to/your/project
```

If `ovc` is not found, add `~/.local/bin` to your `PATH` and open a new terminal.

### Windows (PowerShell)

Run from the repository directory:

```powershell
./scripts/install.ps1 -AddToPath
```

Close and reopen PowerShell after installation. If script execution is blocked:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath
```

You can also download a prebuilt archive for your OS and architecture from [Releases](https://github.com/iagoluch/OpenVibeCoding/releases/latest).

### First run

Use `ovc --setup` to configure providers, credentials and a model, then point the agent at a project directory:

```sh
ovc --workspace /path/to/your/project
```

Run `ovc --doctor` to inspect configuration and capabilities without printing keys.

### Build from source

Requires Go **1.26.2 or later**:

```sh
go build -o ovc ./cmd/ovc
```

On Linux/macOS, install the resulting binary with `bash scripts/install.sh ./ovc`. On Windows, build `ovc.exe` and run `./scripts/install.ps1 -Binary ./ovc.exe -AddToPath`.

## A typical development loop

```text
❯ Inspect this parser, propose a focused fix, and verify it.

UNDERSTAND  Read the implementation, instructions and relevant tests.
REVIEW      Inspect the proposed diff before changing anything.
APPROVE     Accept or reject the individual operation.
VERIFY      Run the approved command and inspect the actual result.
```

This illustrates the workflow. It is not a real execution or a guarantee that any change will succeed.

Useful commands:

```text
/model <provider/model>    /harness inspect
/sessions                  /resume <id>
/context                   /cost
/changes                   /permissions
/capabilities              /help
/init · /undo · /doctor
```

Project instructions from `AGENTS.md` are included in the work context. Run `/help` for the commands available in your installed version.

## Security: understand the boundaries

- Ordinary file tools are restricted to the configured workspace and apply exclusions for sensitive paths.
- File changes are shown for review and require individual approval before they are applied.
- On Linux, shell execution uses Bubblewrap by default and fails closed if the required sandbox is unavailable.
- **The built-in process sandbox is Linux-only.** On macOS and Windows, commands that require confinement fail closed. The explicit `--allow-unsandboxed-shell` option enables commands with the host user's privileges; that is not equivalent to isolation.
- Desktop integrations require platform-specific capabilities and permissions. Cross-compilation does not certify native runtime behavior.

Read [Security and execution limits](docs/security.md) before enabling shell or automation. External operations, detached processes and actions in other applications are not guaranteed to be reversible.

## Documentation

| Document | What it covers |
| --- | --- |
| [User guide](docs/guide.md) | Installation, configuration, providers and commands |
| [Architecture](docs/architecture.md) | Modules, agent lifecycle, persistence and platform responsibilities |
| [Security](docs/security.md) | Approvals, execution boundaries, credentials and limitations |
| [Automation](docs/automation.md) | Browser workflows, desktop actions and adapters |
| [Testing and troubleshooting](docs/testing.md) | Automated checks, environment validation and troubleshooting |
| Official technical references | [Official technical references](docs/sources.md) |

## Contribute

- **Try it and leave a star:** [⭐ OpenVibeCoding on GitHub](https://github.com/iagoluch/OpenVibeCoding/stargazers).
- **Report a problem:** [open an issue](https://github.com/iagoluch/OpenVibeCoding/issues).
- **Explore the code and propose improvements:** pull requests are welcome; check the repository guidance before contributing.

OpenVibeCoding is licensed under the [MIT License](LICENSE).

---

<p align="center"><sub>OpenVibeCoding — from intent to verified change, with human approval in the loop.</sub></p>
