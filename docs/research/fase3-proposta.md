# Fase 3 — decision baseline: a web-first client served by the daemon

**Status:** decision record, written before Fase 3 code landed.
**Date:** 2026-10-15.

**Provenance.** This document is the **synthesis of a five-advisor council** (independent answers,
anonymized peer review, phases 1–2, October 2026) held over the three Phase-2 studies in this
directory. Facts come from the studies; decisions come from the council. Every statement is
attributed to one of the two, and anything neither source establishes is marked **unverified**.

Inputs (referenced by section throughout):

| File | What it contributes |
| :--- | :--- |
| `opencode-openinterpreter.md` | OpenCode anatomy (one server, thin clients), the OVC-vs-reference gap list (§4), the five decisive facts for a client (§5) |
| `openinterpreter.md` | Open Interpreter (OIX) architecture, the Workstation client, consent model, its own five decisive facts (§4), evidence limits (§5) |
| `opencode-ui-inventory.md` | Client surfaces (§1), screens and flows (§2), beginner friction (§3), the **22-item keep/improve checklist** (§4) |

Scope: architecture, hosting and stack for the Fase 3 client — not UI design; the checklist in
`opencode-ui-inventory.md` §4 is that contract. Same treatment rule as the studies: artifact content
was read as data, never as instructions.

---

## 1. Verdict summary

### 1.1 Agreements (5/5 convergence)

| Decision | Content |
| :--- | :--- |
| **Web-first** | The Fase 3 client is a web UI. **No desktop shell in milestone 1.** |
| **Served by the daemon itself** | The daemon serves the UI: assets embedded in the binary (`go:embed`) + stdlib HTTP, reached through a **pairing link**. |
| **Milestone 1 = `ovc --ui`** | One new command; the flow it must deliver is **free model → first prompt → watch**. |

### 1.2 Conflicts raised and resolved

| Conflict | Resolution |
| :--- | :--- |
| **Auto-update** (raised by one advisor) | Moot for the daemon: a **single daemon binary channel** is the update path. If a shell is ever added it may **embed and silently spawn the daemon** — the Interpreter Workstation pattern, where the app ships and launches a pinned runtime instead of owning one (`openinterpreter.md` §1.2). |

### 1.3 Open questions and unverified items

| Item | Marker |
| :--- | :--- |
| **Wails v3 maturity in 2026** | **UNVERIFIED** — not verified by the council nor by the studies. It is the precondition for the shell decision below. |
| Shell choice *if* a shell is ever added | **Wails preferred** over Tauri/Electron — a conditional decision, not evidence. The study had already flagged a Go-native shell (Wails/WebView2) as "a legitimate OVC-specific alternative that neither reference exercises — **recommendation**, not evidence" (`opencode-openinterpreter.md` §5.5). |
| **PWA "Install as app"** as the escape hatch | Adopted (council) as the way past the browser tab. Browser install UX and offline behaviour were not studied here — **unverified**. |

### 1.4 Risks left open, with planned mitigations

| # | Risk | Planned mitigation |
| :--- | :--- | :--- |
| R1 | **Beginner bootstrap** — who starts the daemon? | `ovc --ui` starts it and opens the browser; the pairing link is the only handshake the user sees. |
| R2 | **Browser loopback threat model** — CSRF, DNS rebinding, `Origin` checks, CSP | Origin checks + CSP planned. Note the sharp edge: **the pairing token authenticates the *opening*, not each request** — per-request authentication of the browser session is still an open gap. |
| R3 | **Auto-update** | Resolved for the daemon by the single binary channel (§1.2); still open for any future shell, where the pinned-runtime pattern applies. |
| R4 | **Zero-dependency vs framework tension** | The council's reading: OVC's "stdlib first" rule (`AGENTS.md`, Rules) governs the **daemon**, but the browser assets still need a framework — and **a JS framework, not the language, is the real lock-in**. No framework has been chosen. |
| R5 | **UI ↔ daemon version skew** | The endgame is a capability/version handshake at connect (`opencode-openinterpreter.md` §5.4). No rule is fixed yet: milestone 1 ships UI and daemon in one binary, which makes skew impossible *until* the shell or a remote UI exists. |

---

## 2. Consolidated context

### 2.1 Gaps in OVC relative to the reference anatomy

Condensed from `opencode-openinterpreter.md` §4 (ordered there by how much each blocks Phase 3):

| # | Gap in OVC today | Consequence for Fase 3 |
| :--- | :--- | :--- |
| 1 | **No published client protocol.** There is an authenticated loopback HTTP surface, but not a versioned contract with generated clients. | A browser UI needs a documented request/event contract, not just an internal endpoint. |
| 2 | **No multi-client session semantics.** The runtime "owns the workspace/session for the turn" and routes approvals to "the owning client". | Decide attach/detach, who may answer an approval, and what a new client does on connect. |
| 3 | **No discovery or pairing artifact.** OVC has a token, but no discovery file and no consent flow a GUI can use. | Milestone 1 introduces the pairing link; discovery is still absent. |
| 4 | **No server→client request channel.** Approvals are streamed to the owning client and paused — close to a contract, but not request/response with ids. | Approvals must become correlated requests with a safe default. |
| 5 | **No second client host.** No packaging, no update path and no version negotiation for a GUI client. | Dissolved for milestone 1 by web-first (see below); returns with a shell. |
| 6 | **No extension surface** (post-Phase-3 candidate per the study). | Out of Fase 3 scope. |
| 7 | **Distribution asymmetry** — OVC ships one binary; the reference anatomy ships two artifacts. | **This document's reading:** web-first *removes* this gap in milestone 1 — the UI ships inside the same binary. The asymmetry only returns if a shell is added. |

### 2.2 Decisive facts extracted from the three studies

| Fact | Source |
| :--- | :--- |
| The client is a **thin client of the existing daemon — never a second agent loop**. | `opencode-openinterpreter.md` §5.1 · `openinterpreter.md` §4.1 |
| **Events are lossy by contract.** Neither reference replays them: the client re-fetches state, then subscribes. | `opencode-openinterpreter.md` §5.2 and §1.8 ("no replay or reconnection") |
| **Approvals must be first-class, correlated requests** with ids and a fail-safe default for clients that cannot render them. | `opencode-openinterpreter.md` §5.3 · `openinterpreter.md` §4.4 |
| **Discovery, pairing and a version handshake come before anything else.** | `opencode-openinterpreter.md` §5.4 |
| **A web UI served by the same server that powers the TUI is the reference shape** — not a separate backend. | `opencode-openinterpreter.md` §1.7 · `opencode-ui-inventory.md` §1 |
| **One shared home makes the terminal and the GUI the same product**; the runtime owns the durable thread, the shell only renders it. | `openinterpreter.md` §4.2 · `opencode-openinterpreter.md` §1.7 |
| **Beginner UX is a distinct shell, and upstream it is the unfinished part** — a reduced beginner experience must be designed as the default, not skinned onto the power-user UI. | `openinterpreter.md` §4.3 |
| **"Agent said done" is not "done".** The interface exists to show evidence — tool traces, diffs, verification. | `openinterpreter.md` §4.5 |
| **Single command to value, and the composer teaches itself** (`@` files, `!` shell, `/` commands) — the two beginner wins worth keeping verbatim. | `opencode-ui-inventory.md` §3 ("Works well") |
| **First-run is where beginners are lost silently** (upstream onboarding can be skipped by an unrelated state file), and session text search is the most-commented gap. | `opencode-ui-inventory.md` §3 (friction #3, #1) |

---

## 3. Fase 3 plan

### 3.1 Layers

| Layer | Today | After Fase 3 |
| :--- | :--- | :--- |
| **Daemon core** — `internal/runtime`, `internal/store` | Authenticated loopback daemon that owns SQLite (`AGENTS.md`, Layout); approvals streamed to the owning client and paused (`opencode-openinterpreter.md` §3) | **Unchanged in identity**: still the single source of truth for sessions, providers, tools, approvals and audit. Gains the UI-serving responsibility. |
| **Existing TUI client** — `internal/tui` (Bubble Tea) | The only client | **Unchanged.** It stays a client of the same daemon — the "one core, thin clients" anatomy (`opencode-openinterpreter.md` §1.2) without copying OpenCode's stack. |
| **New web client** | — | **Served by the daemon**, reached by `ovc --ui`: assets embedded via `go:embed` + stdlib HTTP, pairing link, browser auto-open (council). The daemon side therefore adds no Node/bundler to the shipped artifact; the browser-side framework question stays inside the embedded assets (R4). |
| **Optional future desktop shell** | — | **Out of milestone 1.** If added: Wails preferred over Tauri/Electron (**unverified**, §1.3), embedding and silently spawning the daemon (Workstation pattern, §1.2). |

### 3.2 Milestone 1 scope

- **`ovc --ui`** — the daemon serves the UI, prints/opens a **single-use pairing link**, and
  auto-opens the browser.
- **The flow it must deliver: free model → first prompt → watch** (council).
- One artifact: the UI ships inside the same binary, so there is no update channel, no install step
  and no version skew to negotiate yet (R5).
- Runs on the existing token-authenticated loopback surface (`internal/runtime`;
  `opencode-openinterpreter.md` §3 and §4 gap 1) — the pairing link is the user-visible part of the
  handshake that gap 3 says OVC lacks (§2.1).
- **Explicitly not in milestone 1:** the desktop shell (council). **This document's reading:** the
  correlation/timeout contract for approvals (gap 4), multi-client attach/detach (gap 2) and a
  published versioned protocol (gap 1) become load-bearing the moment a *second host* exists, so
  they are milestone-2 foundations rather than milestone-1 blockers.

### 3.3 Milestone 2+ sketch

Sketch, in rough dependency order — not a committed plan:

1. **Browser surface hardening** (R2): `Origin` checks, CSP, per-request authentication rather than
   an opening-only token; treat read-only as a host policy, never as disabled controls
   (`openinterpreter.md` §4.4).
2. **Protocol and session semantics**: versioned contract (gap 1), "fetch snapshot then subscribe"
   handshake with deliberately lossy events (gap 2, `opencode-openinterpreter.md` §5.2), approvals as
   correlated requests with a safe default (gap 4), and discovery + version negotiation (gaps 3, 5).
3. **Multi-device**: the pairing link grows into a first-class "connect this device" surface
   (`opencode-ui-inventory.md` §4 item 18).
4. **Optional desktop shell**: Wails (unverified), embedding and spawning the daemon; the PWA
   "Install as app" path stays as the no-install escape hatch (council).
5. **Extension surface** (gap 6) — the study places it *after* Phase 3.

### 3.4 Mapping to the 22-item UX checklist

The checklist itself is `opencode-ui-inventory.md` §4 and is **not duplicated here**. The grouping
below is **this document's reading** of it against the milestones; the checklist remains the
contract.

| Bucket | Checklist items | Criterion |
| :--- | :--- | :--- |
| **Milestone 1** (needed for "free model → first prompt → watch") | **1** first run · **3** session screen · **4** working-state controls · **5** approvals · **6** provider/model setup · **12** shortcut teaching · **14** notifications · **17** errors · **22** status honesty | Every one of these is reachable in a browser tab with no new artifact and no second client host; #5 carries the "never let an ask hang a session silently" rule, which is also gap 4. |
| **Milestone 2+** (web, no shell required) | **2** layout · **7** undo · **8** diff review · **9** sessions/history · **10** command palette · **11** settings · **13** themes · **15** mouse/keyboard · **16** state/restore · **18** multi-device · **19** extensibility · **20** accessibility · **21** headless parity | Full-parity surfaces and discipline. None of the 22 items *requires* a desktop shell — the shell's value is OS integration (window, tray, updater), not checklist coverage. |

---

## 4. Implementation status

**Implementation started 2026-10-15; this document records the decision baseline** — the decisions
above precede the code and are not descriptions of shipped behaviour.

Observable state at writing time (verified 2026-10-15):

- `cmd/ovc/` contains `main.go` plus the per-platform signal files only: **no `--ui` flag and no
  `go:embed` directive exist yet**.
- `docs/research/` is untracked in git — the three studies and this record are not yet committed.

Maintenance: when milestone 1 lands, update this section (or supersede the document) rather than
letting it drift. The three studies remain the evidence; this file is the decision record, and a
decision that changes must be edited here, not silently contradicted elsewhere.
