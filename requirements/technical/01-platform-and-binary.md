# Technical: Platform and binary

## Platforms

| Role | Platforms |
| --- | --- |
| Agent server | Linux, macOS, and Windows, with the same product on each |
| Native GUI client | Linux / macOS / Windows via **OS-built-in webview** |
| Browser client | Any modern browser against the server URL (including phones) |

## REQ-PLAT-1 Three platforms, one product

- **Bin:** explicit requirement
- **Statement:** Keimling shall support Linux, macOS, and Windows as the same product. It shall not be described or treated as a Linux-first project.
- **Origin:** User, 2026-11-03 (quote: "ich hab schon mal geäußert, dass ich nicht möchte, dass das als Linux-First-Projekt bezeichnet wird. Weil alle drei Plattformen, Windows, macOS und Linux gleichermaßen supported werden soll. Also es soll eher ein Cross-Platform Projekt sein.").
- **Rationale:** A person should be able to build and use Keimling on the computer they have, and the docs should not tell an assistant otherwise.
- **Fit criterion:** Product docs do not call Keimling Linux-first or Linux-only. A CLI/server build succeeds for linux, darwin, and windows. The native window is built on that operating system from the system web view.
- **Priority:** shall
- **Status:** confirmed

## REQ-PLAT-2 GUI builds from GitHub

- **Bin:** explicit constraint
- **Statement:** GitHub shall produce the Keimling builds that include the native window, for Linux, macOS, and Windows, and those builds shall be downloadable from the workflow run.
- **Origin:** User, 2026-11-03 (quote: "Also Bauen ohne GUI finde ich etwas witzlos. Daher noch mal die direkte Frage, könnte Github für uns auch die Version mit GUI Support bauen?" and "ok, dann baue gerne den Workflow auch mit ein").
- **Rationale:** A person on macOS or Windows should be able to fetch a windowed binary without setting up a C compiler locally.
- **Fit criterion:** `.github/workflows/gui.yml` runs on GitHub and uploads a windowed binary for Linux amd64, macOS arm64, macOS amd64, and Windows amd64. The Windows artifact links WebView2. The macOS artifacts link WebKit. The Linux artifact links WebKitGTK.
- **Priority:** shall
- **Status:** confirmed

## REQ-PLAT-3 Experimental builds use the short commit

- **Bin:** explicit requirement
- **Statement:** While releases are experimental, a build shall be identified by the short form of the git commit it was built from.
- **Origin:** User, 2026-11-03 (quote: "Also für den Moment würde mir die Kurzform des letzten commits reichen. Es sind aktuell ja alles noch experimentelle versionen").
- **Rationale:** A downloaded binary and a local build need a name that points at the exact source, without a separate release number yet.
- **Fit criterion:** `--version` prints `keimling` and that short commit. `make build` and `.github/workflows/gui.yml` stamp it into the binary. Workflow artifact names include the same short commit. A build without the stamp prints `keimling dev`.
- **Priority:** shall
- **Status:** confirmed

## REQ-PLAT-4 A release after every push to main

- **Bin:** explicit requirement
- **Statement:** Every push to `main` shall publish a GitHub Release named with the short commit and containing the windowed binaries for Linux, macOS, and Windows.
- **Origin:** User, 2026-11-03 (quote: "ok, können wir automatisiert nach jedem push auf main ein release bauen?").
- **Rationale:** An end user should find the binaries on the repository's Releases page, not only inside a workflow run.
- **Fit criterion:** After a push to `main`, `https://github.com/arendtio/keimling/releases/tag/<short>` offers the four windowed binaries. A pull request does not publish a release.
- **Priority:** shall
- **Status:** confirmed

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
