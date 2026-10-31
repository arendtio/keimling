# Feature: Workspaces and Threads

## Intent

Keimling work is organized by **workspace** (a filesystem root + side data) and **thread** (a conversation inside a workspace). A long-lived **host** process serves the UI and supervises **one OS worker process per workspace** so tools share a stable CWD and agents can run in parallel. The UI **attaches** to a worker; switching workspace does not restart the host or stop other workers.

No backward compatibility with pre-workspace session paths is required.

## Concepts

| Term | Meaning |
| --- | --- |
| **Host** | Process that serves the Web UI, workspace registry/API, and reverse-proxies to the attached worker. |
| **Worker** | OS process for one workspace (`chdir` to root); agent loop + tools + thread journals. |
| **Attach** | Which worker the UI currently talks to (via host proxy). |
| **Thread** | One journal/charter conversation inside a workspace. |
| **Global workspace** | Built-in workspace `id=global`, root=`$HOME`, always present. |
| **Side data** | Artifacts, thread journals, workspace meta — **not** inside the git repo. |

## Storage (XDG)

```
~/.config/keimling/          # configuration only
  config.json
  gui-prefs.json
  mcp.json

~/.local/share/keimling/
  workspaces/
    <workspace-id>/
      meta.json
      artifacts/
      threads/
        <thread-id>/
          session.jsonl
          thread.json        # title, etc.
```

## Process model

```
Host (UI + /api/workspaces* + proxy)
 ├── Worker A (workspace A)  … keeps running while attached elsewhere
 ├── Worker B (workspace B)
 └── …
Client ──HTTP──► Host ──proxy──► attached Worker
```

- Switching workspace = **reattach** (proxy target changes). No host/UI relaunch.
- Workers keep running until the **host** exits (or an explicit stop later).
- Threads switch in-process inside a worker (same CWD).
- Workers are started with `KEIMLING_ROLE=worker` so they do not nest another host.

## Startup

1. Host ensures `global` workspace; resolves launch CWD → initial workspace.
2. Host starts (or reuses) a worker for that workspace and attaches to it.
3. Opening another workspace starts its worker if needed and attaches; previous workers stay up.

## GUI

- Sidebar: workspace switcher, thread list, settings.
- Paper: chapters flyout; context + artifacts; composer shows model/mode.
- Workspace switch reloads session/SSE for the new attach — app stays up.

## API

Host-owned:

- `GET /api/workspaces` — list + `current` (attached) + running flags
- `POST /api/workspaces/open` — `{ "path" }` ensure worker + attach
- `POST /api/workspaces/switch` — `{ "id" }` attach (`relaunch: false`)
- `POST /api/workspaces/hide` — `{ "id" }` remove from the menu; no files deleted. Opening the path again lists it.

Proxied to attached worker: `/api/session`, `/api/threads*`, `/api/message`, `/events`, `/files/*`, settings, skills, mcp, …

## Non-goals (for now)

- Cross-workspace agent overview dashboard beyond list/running
- Agent role library / inter-agent channels
- Keeping host alive after WebView close as a daemon (optional later)

## Requirements

### REQ-WS-1 Working directory in the prompt

The model is told which workspace it is in and the directory its commands start in.

Origin: user, 2026-10-31. "agents often do not know what their current work directory is. Maybe we can include that information as part of the workspace information?"

Fit: instructions for a workspace rooted at a known directory contain that directory and the workspace id. After a workspace change, the next request names the new directory.

### REQ-WS-2 Remove a workspace from the menu

The user can remove a workspace from the menu. That does not delete files. The home workspace stays. The workspace that is open stays until the user switches away. Opening the folder again puts the entry back.

Origin: user, 2026-10-31. "Now I need a functionality to remove old workspaces from the menu. No Files should be deleted, but the entry from the workspace menu should be removed."

Fit: after remove, the name is gone from the menu, a file in the workspace directory is still there, and Keimling's record of that workspace is still on disk. Opening that folder lists it again. Home cannot be removed.
