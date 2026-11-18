# Feature: Workspace runs

## REQ-WR-1 Auto mode may edit the workspace

- **Bin:** explicit requirement
- **Statement:** In auto mode the workspace agent may edit and create any file inside the workspace, including a file the run just created. A path outside the workspace or a protected path still asks.
- **Origin:** User, 2026-11-18 (quote: "Auto darf jede Datei im Workspace bearbeiten").
- **Rationale:** Asking on every existing file, including one the run just wrote, blocks ordinary work.
- **Fit criterion:** An in-workspace edit or write is allowed in auto mode. An outside path still asks.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-2 Built-in tools have a per-tool ask setting

- **Bin:** explicit requirement
- **Statement:** Settings lists each built-in tool with ask on or off. The default list never asks for harness tools. A hard floor still asks. Yolo still does not ask.
- **Origin:** User, 2026-11-18 (quote: "pro integrierten Tool auch das Fragen an- und auszumachen" and the never-ask list as proposed).
- **Rationale:** Ask mode should not interrupt tools that only serve the harness, and the user can change that choice.
- **Fit criterion:** `search_history` is allowed in ask mode until the setting says ask. `sudo` still asks when exec is set to not ask.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-3 Threads run independently

- **Bin:** explicit requirement
- **Statement:** Two threads in one workspace may call the model at the same time. A message to one thread does not wait for the other. Stopping one does not stop the other. Each thread keeps its own composer text. A local model runs one call at a time. A cloud model is unlimited.
- **Origin:** User, 2026-11-18 (quote: "Zwei Threads im selben Workspace müssen das Modell gleichzeitig aufrufen können").
- **Rationale:** A thread is its own conversation. Another thread's run is not a reason to queue.
- **Fit criterion:** While one thread is running, a message to the viewed thread starts that thread and leaves the first journal unchanged.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-4 A refresh token is used before a new sign-in

- **Bin:** explicit requirement
- **Statement:** An MCP server with a refresh token stays signed in when the access token lapses. Refreshing one server does not reconnect the others. The MCP and Context sidebar tabs update while they stay open.
- **Origin:** User, 2026-11-18 (quote: "dass sie den Refresh-Token auch erstmal nutzen, bevor sie den User jetzt um neuen Sign-in fragen").
- **Rationale:** The access token lasts about an hour. The refresh token is the grant the user already gave.
- **Fit criterion:** Expired access plus a refresh token and a token URL reports signed in.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-5 Instruction text opens in a large editor

- **Bin:** explicit requirement
- **Statement:** Settings has a button for autopilot instructions and a button for auto-review instructions. Each opens a large editor. The auto-review text is global. Empty saved text means the built-in prompt.
- **Origin:** User, 2026-11-18 (quote: "Ich möchte nicht, dass es einen großen Editor in den Einstellungen gibt, sondern dass man aus den Einstellungen heraus den Autoreview-Text und den Autopilot-Text editieren kann").
- **Rationale:** The prompts are long. The settings form stays a list of controls.
- **Fit criterion:** The settings form shows the two buttons and not the prompt body. Saving from the editor persists the text.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-6 Result references stay for the thread

- **Bin:** explicit requirement
- **Statement:** A thread's result references are not dropped for count or for a 32 MiB total. A result that would fill the model window still stays a reference.
- **Origin:** User, 2026-11-18 (quote: "die 32 MB Grenze auch weg, und die Anzahl").
- **Rationale:** DataLang is for large files, including one around 40 MiB.
- **Fit criterion:** A store with the default caps keeps more than 64 items and more than 32 MiB.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-7 The thinking bar moves

- **Bin:** explicit requirement
- **Statement:** The sidebar thinking bar animates even when the platform asks for reduced motion.
- **Origin:** User, 2026-11-18 (quote: "Animate the bar anyway").
- **Rationale:** The motion is the status. Reduced-motion support can come later.
- **Fit criterion:** The sweep animation is not removed by `prefers-reduced-motion`.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-8 The window exists before the workspace worker starts

- **Bin:** explicit requirement
- **Statement:** The desktop window is created before the workspace worker process starts. The Edit menu provides copy and paste. A failed window still falls back to the command line. The start does not retry in a loop.
- **Origin:** User, 2026-11-18 (quote: "warum es fehlschlägt" and "ich will jetzt nicht, dass er dreimal versucht").
- **Rationale:** Starting the worker first loads WebKit in a second process while the window is still checking in with Launch Services.
- **Fit criterion:** The host attaches the worker after the native window has been created and before the window event loop runs.
- **Priority:** shall
- **Status:** confirmed

## REQ-WR-9 Artifact proxy is a setting

- **Bin:** explicit requirement
- **Statement:** An artifact proxy setting is off by default. When it is on, an artifact may fetch any http or https host through this server.
- **Origin:** User, 2026-11-18 (quote: "wenn an, dann jeder Host").
- **Rationale:** A sandboxed artifact has its own origin, so a request to another host needs a proxy on this server.
- **Fit criterion:** With the setting off the proxy refuses. With it on, a request for an http or https URL is forwarded.
- **Priority:** shall
- **Status:** confirmed
