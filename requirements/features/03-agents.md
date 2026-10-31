# Feature: Agents (definitions and instances)

## Intent

An **agent** is a reusable **definition** that can be **instantiated**. Instances are independent of workspaces: a definition lives globally; an instance may be granted access to one or more workspaces and runs with its own main thread, message queue, and OS worker process.

Human workspace **threads** stay as they are. Agents are created explicitly (no default instance).

## Concepts

| Term | Meaning |
| --- | --- |
| **Definition** | Reusable agent recipe: name, instructions/prompt, optional model defaults. Config store. (API: `recipes`.) |
| **Instance** | Embodiment: ACL (workspaces + peer instances), main thread, inbound queue, alarms. Data store. |
| **Main thread** | Single conversation journal for an instance. |
| **Idle / Busy** | Idle = not in a run loop (no green status). Busy = executing (green). |
| **Alarm** | Relative (`in 2m`) or absolute (`at 15:04` / RFC3339) wake; persists; injects a user wake message and starts a run if idle. |
| **Peer ACL** | Instance ids this agent may message via `message_agent`. |
| **Inbound queue** | Durable messages (human, peer, alarm) while busy; drained when idle. |
| **Team** | Named set of definition ids; spawning creates one instance per definition in the current workspace. |
| **Stop / Delete** | Stop cancels run; delete instance removes data. Delete definition removes recipe (blocked if instances remain, or cascade-none). |

## Storage (XDG)

```
~/.config/keimling/
  agents/<definition-id>.json
  agent-teams/<team-id>.json

~/.local/share/keimling/
  agent-instances/<instance-id>/
    meta.json          # recipe_id, workspaces[], peers[], …
    alarms.json
    queue.jsonl
    thread/session.jsonl
```

## Process model

- One OS worker per instance (`--agent-instance`).
- Host keys: `agent:<id>` vs `ws:<id>`.
- `change_workspace` among ACL roots.
- Peer delivery is same-host filesystem enqueue to the target’s `queue.jsonl` (+ ensure/wake worker when possible).

## Lifecycle

1. Create/edit/delete **definitions**; optional **teams**.
2. Create **instance** from definition (or spawn team).
3. Messages / peer messages / alarms → queue → run when idle.
4. Stop cancels; delete instance removes dir + worker.

## UI

- Threads then Agents (stable thread list when an agent is open).
- New agent: pick/create definition → instance name.
- Definitions manager in the same dialog: edit + delete.
- Teams: list/create/spawn (optional section in dialog).
- Green status only while busy.

## Tools (agent-instance only)

| Tool | Purpose |
| --- | --- |
| `change_workspace` | Switch CWD within workspace ACL |
| `set_alarm` | Relative or absolute alarm + wake message |
| `list_alarms` / `cancel_alarm` | Inspect / remove |
| `list_agents` | Discover instances that share a workspace, and which contacts the user allowed |
| `message_agent` | Enqueue text on a contact’s main channel. Contact is set by the user and works both ways |

## API (host)

| Method | Path | Notes |
| --- | --- | --- |
| `GET`/`POST` | `/api/agents/recipes` | List / create |
| `PUT`/`POST` | `/api/agents/recipes/update` | Update definition |
| `POST` | `/api/agents/recipes/delete` | Delete definition (fail if instances use it) |
| `GET`/`POST` | `/api/agents/instances` | List / create |
| `POST` | `/api/agents/instances/attach\|stop\|delete` | Lifecycle |
| `POST` | `/api/agents/instances/message` | Enqueue+wake `{ id, text, from? }` |
| `GET`/`POST` | `/api/agents/instances/access` | Workspaces, tools, and contacts. POST `peers` omitted leaves contacts unchanged; a list replaces them in both directions |
| `GET`/`POST` | `/api/agents/teams` | List / create team |
| `POST` | `/api/agents/teams/spawn` | Spawn instances for team |
| `POST` | `/api/agents/teams/delete` | Delete team |

## Non-goals

- Definition marketplace
- Cross-machine peer messaging

## Access and the main channel

Added 2026-10-31 from the user. The tables above are the earlier design. These records are the ask. A matrix control, and how a sent line is drawn, are not decided here.

The earlier text grants an instance workspace and peer access. It does not say the user chooses which tools an instance may use. That choice is REQ-AG-2.

### REQ-AG-1 Workspaces an instance may use

- **Bin:** explicit requirement
- **Statement:** The user shall choose, for each agent instance, which of the existing workspaces that instance may use. An instance shall be able to work in more than one workspace when more than one is granted. A workspace that was not granted shall not be usable by that instance.
- **Origin:** User, 2026-10-31 (quote: "es soll auch Agents geben können, die Workspace übergreifend arbeiten können" and "für jede Agent-Instanz genau sagen kann, du hast Zugriff auf folgende Workspaces. Im Sinne von, es gibt diese Workspaces und da hast du Zugriff").
- **Rationale:** Some agents should work across workspaces. The user wants to see the workspaces that exist and mark which of them this instance may use.
- **Fit criterion:** Workspaces W1, W2, and W3 exist. The user grants an instance W1 and W2. The instance can work in W1 and in W2. It cannot work in W3. The user can see that W1 and W2 are granted and W3 is not.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-2 Tools an instance may use

- **Bin:** explicit requirement
- **Statement:** The user shall choose, for each agent instance, which tools that instance may use. A tool that was not granted shall not be available to that instance.
- **Origin:** User, 2026-10-31 (quote: "dass man genau sagen kann … auf welche … Workspaces und … Tools ein Agent Zugriff haben soll" and "so wie eine Form Matrix").
- **Rationale:** Not every agent should have every tool. The user wants the same kind of explicit grant as for workspaces.
- **Fit criterion:** The user grants an instance a subset of the tools. A run of that instance can call a granted tool and cannot call a tool that was not granted. The user can see, against the tools that exist, which ones this instance may use.
- **Priority:** shall
- **Status:** superseded by REQ-AG-5
- **Open:** What an instance may use before the user has made a choice is not stated. Do not treat "all tools" or "no tools" as the requirement.

### REQ-AG-5 Definition tools, instance workspaces, instance may only narrow tools

- **Bin:** explicit requirement
- **Statement:** A definition shall say which tools that kind of agent may use. A definition shall not name workspaces. An instance shall choose which existing workspaces it may use. An instance should be able to use fewer of the definition’s tools. An instance shall not gain a tool the definition does not grant. When the instance has not narrowed tools, it uses the definition’s tools.
- **Origin:** User, 2026-10-31 (quote: "die Definition, sollte auch grundsätzlich Access Rights zu Tools beispielsweise definieren" and "die Instanz … diese Access Rights für die einzelne Instanz überschreiben" and "welche Workspaces relevant sind … die Agenten-Definition muss jetzt nicht Workspaces enthalten" and "der Zugriff auf bestimmte Tools … einschränkbar … je nachdem, in welchem Kontext").
- **Rationale:** The definition says what this kind of agent can do. The instance says where it is used, and may be trusted with less in that place. This is the same shape as a thread model that follows settings until it has its own choice.
- **Fit criterion:** A definition grants tools T. A new instance of it can call a tool in T and cannot call a tool outside T. The user narrows that instance to a subset S of T. The instance can call S and cannot call T−S or anything outside T. Saving the instance with T again makes it follow the definition. The user grants the instance workspaces W1 and W2. It appears in both. The definition has no workspace list.
- **Priority:** shall for the definition’s tools, the instance’s workspaces, and the ban on widening. should for the instance narrowing tools.
- **Status:** confirmed

### REQ-AG-6 One place to manage definitions, instances, and teams

- **Bin:** explicit requirement
- **Statement:** The user shall have one place to manage definitions, teams, and instances, including creating an instance, setting its workspaces and tool narrowing, and deleting an instance. The sidebar list of agents in the current workspace shall open an agent’s main channel and shall be able to stop a run. Creating, deleting, and access choices shall not be actions on that sidebar row.
- **Origin:** User, 2026-10-31 (quote: "irgendwo eine Liste … hier kann ich meine Agenten verwalten … die Definitionen" and "den Button neben Agents nicht New nennen sollte, sondern vielleicht Manage" and "löschens eines Agenten … nur über den Manage Agents Dialog … in der Sidebar … den Agenten stoppen").
- **Rationale:** Definitions, teams, and instances are maintenance. The sidebar is for opening the channel that is relevant in this workspace.
- **Fit criterion:** From the agents section the user opens management and can edit a definition’s tools, see every instance, set an instance’s workspaces and narrower tools, create an instance, and delete an instance. The sidebar row for an agent in the current workspace can stop a run and does not offer delete or access.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-7 Tool access grouped by kind of work, read separate from write

- **Bin:** explicit requirement
- **Statement:** The user should see tool access grouped by the kind of work, and should choose reading and writing separately. For files, editing and writing should be one writing choice, separate from reading. For alarms, listing should be separate from setting and cancelling. Beside a choice, the user should see which tools that choice turns on.
- **Origin:** User, 2026-10-31 (quote: "die Tools sind nach Thema sortiert und dann kann ich lesende und schreibende Operationen nochmal unterteilen" and "Dateien lesen, editieren, schreiben, dass das alles ein Block ist" and "editieren und schreiben ist beides eigentlich die Schreibberechtigung" and "List Alarm ist was Lesendes, Set und Cancel sind schreibende Operationen" and "wenn ich bei schreiben die Checkbox setze, dass dann gleich dran steht, welche Tools das jeweils betrifft").
- **Rationale:** Granting tools one by one is slow, and edit and write are the same kind of change. The user wants to decide by topic and by whether the agent may only look or may also change.
- **Fit criterion:** On a definition and on an instance, file reading is a different choice from file editing and file writing, and those two change together. Listing alarms is a different choice from setting and cancelling alarms, and those two change together. The writing choice shows the tool names it covers. An instance still cannot gain a tool the definition does not grant.
- **Priority:** should
- **Status:** confirmed

### REQ-AG-8 MCP access is per server

- **Bin:** explicit requirement
- **Statement:** MCP access should be chosen per server, not as one choice for every MCP tool. The user should be able to include or leave out a whole server. Beside that choice, the user should see that server’s tools. The product should not ask the user to mark those tools as reading or writing.
- **Origin:** User, 2026-10-31 (quote: "bei den MCP-Servern bin ich mir nicht sicher, ob das sinnvoll ist, alles in einen zu packen" and "eine zweite Tabelle, wo dann die ganzen MCP-Server-Tools drin sind" and "da wissen wir ja nicht, welche davon schreibend, welche davon lesend sind" and "pro MCP-Server eine Liste von Tools und man kann dann den ganzen MCP-Server reinnehmen oder rausnehmen").
- **Rationale:** MCP servers do different jobs. One switch for all of them is too coarse, and the app cannot honestly sort those tools into reading and writing.
- **Fit criterion:** Two MCP servers are configured. The user can include one and leave the other out, on a definition and, more narrowly, on an instance. The included server’s tools are listed beside that choice. The other server’s tools are not available. There is no read/write split for those tools. An instance cannot include a server the definition does not include.
- **Priority:** should
- **Status:** confirmed

### REQ-AG-9 The human sets who may contact whom, in both directions

- **Bin:** explicit requirement
- **Statement:** The user shall choose, for an agent instance, which other instances it may contact. That choice shall allow both instances to message each other, so the receiver can reply. An instance shall not message an instance the user has not included. The user sets this while managing instances.
- **Origin:** User, 2026-10-31 (quote: "dieser Agent soll Zugriff auf den Agent haben" and "ich würde das immer beim sendenden Agent aktuell machen" and "eigentlich ist es interessanter zu sagen, ein empfangender Agent akzeptiert die Nachrichten" and "der Mensch festlegt, welcher Agent zu wem Kontakt aufnehmen darf" and "es muss ja eigentlich immer eine bidirektionale Erlaubnis sein … der Empfänger was zurückschreiben darf" and "das kann man da irgendwie auch bei dem Manage Agent mit eintragen" and "obwohl ich in manage agents es eingestellt habe bekommt der agent nur: message_agent error: the user has not allowed contact with this agent").
- **Rationale:** Without a human-set boundary, any agent could message any other. A one-way grant leaves the receiver unable to answer when something goes wrong.
- **Fit criterion:** Instances A, B, and C exist. The user includes B on A. A can message B, and B can message A. A cannot message C, and C cannot message A. The user can see and change that set while managing instances. If A passes its own id, the reply says that id is A and names the other id to pass. It does not say the user has not allowed contact.
- **Priority:** shall
- **Status:** confirmed
- **Open:** Whether a team template stores a contact pattern before spawn is not stated. Spawn today connects every instance created in that spawn with every other.

### REQ-AG-3 One main channel, listed in every granted workspace

- **Bin:** explicit requirement
- **Statement:** An agent instance shall have one main channel. That channel shall be the agent's ongoing process, and the user shall be able to put requests into it. When the instance may use more than one workspace, the user shall be able to open that same channel from each of those workspaces.
- **Origin:** User, 2026-10-31 (quote: "wenn ein Agent in mehreren Workspaces Zugriffsrechte hat, dann sollte auch in beiden sein Hauptkanal auftauchen" and "der Hauptkanal eines Agents ist eigentlich sein Denkprozess, in dem man Anfragen reinschmeißen kann").
- **Rationale:** The user moves between workspaces and still wants the same agent conversation, including what they sent and what agents sent each other.
- **Fit criterion:** An instance may use W1 and W2. The user opens its main channel from W1 and sends a request. The user opens the same instance from W2. The channel shows that request and the same later messages. It is not a second conversation.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-4 A message between agents is visible in both main channels

- **Bin:** explicit requirement
- **Statement:** When one agent sends a message to another, the product should show that exchange in both main channels. The sender's channel should show that this agent sent that text to the other agent. The receiver's channel should show that the text came from the sender. A person reading either channel should be able to see the exchange without opening the other agent.
- **Origin:** User, 2026-10-31 (quote: "Agent A hat in seinem Hauptkanal stehen, dass er grade folgende Nachricht an Agent B geschickt hat und bei Agent B steht dann die Nachricht in dem Hauptkanal, dass sie von Agent A kommt. Ich glaube, sowas wäre sinnvoll").
- **Rationale:** The main channel is the agent's process. A message to another agent is part of that process, and the receiver's process should show where the request came from.
- **Fit criterion:** Agent A sends a text to agent B. A's main channel shows that A sent that text to B. B's main channel shows that same text as coming from A. Both remain visible when the user later opens either channel from a workspace that agent may use.
- **Priority:** should
- **Status:** confirmed

### REQ-AG-10 Identifiers a model copies do not contain a full stop

- **Bin:** explicit requirement
- **Statement:** A newly created agent instance id and definition id shall not contain a full stop. A model that is asked to copy the id shall not be able to treat the stop as the end of the id.
- **Origin:** User, 2026-10-31 (quote: "Having a full stop/point in the ID confuses some models... not such a good idea, why not use another dash?").
- **Rationale:** The id is something the model passes to tools. A period in the middle is read as the end of the token.
- **Fit criterion:** A new instance id and a new definition id contain no `.`. An id created before this change still opens.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-11 Leave a running channel without stopping it

- **Bin:** explicit requirement
- **Statement:** While a run continues, the user shall change thread, agent, or workspace, open settings and other dialogs, and queue a message in any thread. The run shall continue on the channel where it started. A message queued in a different thread of that workspace shall wait there and run when that thread is free.
- **Origin:** User, 2026-10-31 (quote: "Während ein Agent denk oder was anders tut, kann man teilweise nicht in andere thread oder agents oder den manage agent thread wechseln" and "ich will doch jederzeit den thread, agent oder workspace ändern können und auch jederzeit in settings oder andere dialoge wechseln und auch jederzeit in einem beliebigen thread nachrichten queuen können").
- **Rationale:** Waiting until the run finishes blocks looking at another conversation, changing access, or leaving a message for another thread.
- **Fit criterion:** A workspace thread is running. The user opens another thread of that workspace, another agent, another workspace, and settings. The first run is still going. Send stays available on the other thread. A message sent there is queued on that thread and is not written into the running thread. It runs when that thread is free. Settings can be saved while the run continues.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-12 An opened message stays open when a later message arrives

- **Bin:** explicit requirement
- **Statement:** When the user has opened a message, adding a later message shall leave that message open.
- **Origin:** User, 2026-10-31 (quote: "wenn jedes mal wenn ein Nachricht zum Thread hinzugefügt wird, alle aufgeklappten NAChrichten wieder zugeklappt werden... obwohl ja nur eine nachricht hinzukommt").
- **Rationale:** Rebuilding the whole transcript closes what the user just opened and makes the thread jump.
- **Fit criterion:** The user opens a tool line or a folded stretch. A new message arrives. That opened line is still open. Messages above it are not rebuilt.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-13 The definition model offers known model ids

- **Bin:** explicit requirement
- **Statement:** When the user sets the model on a definition, the product shall offer the model ids already configured, so the saved id is one the product knows.
- **Origin:** User, 2026-10-31 (quote: "beim setzen des models in der agent definition sinnvoll vorschläge zu bekommen, damit man auch wirklich die richtige id einfügt").
- **Rationale:** A mistyped id fails later, when the agent process starts, not while the user is looking at the field.
- **Fit criterion:** The model field lists configured model ids. The user can still leave it empty. Saving stores the chosen id.
- **Priority:** should
- **Status:** confirmed

### REQ-AG-14 Manage agents tabs are readable in the dark theme

- **Bin:** explicit requirement
- **Statement:** The tab titles in Manage agents shall stay readable in the dark theme. Theme colors shall come from the shared color variables.
- **Origin:** User, 2026-10-31 (quote: "Die Tab-Überschriften im Manage Agents Dialog sind im Dark Theme nicht lesbar (dunkle Schrift auf dunklem Grund). Sollte sowas nicht grundsätzlich über CSS variable gelöst sein?!?").
- **Rationale:** A control the user cannot read cannot be used. A one-off dark color breaks when the theme changes.
- **Fit criterion:** In the dark theme, Definitions, Instances, and Teams are readable on the dialog background. Their color is a theme variable, not a fixed dark ink.
- **Priority:** shall
- **Status:** confirmed

## Design notes (not requirements)

Until the user saves tools on a definition, that definition grants every tool. Saving stores the checked tools. An instance with no narrower set uses that definition. Saving the same set again clears the instance override. REQ-AG-2’s unanswered default is this design, not a stakeholder shall.

Tool choices are shown as a topic table with a read column and a write column. The stored grant is still the individual tool names. A cell that is only partly granted stays mixed until the user clicks it, so saving does not silently add the rest. An instance cell lists only the tools the definition allows.

MCP servers are a second table: one checkbox per server, and that server’s tool names beside it. The stored choice is the server, so tools it exposes later stay included. A saved `mcp` choice from before this table still means every server until the user saves again.

Contacts are set by the user on the instance. Saving the set writes the same link on both instances. A message is allowed when either instance lists the other, so an older one-way link still lets the other side reply. The model is not offered a tool that adds contacts. A definition may store a model id; the next start of that agent’s process uses it. Empty means the model from Settings.

A message from A to B is a line in A’s channel ("To B") and, when B’s worker runs, a line in B’s channel ("From A"). The host starts B’s worker when a message is waiting. The checkbox screen is the take-back: uncheck and save again. Delete of an instance asks by name and is not beside Save. There is no restore after that confirm.

New ids use a dash before the nanosecond fraction. Ids that already contain a period stay valid. While a workspace run is busy, the user can open another thread, queue a message there, and open settings. The running journal stays the one the run is writing. Opened tool lines and chapter folds stay in the page when a later message is appended. The definition model field suggests ids from the configured providers. Manage tabs use `--fg` on a transparent background, because the default button color `--accent-ink` is the ink for a filled accent button and is dark in the dark theme. Dialog text uses `--fg`. `--ink` is not a theme variable.
