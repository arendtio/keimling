# Feature: Client–Server Keimling

## Vision

Keimling remains a single Go binary, but splits into a long-running **agent server** and thin **clients** (CLI, native WebView GUI, or any web browser). The server owns the agent loop, tools, sessions, and policy. Clients are view/control surfaces only.

## Goals

- Run the agent continuously on a host; attach intermittently from another machine or phone (browser).
- Same interactive surface in native GUI WebView and in a normal browser (UI served by the server).
- Keep using the app as a local CLI when no GUI is available.
- One binary under ~20 MB (smaller preferred), with web assets embedded (`embed.FS`).
- Use the OS webview (no bundled Chromium). Dynamic system libraries are acceptable.

## Non-goals (MVP)

- Full embedded terminal emulator (xterm) in the GUI.
- Bundled browser engine.
- Mobile native apps (phone uses the browser against the server).
- Separate frontend framework / SPA toolchain (vanilla HTML/CSS/JS).

## Functional outline

Presentation is selected with **normal flags** (and config keys), not subcommands:

| Mode | Behavior |
| --- | --- |
| *(default)* | Autodetect: try native GUI; on failure run CLI. |
| `--ui` | Require native GUI; fail if WebView/display unavailable. |
| `--cli` | Shell client only. |
| `--server` | Agent HTTP server only; no interactive client. |

Transport is **HTTP** (WebSocket over HTTP). No in-process HTTPS; TLS via external proxy if needed. Localhost embedded use is plaintext by design.

Clients (CLI or GUI) may use a **local** embedded agent or **connect** to a remote agent server. Tools always execute on the machine that runs the agent server. All CLI options are also expressible in the config file.
