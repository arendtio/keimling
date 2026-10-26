# User stories: modes and attach

## US-01 — Always-on agent host

**As** an operator, **I want** to run `keimling --server --listen …` with an access secret (flag or config), **so that** the agent keeps working overnight and clients can attach later.

**Acceptance**

- Without `--listen` (or config equivalent), `--server` refuses to start.
- Without an access secret (flag or config), `--server` refuses to start.
- Tools (shell, files, …) execute on the server host.
- Opening the listen URL in a browser shows the same UI a WebView client would show.
- Server speaks HTTP; TLS is optional via an external reverse proxy.

## US-02 — Attach from another machine

**As** a user on a laptop or phone, **I want** to open the server URL (browser) or use the native GUI against that URL, **so that** I can inspect state and send new instructions without running tools locally.

**Acceptance**

- GUI can target a remote base URL (`--connect` / config); no local agent starts in that case.
- CLI can `--connect` to the same server.
- Unauthenticated requests to the API/UI are rejected.

## US-03 — Local GUI by default

**As** a desktop user, **I want** to start `keimling` (or `keimling --ui`) and get a native window, **so that** I am not forced to use a terminal.

**Acceptance**

- Autodetect prefers GUI when WebView works.
- `--ui` fails clearly if WebView/display is unavailable.
- Local GUI uses an embedded localhost HTTP agent; binary stays one process from the user’s point of view.
- On macOS/Windows, only OS-provided webview is used (no extra browser install required by us beyond normal WebView2 presence on Windows).
- After local desktop launch, the composer is enabled for a new message (not stuck disabled).
- Tool runs appear as a compact one-line “Ran …” summary that expands to the full command/result.
- Approval (Approve/Deny) appears next to the pending command/reason, not as permanent composer chrome.
- Settings (sidebar) covers agent connection (local vs remote URL); full config editing is still deferred.

## US-04 — CLI fallback

**As** a user on a headless or broken-GUI environment, **I want** `keimling` / `keimling --cli` to work in the shell, **so that** I still control a local or remote agent.

**Acceptance**

- Autodetect falls back to CLI when GUI cannot start.
- `--cli` never attempts WebView.
- CLI can use embedded local agent or `--connect`.

## US-05 — Small binary

**As** a maintainer, **I want** the release binary under 20 MB with embedded UI, **so that** Keimling stays deployable on small machines.

**Acceptance**

- CI or release checklist records binary size.
- UI is vanilla assets via `embed`; no Electron.

## US-06 — Config parity

**As** an operator, **I want** every CLI flag (including `--server`, `--listen`, access secret, `--connect`, `--ui` / `--cli`) available in the config file, **so that** long-running server setups need no fragile wrapper scripts.

**Acceptance**

- Documented config keys for all presentation/network/auth options.
- Flags override config (existing Keimling precedence).
