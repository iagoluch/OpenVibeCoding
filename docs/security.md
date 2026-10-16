# Security and execution guarantees

[Português](pt-BR/seguranca.md) · [English](security.md) · [Español](es/seguridad.md) · [简体中文](zh-CN/security.md)

## Workspace file tools

`os.Root` provides traversal-resistant file access. Absolute/parent paths and symlink components are rejected. Secret locations (`.env*`, `.ssh`, `.aws`, credential filenames and private-key extensions), generated directories and `.gitignore` matches are excluded from ordinary read/search/list tools. Reads accept regular UTF-8 text up to 128 KiB. This is a denylist and ignore policy, not a general secret detector. Nested ignore handling is deliberately conservative: an ancestor exclusion cannot be reopened by a nested exception.

A proposal does not write. Each replacement/deletion shows a unified diff; each move names its source/destination and shows the affected content. Approval applies to one operation. The original file digest is rechecked before mutation. Replacements use exclusive temporary files, fsync and rename; moves create the destination exclusively before removing the original. If a move cannot remove its source, it reports the partial result rather than claiming success. Parent directories may be created inside the root. Unrelated destination files are never overwritten by a move.

The boundary protects against model-supplied paths, not a hostile concurrent local process, hard links or mounted filesystems controlled by the same OS user. Digest checking and replacement are not a filesystem-wide compare-and-swap; avoid concurrent external edits during application. Per-workspace runtime exclusion prevents two agent turns from modifying the same workspace concurrently.

## Linux process isolation

Default shell execution uses Bubblewrap and fails closed if unavailable or denied by host policy. The sandbox exposes `/usr` and required system runtime directories read-only, an ephemeral `/tmp`, limited devices, and the selected project at `/workspace`. Host home and `/proc` are absent. Protected secret entries inside the project are masked; dependency directories remain available so builds can use installed dependencies. It requests separate namespaces, a new session and parent-death termination. It is not claimed to mitigate kernel vulnerabilities or side channels.

Network is isolated by default. An explicit network grant shares the host network and mounts required resolver/certificate files; the approval labels network-enabled execution. An explicit install grant allows the package-install tool. General shell code can perform equivalent operations inside its allowed filesystem/network boundary, so tool-category grants alone are not semantic enforcement of arbitrary programs.

There is currently no built-in process sandbox on macOS or Windows. Confined command requests fail closed there. `--allow-unsandboxed-shell` deliberately opts into host-user execution on any platform, with per-command approval and an explicit warning in the review. It can access host paths, network and installed applications. Never enable host mode when a stronger OS boundary is required. OS privileges are never escalated automatically.

Children receive a small environment allowlist instead of provider credentials. Host mode can nevertheless read credentials from host files. Commands have a sixty-second limit and bounded stdout/stderr. Unix host-shell cancellation kills its process group; automation workers also use process groups. Windows uses process-tree taskkill on cancellation. Intentionally detached processes and completed actions are not guaranteed reversible or terminable. These are cancellation mechanisms, not OS isolation on Windows/macOS.

## Runtime

A random 256-bit token authenticates loopback IPC. The token file is private, is never displayed, and clients reject non-loopback addresses/redirects. Requests are bounded, methods/routes validated, and cross-origin browser requests rejected. OS locks prevent duplicate daemons and release automatically after crashes. Metadata is replaced only after acquiring ownership. Protocol mismatches fail rather than silently using incompatible state.

Approval IDs are random, one-use and bound to the originating client identity. Storage mutation belongs to the runtime; direct client append/summary/usage writes are rejected. Active sessions cannot be deleted, recovered, renamed or exported. Turns are serialized by session and workspace. A disconnected client cancels its own request; unrelated clients continue. Runtime stop refuses active tasks; OS termination cancels them before storage shutdown.

The local authenticated user can configure grants and endpoints. Authentication does not make an AI-generated command trustworthy: every mutation, shell command and automation action still goes through approval, and inspect mode refuses them.

## Secrets, audit and context

Credentials come from environment overrides, OS keychain or explicitly selected file fallback. Unix fallback files with group/other access are refused. New files use 0600 and directories 0700; Windows relies on profile ACLs. No secret is included in JSON configuration. Known credential values and terminal controls are redacted before display/persistence. Arbitrary application secrets cannot be reliably detected: do not ask tools to print them. Sessions are not encrypted at rest.

SQLite records proposals/approvals before execution. Persistence failure stops the operation. Task starts/finishes and configuration changes are audited. A crash between mutation and durable result produces an unknown-outcome recovery message; it never causes automatic replay. Runtime logs contain operational metadata, not request bodies or screenshot data.

Original history is authoritative. Structured summaries retain source IDs and are not permissions. Recall retrieves originals in bounded chunks. Screenshots are transient: only a placeholder is stored, and image data is sent to the configured model during the current task. Users must authorize capture, choose an image-capable model and consider the information visible on screen.

## Browser and native desktop

Both are separately enabled and approved. Clipboard has an additional grant. The Playwright worker uses a fresh profile and an exact-origin page routing policy, blocks downloads/service workers/WebSockets, limits steps/time/output and reports actual failures. Its routing is not OS-level network isolation.

Native input acts on the user's visible desktop. Launch/focus is allowlisted; global mouse/keyboard gestures are not confined to one process. The user reviews the target/action before execution. Accessibility, Screen Recording, locked/UAC desktops and compositor restrictions are respected. Linux Wayland is unsupported by the built-in X11 backend. External adapters are trusted user configuration and must enforce their own OS policy. See automation.md for the action protocol and cancellation limits.
