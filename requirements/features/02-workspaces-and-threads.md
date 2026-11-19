# Feature: Workspaces and Threads

## Intent

Keimling work is organized by **workspace** (a filesystem root + side data) and **thread** (a conversation inside a workspace). Workspaces, their threads, and agent instances run in the one process that serves the UI (REQ-WS-6, REQ-WS-7). A session in a workspace starts in that workspace's directory, and tools are still judged inside or outside that directory (REQ-WS-8). Each thread keeps its own working directory after that (REQ-WS-11).

No backward compatibility with pre-workspace session paths is required.

## Concepts

| Term | Meaning |
| --- | --- |
| **Host** | The process that serves the Web UI and runs the workspaces and agent instances. |
| **Worker** | Retired by REQ-WS-7. A workspace is not its own operating-system process. |
| **Attach** | Which workspace or agent the UI is showing. |
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

One process. Switching workspace is the same kind of change as switching thread (REQ-WS-6). A run keeps going while the user looks at another thread, including a thread in another workspace (REQ-AG-11). A session starts in that workspace's root (REQ-WS-8). After that, each thread keeps its own working directory (REQ-WS-11).

```
One process (UI + workspaces + threads + agent instances)
Client ──HTTP──► that process
```

## Startup

1. The process ensures `global` and resolves the launch directory to the initial workspace. It does not change the process working directory for that.
2. Opening a workspace builds it in this process and shows it.
3. Opening another workspace does the same. A run already going in the one left behind keeps going.

## GUI

- Sidebar: workspace switcher, thread list, settings.
- Paper: chapters flyout; context + artifacts; composer shows model/mode.
- Workspace switch reloads session/SSE for the new attach — app stays up.

## API

- `GET /api/workspaces` — list + `current` + which are open in this process
- `POST /api/workspaces/open` — `{ "path" }` open that folder in this process
- `POST /api/workspaces/switch` — `{ "id" }` show that workspace
- `POST /api/workspaces/hide` — `{ "id" }` remove from the menu; no files deleted. Opening the path again lists it.

Served by the same process: `/api/session`, `/api/threads*`, `/api/message`, `/events`, `/files/*`, settings, skills, mcp, …

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

### REQ-WS-3 Start a thread without a question

Starting a thread in the open workspace does not ask for confirmation. The previous thread stays in the list.

Origin: user, 2026-11-07. "können wir diese confirmation wenn man einen neuen thread öffnet auch ausbauen? Ich meine der schrit ist ja einfach rückgängig zu machen, da brauch man kein confirm."

Fit: the New control creates the thread immediately. The thread that was open is still in the list and can be opened again.

### REQ-WS-4 Name a chat in another workspace

A chat in this workspace can name a chat in any workspace and read that chat's journal. The name is the workspace and the thread. The named journal stays read-only. The files of that workspace are not part of the reference.

Origin: user, 2026-11-14. "dann hätte ich gern eine Möglichkeit, um Chats über Workspaces hinweg zu referenzieren." The same conversation settled that the analysis happens in one workspace and the other chat is named, not copied into the message.

Fit: from one workspace, a read of `chat:<workspace>/<thread>` returns text that is only in the other chat's journal. An edit or overwrite of that journal does not change it. An unknown or ambiguous name says which workspaces or threads match.

### REQ-WS-5 Copy the open chat's address

The Now tab shows the open chat's workspace and thread in a footer that stays at the bottom of that tab. Copy places `chat:` plus the workspace id and the thread id on the clipboard. The ids stay valid after a rename. The footer does not show an agent chat, which has no workspace thread address.

Origin: user, 2026-11-14. "ob es irgendwie 'ne Möglichkeit gibt, wie ich die ID eines Chats kopieren kann" and, agreeing to the footer, "ok, bitte bauen."

Fit: with a workspace thread open, Now shows that workspace's name and that thread's title at the bottom of the tab, and it stays there while the plan is long enough to scroll. Copy puts `chat:<workspace id>/<thread id>` on the clipboard and the button says Copied. An agent chat does not show the footer.

### REQ-WS-6 Switch workspace like a thread

- **Bin:** explicit requirement
- **Statement:** Switching from the open workspace to another known workspace shall be as seamless as switching from one thread to another. The switch shall not wait for a new process to become ready, and it shall not abort because that process never becomes ready.
- **Origin:** User, 2026-11-19 (quote: "wenn ich wechseln möchte, es erstens teilweise sehr lange dauert und zweitens manchmal auch gar nicht dazu kommt, dass der neue Prozess healthy ist und es dann abbricht" and "das ist aus meiner Sicht schon mal, 'ne kleine Katastrophe" and "der Wechsel von einem Workspace zum anderen genauso seamless funktionieren wie der Wechsel zwischen Threads").
- **Rationale:** The pause and the abort are the failure the user hits. A boundary that only keeps the window alive while the switch fails is not a benefit.
- **Fit criterion:** The user selects another workspace that is already in the list. Its chat is shown. When the thread they land in has no messages, that empty thread is shown, not the chat they just left. The switch does not start a process and does not fail with a worker that never became healthy. A run that was already going in the workspace they left is still going.
- **Priority:** shall
- **Status:** confirmed

### REQ-WS-7 One process for the window, the workspaces, and the agents

- **Bin:** explicit constraint
- **Statement:** A workspace shall not be its own operating-system process. An agent instance shall not be its own operating-system process. A run that continues while the user looks elsewhere shall do so inside the process that serves the window.
- **Origin:** User, 2026-11-19 (quote: "was dieses Multiprozess-Geraffel für die einzelnen Workspaces wirklich bringt" and "dafür diese technische Hürde, ich mach einen eigenen Prozess aus, auf eigentlich Overkill" and "im Hintergrund geht es weiter, das ist davon ja auch komplett unabhängig" and "dann brauchen wir auch diesen ganze Geraffel mit dem Multiprozess-Kram alles nicht").
- **Rationale:** Separate processes are what make the switch slow and what let a failed start disappear as an unhealthy child. Continuing in the background is already how two threads share one process. Clean separation belongs in the program, not in a process per workspace.
- **Fit criterion:** Opening a second workspace does not listen on a new port and does not print another "keimling server listening". An agent with a queued message, an alarm, or a watch runs without a second process of this program. The window process is the one that serves the UI and runs those chats.
- **Priority:** shall
- **Status:** confirmed

### REQ-WS-8 A session starts in the workspace directory

- **Bin:** explicit constraint
- **Statement:** At the start of a session in a workspace, the working directory shall be that workspace's root. When a tool runs, the product shall check whether the path is inside that workspace or outside it.
- **Origin:** User, 2026-11-19 (quote: "immer wenn ich in einem Workspace bin, der Agent automatisch im richtigen Arbeitsverzeichnis ist und das muss auch unbedingt so weiter sein" and "das Hauptarbeitsverzeichnis, dass das zum Beginn einer Session das Richtige ist, ist sehr wichtig" and "darauf geprüft wird, wenn Tools ausgeführt werden, ob die im Workspace sind oder außerhalb. Auch das ist sehr wichtig").
- **Rationale:** The directory is the real boundary of a workspace. Removing the process split must not drop it.
- **Fit criterion:** A session opened in a workspace starts with that workspace's root as its working directory. A tool path inside that root is treated as inside. A tool path outside that root is treated as outside. Both stay true after workspaces run in the one window process.
- **Priority:** shall
- **Status:** confirmed

### REQ-WS-9 A failure is shown

- **Bin:** explicit requirement
- **Statement:** A failure shall be shown to the user so it can be fixed. The product shall not hide a failure behind a process boundary so that it looks more stable than it is.
- **Origin:** User, 2026-11-19 (quote: "wenn es Fehler gibt, ich die schnell mitbekomme und dann fixen lassen kann" and "ein System hab, was Fehler irgendwie verschluckt und nicht an die Oberfläche bringt und dann stabiler wirkt, als es eigentlich ist").
- **Rationale:** A window that stays up while the workspace never opens hides the defect. The user would rather see it and have it fixed.
- **Fit criterion:** When a workspace switch cannot complete, the user sees what failed. A crash or a failed start is not only the absence of a healthy child process.
- **Priority:** shall
- **Status:** confirmed

### REQ-WS-10 Threads from different workspaces can sit together

- **Bin:** explicit requirement
- **Statement:** The product should allow a later view in which threads from different workspaces sit beside each other and the user moves between them directly. That view is not required yet. The way a workspace is opened shall not make that view wait on a process.
- **Origin:** User, 2026-11-19 (quote: "es wäre auch denkbar, dass man später mal 'ne Ansicht baut, in der Threads aus verschiedenen Workspaces nebeneinander sind und man direkt hin und her wechseln möchte").
- **Rationale:** A workspace switch is a thread switch one level up. A later side-by-side view only works if opening another workspace is already that cheap.
- **Fit criterion:** Nothing in the way a workspace is opened requires a new process before a thread from that workspace can be shown next to a thread from another workspace. The side-by-side view itself is not built.
- **Priority:** should
- **Status:** confirmed

### REQ-WS-11 Each thread has its own working directory

- **Bin:** explicit requirement
- **Statement:** Each thread shall have its own working directory. A directory change in one thread shall affect later commands of that thread only.
- **Origin:** User, 2026-11-19 (quote: "das jeder thread seinen eigenes work directory hat, also wenn in einem cthread ein cd ausgeführt wird, betrifft dies nur den jeweiligen thread").
- **Rationale:** Two threads in one workspace do different jobs. A `cd` in one of them must not move the other. The workspace root stays the start (REQ-WS-8) and the inside-or-outside check. The thread directory is where that thread's commands start.
- **Fit criterion:** Two threads in one workspace both start at that workspace's root. After a command in thread A changes its directory, the next command in A starts in that directory. The next command in thread B starts in B's directory, which is still the root until B changes its own. A new thread starts at the workspace root.
- **Priority:** shall
- **Status:** confirmed
