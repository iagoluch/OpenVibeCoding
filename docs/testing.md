# Validation and troubleshooting

## Automated checks

```sh
go test ./...
go test -race ./...
go vet ./...
bash scripts/package.sh
```

Coverage includes native and compatible streaming protocols, tool fragments/usage/errors/redirects, endpoint validation, credential fallback permissions/deletion, SQLite persistence/recovery, incremental summary preservation and non-repetition, remote approval-before-write, runtime authentication/duplicate-instance prevention/session exclusion/export, ignore/path/symlink boundaries, stale diffs, creation/move/delete, command exit status/cancellation, slash validation and UI shutdown.

Environment integration tests require explicit fixtures:

```sh
# First install browser prerequisites with ovc --setup-browser.
OPENVIBECODING_TEST_PLAYWRIGHT=/absolute/app-data/qa/node_modules/playwright go test ./internal/automation -run TestRealBrowserWorkflow
OPENVIBECODING_TEST_SANDBOX=1 go test ./internal/workspace -run TestSandboxBoundary
# Only point this at an isolated test desktop, never your normal work session:
DISPLAY=:93 OPENVIBECODING_TEST_DESKTOP=1 go test ./internal/automation -run TestRealDesktop
```

The real-browser test fills a synthetic form, clicks, checks the visible result, captures an image and verifies unauthorized-origin denial. The real-desktop test moves/clicks/scrolls, observes pointer position, captures a screenshot and types text into a synthetic xterm process, verifying received input via a fixture file. Xvfb needs `-noreset` so the server does not reset pointer state between tool processes. Native desktop tests are opt-in because they generate input.

The sandbox test verifies access to the selected project, denial of an external fixture file, masking `.env`, and an observed write inside the workspace. Host namespace policy can prevent Bubblewrap; this must be recorded as unavailable rather than silently falling back to host execution.

## Local evidence

During development, all packages passed unit/integration tests and the race detector. Real Linux browser, Bubblewrap and X11 tests passed. Two terminal clients attached to the same daemon; one closed while the other continued listing shared sessions. Protocol fixtures covered OpenAI, Anthropic and Gemini. Archive generation compiled six targets with the `ovc` launcher.

The first X11 fixture run failed because Xvfb reset after clients disconnected; `-noreset` fixed the fixture and the actual mouse/keyboard test passed. Playwright reports Amazon Linux as an unsupported distribution and downloads its Ubuntu fallback build; the actual Chromium workflow ran successfully in this sandbox.

No paid external-provider request or native macOS/Windows GUI/keychain execution is claimed. CodeRabbit emulate v0.0.1's catalog was inspected and has no matching generation API emulator. Those provider adapters were tested with synthetic HTTP/SSE responses instead. CI is configured for native OS tests, but its remote execution is not implied by local results.

## Live provider acceptance

1. Create a disposable project with a README and synthetic data. Store credentials using `ovc --credential <provider>`; do not paste them into the project/session.
2. Check `--doctor`; launch with `--workspace` and select `/model provider/model`.
3. Ask to inspect the README and create `hello.txt`. Reject once and verify absence; approve a new proposal and inspect the actual file.
4. Approve a harmless test command; verify output and a deliberate nonzero exit code are reported accurately. Use isolation if available; host mode requires explicit opt-in.
5. Exit/resume and verify transcript, `/changes` and saved `/cost`.
6. Repeat with Anthropic/Gemini and a compatible endpoint. Confirm native tool responses continue coherently after resumption/model switching.
7. For image-capable models, enable browser/desktop, approve a screenshot and verify the model refers to observed evidence. Resume and confirm it requests fresh capture rather than inventing old image details.

## Troubleshooting

- **Runtime startup:** run `ovc --runtime serve` in a terminal to see setup errors. `--runtime status` checks compatible authentication. Do not delete an active lock file; ownership uses an OS lock, not file existence. After a crash, reconnect normally.
- **Changed credentials/config:** stop the idle runtime and restart it. Model/harness/grant slash switches apply on the next turn immediately.
- **Keychain unavailable:** unlock/configure the OS credential service or explicitly choose `--credential-file`. Linux requires a Secret Service session. Do not weaken file permissions.
- **Provider HTTP/protocol errors:** verify account, endpoint, model and protocol; error bodies are omitted intentionally. Disable `stream_usage` for compatible providers that lack it. Text-only models should set `tools:false` and should not be used for screenshot workflows.
- **Context preparation failed:** originals remain saved. Review/export the session and begin a smaller task if the current turn itself exceeds the budget. Invalid model summaries are rejected.
- **Browser unavailable:** run `--setup-browser`; verify Node/npm and Chromium system libraries. Downloads require network access. Allowed origins must include required test-site subresources.
- **Desktop unavailable:** inspect `/capabilities`, DISPLAY/OS session and user-granted permissions. X11 needs xdotool/ImageMagick/xclip. Windows must have an interactive desktop and allow the fixed PowerShell script under its existing execution policy. macOS requires Accessibility/Screen Recording.
- **Cancellation:** Esc/Ctrl+C cancels the action/worker. Inspect the target afterward; completed clicks, submissions, typed text or external effects cannot be undone by cancellation.

## Backup

For individual idle sessions, use `ovc --export ID > session.json`. For a full restorable backup, stop the runtime and copy the entire application directory (including SQLite auxiliary files if present). Restore into a separate `OPENVIBECODING_HOME` first and inspect it. Never copy a live SQLite database file alone.
