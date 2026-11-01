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

**Trust:** anyone with the agent access secret may change server-side settings and credentials (model provider key and separate OpenAI key for web/transcription). Details: `requirements/technical/04-auth-and-settings.md`.

### REQ-CS-1 A server asks for its secret before it shows anything else

- **Bin:** explicit requirement
- **Statement:** Opening a server the browser is not yet authorized for shall ask for the connection secret, and that response shall be only the login page. Settings, saved connections, models, and secret fields shall not be in the document until that secret is accepted.
- **Origin:** User, 2026-11-01 (quote: "when I open that server, I see the connection dialog of the settings" and "greet the user in this case with something similar to a login screen requesting the connection secret before showing anything else" and "Otherwise the backend might accidentally share other secrets with some unauthorized entity"). User, 2026-11-01 (quote: "the dom ist already completely there with all secret input fields" and "Wouldn't it be better to show only the login page until the access secret is provided?").
- **Rationale:** Hiding the settings document is not the same as withholding it. The page an unauthorized visitor receives should not contain those fields at all.
- **Fit criterion:** The unauthorized response has a connection-secret field and does not contain the settings tabs, the secret inputs, or the app shell. A wrong secret stays on that page and does not return the app document. An accepted secret loads the session. A later Settings action can still show the connection page.
- **Priority:** shall
- **Status:** confirmed
