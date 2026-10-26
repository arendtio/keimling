# Technical: Platform and binary

## Platforms

| Role | Platforms |
| --- | --- |
| Agent server | Linux, macOS, Windows (Linux is the primary day-to-day host) |
| Native GUI client | Linux / macOS / Windows via **OS-built-in webview** |
| Browser client | Any modern browser against the server URL (including phones) |

Previous docs said “Linux only”; that was the tested footprint, not the product limit.

## Native webview by OS (no extra shipped browser)

| OS | Engine | Extra app library? |
| --- | --- | --- |
| **macOS** | **WKWebView** (system WebKit / Cocoa) | No — part of the OS |
| **Windows** | **WebView2** (Edge/Chromium engine as OS/runtime component) | No separate browser bundle in our binary; WebView2 Evergreen is normally already on Win10/11. We do not ship Chromium. |
| **Linux** | **WebKitGTK** (GTK + WebKit) | Distro packages (`webkit2gtk-4.0` and/or `4.1`, …). **WebKitGTK versioning is Linux-only**; it does not apply to macOS/Windows. |

**Requirement:** On macOS and Windows, rely only on what the OS already provides. Do not require installing a third-party browser or bundling one in the Keimling binary.

## Linux WebKitGTK flexibility

- Support **both 4.0 and 4.1** (and prefer detecting whichever `pkg-config` provides at build or link time).
- Abstract behind the webview binding’s platform backend; application code only calls `Navigate` / window lifecycle — no GTK version checks in business logic.
- Dev machine note (2026-10-25): Arch has `webkit2gtk-4.1`, not `4.0` → the Linux build path must not hard-require only 4.0.

## Binary constraints

- Target size: **&lt; 20 MB** for the release binary with embedded UI assets.
- Prefer static Go linking for the Go parts; **dynamic link to system webview** is required and accepted.
- No Electron / no shipped Chromium.
- UI assets: vanilla HTML, CSS, JS, embedded in the binary.

## HTTP only (no in-process TLS)

- The agent exposes **HTTP** (and WebSocket over HTTP).
- **No HTTPS server in Keimling** for MVP. TLS, if needed, terminates at an external reverse proxy.
- Local embedded use binds **localhost** → plaintext HTTP is acceptable.

## Auth and listen (server presentation)

- In `--presentation=server` (name TBD), `--listen` is **mandatory**.
- Access **secret** is required for non-localhost (and configurable via CLI **and** config file, like all other options).
- Localhost embedded agent may use an ephemeral port + launch-scoped token (see process-model doc).
