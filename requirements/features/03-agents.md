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
    watches.json
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
3. Messages / peer messages / watches → queue → run when idle.
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
| `create_watch` | Wake this agent. A time and no condition is the former alarm. A condition, a tool chain, or an event source runs without the model until it matches. See REQ-WC-6. |
| `list_watches` / `cancel_watch` | Inspect / remove |
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

### REQ-AG-15 Address an agent from a workspace thread

- **Bin:** explicit requirement
- **Statement:** From a normal thread of a workspace, the user shall send a message to an agent instance that may use that same workspace. The user shall be able to start from the agent's name and choose the instance when the name alone does not identify it.
- **Origin:** User, 2026-10-31 (quote: "I would like to be able to send messages to agents from every normal thread (where the agent is part of the same workspace) for example by writing @ and then the name of the agent like \"Entertainer 4\" and a small selection dialog should help with finding the correct instance ID." and "das Menü … ist dann leider abgeschnitten" and "es kommt als Nachricht eben auch das @ mit an, also nicht nur die Nachricht").
- **Rationale:** The user is already in the workspace conversation and wants to hand that agent a message without leaving the thread or copying an id.
- **Fit criterion:** A workspace thread is open. Entertainer 4 may use that workspace. The user addresses Entertainer 4 from that thread and the message is delivered to that instance. An instance that may not use the workspace is not offered. When more than one instance fits the name, the user sees each instance id and chooses one before the message is sent. The choice list is fully visible, not clipped by the chat. A message is not delivered to an instance the user did not choose when the name matches more than one. The text that arrives is the message the user wrote after the name. The name used to choose the instance is not part of that text.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-16 A reply goes back to the sender

- **Bin:** explicit requirement
- **Statement:** When an agent chooses to answer a message, that answer shall go back to the sender. The sender may be another agent or a workspace thread. Writing in the agent's own channel shall not be how that answer is delivered. The agent shall be able to address one recipient at a time. The agent does not have to answer every message.
- **Origin:** User, 2026-10-31 (quote: "ich hätte erwartet, dass er mir dann an meinen Thread zurück antwortet" and "wenn der Instructor dem Entertainer eine Nachricht schickt, dann kriegt der Entertainer die Nachricht, er antwortet aber einfach nur in seinem normalen Chat" and "nicht alles, was sie in ihren normalen Chat schreiben, geht an alle Empfänger" and "empfängerspezifisch kommunizieren" and "an Threads schreiben oder an andere Agents" and "er muss ja nicht antworten" and "FROM THREAD NEW THREAD THREAD_ID=… WORKSPACE_ID=… Feels a bit complicated. I think FROM THREAD_ID=… should suffice" and "the Agent did not respond to the thread" and "I think the thread ID is wrong, because that message was sent by a different thread").
- **Rationale:** The agent's own channel is a conversation with the user. A sender who is waiting in a thread, or another agent who asked something, never sees an answer that stays in that channel. A long sender line makes the thread id easy to copy wrong.
- **Fit criterion:** A workspace thread sends a message to an agent. The line the agent sees names that same workspace thread by its thread id. That id is not the agent instance's own thread id. The line does not add the thread title or the workspace id. The agent answers that sender. The answer shows up in that thread. An agent sends a message to another agent. The reply the second agent addresses to the first shows up for the first agent. Text the agent only writes in its own channel stays there. An agent that does not answer leaves the sender without a reply.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-17 An agent knows who it is

- **Bin:** explicit requirement
- **Statement:** Every agent shall know that it is an agent, its name, its instance id, and the task from its definition. It shall know that a message names the sender and that sender's id, and that answering that sender is done by addressing that id, whether the sender is an agent or a thread.
- **Origin:** User, 2026-10-31 (quote: "Die Agents haben anscheinend keine richtige Vorstellung davon, wer sie sind" and "das ist mein Name … das ist meine ID und das und das ist meine Aufgabe" and "dass er überhaupt weiß, dass er ein Agent ist" and "wenn mir jemand schreibt, dann kann ich ihm über das Tool antworten" and "wissen, wem sie antworten, ob sie einem anderen Agenten antworten oder ob sie einem Thread antworten" and "Agents still not respond to other agents via the tooling, but only in their own chat" and "instruct the LLM to always respond via the message_agent tool, if they are responding to a message they received via message_agent" and "does an agent see when a message came from message_agent?").
- **Rationale:** Without its own id and the sender's id, the agent cannot address a reply, and it treats every incoming message as part of its chat with the user. A reply written as chat text never reaches the sender.
- **Fit criterion:** The instructions an agent receives name it as an agent and include its name, its instance id, and its definition task. An incoming message from an agent includes that agent's instance id and says it arrived via message_agent. An incoming message from a thread includes that thread's id and the same marker. The instructions say that an answer to that line is a message_agent call, and that an assistant message in this chat does not deliver it. Those instructions do not also tell the agent that it is the workspace assistant.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-26 The reply flow is shown by example

- **Bin:** explicit requirement
- **Statement:** When a sender asks an agent to get something from someone else and bring it back, the result shall go back to that sender through message_agent. Writing the result in the agent's own chat shall not count as bringing it back. When the sender names the other agent only roughly, the agent shall call list_agents before messaging them. An answer shall be the answer, not a copy of the request.
- **Origin:** User, 2026-10-31 (quote: "Das weniger intelligente Model hat aber 'n Problem, genau es so auszuführen, dass wenn man ihm über Message Agent eine Nachricht schickt, dann diese auch zu beantworten" and "Wenn man aber erst das intelligente Model einsetzt und dann auf das weniger intelligente wechselt, dann sieht das weniger intelligente Model quasi, wie das intelligente Model den Flow gemacht hat … ich frage den anderen Agent, er antwortet mir" and "ob unsere Instruktion nicht konkret genug, nicht greifbar genug sind, sondern zu abstrakt" and "ob wir unsere Instruktionen im Agent Prompt vielleicht noch klarer dahingehend auch mit Beispielen ausschmücken sollten" and "dass ein Agent, der eine Anfrage über dieses Message Agent Tool bekommt, dass der auch wieder dann zu dem korrespondierenden Thread oder Agent antwortet" and "dass der User mit einer groben Beschreibung des anderen Agent die anweisung gibt und zunächst list_agents aufgerufen werden muss" and "Why are there things in curly brackets?!?" and "Why did it write that square bracket stuff?!?" and "I dont get it why does it (Entertainer 14) send the same message back?!?" and "Iam not sure if the examples are too complicated, but it still does not work completely." and "Did you tell me the joke?" and "I think those changes were a regression and that we should change our approach.").
- **Rationale:** A weaker model treats a message_agent line as the human in its own chat, so it writes the result there. A worked example in the instructions gets copied, including the request sent back as the answer. The same model does call message_agent when that turn states the recipient id and the text.
- **Fit criterion:** A direct "Tell me a joke" from another agent is not answered in the agent chat. The model input for that turn names that sender as the message_agent recipient. A first request to ask some other agent by a rough name does not name a recipient yet, so the next call can be list_agents. After that list returns, the model input names the matched agent and the request to pass on. When that agent's answer arrives, the model input names the original sender and the answer text. A later question from the original sender is sent back with message_agent. The instructions do not contain a sample sender line or a sample JSON object. A workspace thread's instructions do not include these rules.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-23 An agent prompt is not a workspace-thread prompt

- **Bin:** explicit requirement
- **Statement:** The instructions an agent instance receives shall be its own: who it is, the task from its definition, where it is working, and how its tools work. They shall not open by saying it is the workspace assistant, and they shall not impose that assistant's voice.
- **Origin:** User, 2026-10-31 (quote: "For agent the prompt is a bit weird. First this is being told: You are keimling, a careful Linux CLI assistant. Later: You are an agent instance named \"Instructor 8\". So now thats confusing even to me and there are probably more examples. I think you should better organize the prompt and make sure the context you build for the agent, actually makes sense. I think currently you jjust dump the system prompt of a normal thread in the context, but that is probably the reason why the agents are confused and have trouble following instructions.").
- **Rationale:** Two identities in one prompt fight each other. The definition is the task. The workspace-thread voice tells the model to be a different assistant and to stay almost silent.
- **Fit criterion:** An agent instance's instructions start from its name and instance id, then its definition task, and do not contain "You are keimling, a careful Linux CLI assistant" or the workspace thread's "EXTREMELY SHORT" voice. A workspace thread's instructions still contain that assistant identity.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-18 A message stands on its own

- **Bin:** explicit requirement
- **Statement:** An agent shall know that another agent or a workspace thread does not share its context. When it sends a message, that message shall carry what the recipient needs in order to understand it.
- **Origin:** User, 2026-10-31 (quote: "wenn sie einen anderen Agenten zu irgendetwas fragen, dass dieser nicht den gleichen Kontext hat" and "der Alarm ist losgegangen und deswegen frage ich dich jetzt. Das ist aber natürlich was, was der Empfänger überhaupt nicht einordnen weiß" and "sie in ihrem eigenen Kontext arbeiten" and "diese nicht den gleichen Kontext haben und in ihrer eigenen Welt leben" and "Sender-Empfänger-Problem, was sie eigentlich jetzt genau schicken").
- **Rationale:** The recipient has its own chat, its own alarms, and its own tools. A reference to "the alarm" or "what just happened" is empty there.
- **Fit criterion:** The instructions an agent receives say that another agent or a thread does not see this chat, these alarms, or these recent events, and that a message has to name the thing it is about so the recipient can place it.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-19 Stop one channel without locking the rest

- **Bin:** explicit requirement
- **Statement:** Stopping shall apply to the agent or thread the user stopped, and it shall take effect at once. That run shall not consume further tokens and shall not start another command. A command that run still has running, including one left in the background, shall be cancelled. The rest of the app shall stay usable: change agent, thread, or workspace, and stop another one. Another agent or thread shall not keep that stopping state.
- **Origin:** User, 2026-10-31 (quote: "wenn ich dann aber den Agenten wechsel zu einem anderen, dann steht da immer noch Stopping" and "das scheint nicht direkt immer sich an dem aktuellen Thread oder Agenten zu orientieren" and "ich will ihn ja nicht die ganze Zeit blockiert sein" and "ich möchte, dass das dann möglichst schnell für mich einfach wie ein normales, benutzbares System ist und ich auch andere Agenten stoppen kann" and "dass wir vielleicht einen Moment brauchen, um endgültig zu stoppen" and "nicht, dass dauerhaft mein Chatfenster da blockiert ist" and "in der Seitenleiste kann ich auch auf den Stopp-Button drücken, aber auch der scheint irgendwie nicht richtig was auszulösen" and "the chat still allows to press Interrupt and the agents just keeps working" and "First there is stopping in the chat history, but then it just disappears and the agent continues" and "does a stop also pause the alarms?" and "When I say Stop I want it to stop immediately, like in not further token consumption and no more command executions, in fact it should cancel any active commands (also background commands)").
- **Rationale:** Stop means abort that run. Letting the model finish a reply, or leaving a command running, is the run continuing.
- **Fit criterion:** After stop, that run starts no further model request and leaves no command running, including a command whose shell already returned. Switching to another agent does not leave that other chat saying Stopping. The sidebar stop and the chat interrupt stop that same run. A due alarm does not start a new run after that stop. The alarm is still there. Sending a message to that agent lets it run again.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-20 Reset an instance to a fresh context

- **Bin:** explicit requirement
- **Statement:** The user shall reset an agent instance so its context is that of a fresh instance of the same agent type. The instance itself shall remain.
- **Origin:** User, 2026-10-31 (quote: "the ability to reset any agent instance by giving it a context like a fresh instance of that agent type").
- **Rationale:** The instance already has a name, contacts, and tools. The user wants the conversation forgotten, not a second copy of the agent.
- **Fit criterion:** An instance has a conversation. The user resets it. The conversation and its durable summary are gone. The same instance id, name, contacts, tools, and alarms remain. A new message is answered without the old conversation.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-21 See the raw context

- **Bin:** explicit requirement
- **Statement:** The user shall be able to read the raw context that the token count is measuring, not only the short charter summary.
- **Origin:** User, 2026-10-31 (quote: "I would love it to have the ability to take a look at the raw context" and "when I open the context dialog I sometimes see just one sentence and at the same time it shows 8000 Token usage. WTH are those 8000 tokens?").
- **Rationale:** The charter can be one sentence while the request that was counted also contains the instructions, the tool definitions, and the transcript.
- **Fit criterion:** The context view shows the charter and the raw parts that were sent: instructions, tool definitions, and the transcript items. Each part shows its size. The token number is identified as the full request, not as the charter sentence alone.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-24 An agent chat does not choose its model

- **Bin:** explicit requirement
- **Statement:** In an agent chat, the user shall not be able to choose a different model. The model that answers in that chat shall be the model on the definition. When the definition has no model, that chat shall use the model from Settings.
- **Origin:** User, 2026-10-31 (quote: "in agent chats it should not be possible to change the model" and "in general the model should be defined in the definition of an agent").
- **Rationale:** A model picked in the chat is easy to confuse with the model the definition names, and it is not obvious which one will answer next.
- **Fit criterion:** An agent chat shows the model that will answer and does not offer a way to change it. A model choice stored earlier from that chat does not change which model answers. The next message uses the definition’s model, or the Settings model when the definition has none. A workspace thread still lets the user choose that thread’s model.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-25 One instance may use another model

- **Bin:** explicit requirement
- **Statement:** The user shall be able to set a model on one instance. When that instance has no model of its own, it shall use the definition’s model. Setting a model on one instance shall not change the definition or any other instance.
- **Origin:** User, 2026-10-31 (quote: "maybe we should allow changing the model in the instance editor, because sometimes it can be helpful but in general the model should be defined in the definition of an agent").
- **Rationale:** The definition is the usual model. One instance sometimes needs a different one, without changing every instance of that definition.
- **Fit criterion:** The instance editor can save a configured model id, or clear it. After it is saved, that instance’s next message uses it. Another instance of the same definition still uses the definition’s model. Clearing the instance model makes the next message use the definition’s model again.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-22 A schedule and a reply stay with the agent that owns them

- **Bin:** explicit requirement
- **Statement:** When the user tells an agent to ask another agent for something on a schedule, that agent shall set its own alarm and shall ask the other agent for the thing. The other agent shall not be asked to set that alarm. A reply shall go back to the agent who asked. It shall not be rewritten as a delivery to the user.
- **Origin:** User, 2026-10-31 (quote: "alle fünf Minuten sollst du dem Entertain Agent auffordern, dir einen Witz zu schicken" and "How stupid is that to set an alarms yourself and also let the other agent set an alarms?!?" and "Please deliver a joke to the user" and "the joke should be just an exchange between the agents. Why do they decide it should be for the user?!?").
- **Rationale:** The user named who asks, who answers, and where the answer goes. Turning that into two alarms and a delivery to the user drops the request.
- **Fit criterion:** The instructions say that create_watch wakes only this agent, that another agent is not asked to create that watch, and that a reply to another agent goes back to that agent rather than to the user. A watch line is described as this agent's own reminder. The tool name follows REQ-WC-6.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-27 An agent definition chooses ask, auto, or yolo

- **Bin:** explicit requirement
- **Statement:** The user shall choose the approval mode for an agent definition. The modes are ask, auto, and yolo. An instance shall use its definition's mode and shall not have a mode of its own. Workspace threads shall keep the mode they already use.
- **Origin:** User, 2026-11-04 (quote: "man will natürlich auch den Modus auswählen, ne? Also man kann das auch auf Ask oder was anderes stellen oder auf YOLO" and "bei der Instanz braucht man es ehrlicherweise nicht, sondern nur in der Definition").
- **Rationale:** An agent is meant to keep working when nobody is sitting in its channel. The mode of an ordinary conversation is the wrong switch for that.
- **Fit criterion:** A definition saved as auto makes every instance of it use auto, including an instance that already existed. There is no mode field on the instance. A second definition saved as ask does not change the first. A workspace thread stays on the mode set for conversations. Changing the definition to yolo makes the next tool call of its instances use yolo.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-28 The auto review prompt is set on the definition and can be reset

- **Bin:** explicit requirement
- **Statement:** When a definition's mode is auto, the user shall be able to replace the whole factory review prompt with a text on that definition. Reset shall drop that text so the definition is bound to the factory prompt again. An instance shall not have a review prompt of its own. An empty text shall not be saveable.
- **Origin:** User, 2026-11-04 (quote: "wir brauchen für jeden Agenten die Möglichkeit, den Autoprompt standardmäßig anzupassen. das sehe ich sogar in der Definition des Agenten, nicht nur bei den Instanzen" and "bei der Instanz braucht man es ehrlicherweise nicht" and "wenn ich auf Auto stelle, will ich auch den Prompt anpassen können. Mit einer ähnlichen Mechanik, wie wir es an anderen Stellen haben, also dass ich den Prompt von Auto wieder zurücksetzen kann und er dann wieder an den Systemprompt gebunden ist").
- **Rationale:** The factory review stance is right for ordinary work and wrong for an agent whose job is a standing exception. The instance would only repeat the definition.
- **Fit criterion:** With no saved text, a review for that definition sends exactly the factory prompt. After a distinctive sentence is saved, the next review sends exactly that sentence and does not contain the factory prompt. Reset removes the saved sentence, and the next review sends exactly the factory prompt again. Saving an empty or whitespace-only text is refused and leaves the previous text in effect. Two instances of the definition use the same text. The review model stays the one from Settings.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-29 Auto refuses instead of asking, and ask warns that it can block

- **Bin:** explicit requirement
- **Statement:** For an agent in auto, a tool call that still needs a decision shall be judged by that definition's review prompt. If the judgment is to ask the user, or no judgment can be obtained, the call shall be refused and the run shall continue. The agent shall not wait. For an agent in ask, that question shall be asked in the agent's main channel. Where the user sets an agent definition's mode to ask, the product shall say that ask can block the agent.
- **Origin:** User, 2026-11-04 (quote: "wenn bei einem Agent im Automodus die Frage aufkommt, ob er etwas tun sollte, dann sollte es standardmäßig abgelehnt werden. Im Automodus. Im Ask-Modus darf er gerne in seinem Hauptkanal die Frage stellen" and "in den Einstellungen und der Definition von Agenten drauf hinweisen, dass, wenn ich im Agenten Ask auswähle, dass das den blockieren kann").
- **Rationale:** An agent in auto has nobody sitting on a prompt. Waiting there stalls the run. Ask is still available, and it should be obvious that the run then stops until the main channel answers.
- **Fit criterion:** An agent in auto proposes a tool call that is not already cleared. The review prompt approves it and the tool runs, with no question shown. The same kind of call is judged as ask and does not run; the channel shows the refusal; the run continues to the next step. A review that cannot be completed refuses the call the same way. An agent in ask shows the question in its main channel and waits there. The definition's mode control, when set to ask, shows that this can block the agent. A workspace thread in auto still asks the user and still keeps the floor the review prompt cannot override.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-30 A conversation can read the instructions a new agent starts from

- **Bin:** explicit requirement
- **Statement:** A conversation shall be able to read the instructions a new agent definition starts from, so a later save can change that text instead of beginning from an empty prompt.
- **Origin:** User, 2026-11-04 (quote: "der Standard, äh, Prompt von 'nem Agenten auch irgendwie dem System bekannt sein. Ähm, damit man darauf aufbauen kann und den vielleicht modifizieren, weiterentwickeln, damit man nicht immer bei null anfängt").
- **Rationale:** Developing an agent prompt inside a conversation needs a known starting text. An empty field makes every new agent begin from nothing.
- **Fit criterion:** The read returns the same instructions that a new definition stores when no instructions are given. A definition saved with empty instructions contains that text.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-31 A conversation can save an agent definition

- **Bin:** explicit requirement
- **Statement:** A workspace thread and an agent instance shall be able to create an agent definition and to replace its instructions later. A new definition's mode is auto. Replacing the instructions leaves the mode as it is. An empty instruction text on a later save leaves the saved instructions.
- **Origin:** User, 2026-11-04 (quote: "ein Tool, mit dem man Agenten und Teams sowohl anlegen" and "aus einem Chat heraus direkt … den Prompt vom Agenten zu entwickeln" and "das Werkzeug zum Anlegen und Instanzieren von Agenten kann von überall aus ausgeführt werden, so wie das andere Werkzeug auch").
- **Rationale:** The prompt is developed in the conversation that is already open. The same action has to be available on a workspace thread and on an agent.
- **Fit criterion:** A workspace thread creates a definition whose instructions are the text it supplied, or the starting text when it supplied none, and whose mode is auto. The same call from an agent instance does this. A later call with new instructions changes the instructions and leaves the mode. A later call with empty instructions leaves the instructions.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-32 A conversation can start an agent in the current workspace

- **Bin:** explicit requirement
- **Statement:** A workspace thread and an agent instance shall be able to start an agent instance from a definition in the workspace the call is running in. An agent shall not start one in a workspace it cannot use.
- **Origin:** User, 2026-11-04 (quote: "Agenten und Teams sowohl anlegen, instanzieren" and "eventuell gibt es auch Use Cases, wo es interessant ist, dass ein Agent einen anderen instanziert, so ähnlich wie man ja auch Delegate oder Submodelle nutzen kann").
- **Rationale:** Starting an agent is a step in the same conversation, including when one agent needs another the way a thread uses a delegate.
- **Fit criterion:** A call from a workspace thread creates an instance of the named definition bound to that thread's workspace. A call from an agent instance creates an instance bound to that agent's current workspace when the agent may use it. A call for a workspace the agent may not use creates no instance.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-33 A conversation can save and start a team

- **Bin:** explicit requirement
- **Statement:** A workspace thread and an agent instance shall be able to save a team of definitions and to start that team in the workspace the call is running in, one instance per definition.
- **Origin:** User, 2026-11-04 (quote: "genauso würde ich auch Teams von Agenten, ähm, ähnlich persistieren wollen" and "ein Tool, mit dem man Agenten und Teams sowohl anlegen, instanzieren").
- **Rationale:** A team is the set of agents that should start together. The conversation that defines them is the place that starts them.
- **Fit criterion:** A call saves a team that names existing definitions. A later call in a workspace starts one instance of each definition in that workspace. The same calls work from an agent instance that may use the workspace.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-34 The existing tool grant decides who may do this

- **Bin:** explicit constraint
- **Statement:** Creating and starting agents and teams shall be native tools, available wherever the other native tools are available. A definition that grants every tool shall include them. A definition with a fixed tool list shall include them only when that list grants them.
- **Origin:** User, 2026-11-04 (quote: "kann von überall aus ausgeführt werden, so wie das andere Werkzeug auch" and "Agenten haben ja bereits eine Mechanik, mit der man Tools abschalten kann, wenn man das möchte oder anschalten und die können wir ganz normal nutzen").
- **Rationale:** Agents already have a per-definition tool switch. A second switch for this action would split the same decision.
- **Fit criterion:** A workspace thread is offered the tools. An agent whose definition grants every tool is offered them. An agent whose definition lists tools and omits these is not offered them. Adding them to that list offers them.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-35 A workspace thread can reach and stop an instance

- **Bin:** explicit requirement
- **Statement:** A workspace thread shall be able to message an agent instance in that workspace, stop that instance, and delete that instance. Stopping shall not ask for confirmation. Deleting an instance shall ask for confirmation. In yolo, neither shall ask.
- **Origin:** User, 2026-11-07 (quote: "der Thread selber konnte dann nicht den Agenten, den er eben gestartet hat, kontaktieren" and "ein Thread sollte die Fähigkeit bekommen, auch Agenten zu stoppen" and "Agenten Instanzen stoppen sollte ein Thread ohne Bestätigung können" and "auch im Modus Ask nicht" and "Agenten löschen sollte eine Bestätigung haben" and "nur im Modus YOLO … braucht er für keine dieser Sachen irgend eine Bestätigung. Also auch fürs Agenten löschen dann nicht").
- **Rationale:** The thread that started an instance is the place the user says to stop or change that work. Asking again for a stop blocks that. Deleting the instance removes it, so that step stays a confirmation except when the mode asks for nothing.
- **Fit criterion:** From a workspace thread, `list_agents` shows instances that may use that workspace. `message_agent` delivers the text to one of those instances as a message from this thread. `stop_agent` holds that instance in ask, auto, and yolo without a confirmation. `delete_agent` asks in ask and auto, and does not ask in yolo. An instance outside the workspace is not addressed.
- **Priority:** shall
- **Status:** confirmed

### REQ-AG-36 Auto allows a call the confirmed purpose already names

- **Bin:** explicit requirement
- **Statement:** When an agent runs in auto, a tool call whose concrete action is named in the job the user confirmed shall be allowed. The check shall see the command inside a watch, not only the tool name. A job that does not name the action shall not allow it. A call that names a different command or target shall not be allowed by that job.
- **Origin:** User, 2026-11-07 (quote: "wenn die Befehle dem Zweck entsprechen, dann kann der Auto-Modus es automatisch freigeben" and "dafür muss er sich aber natürlich auch angucken, was da bei CreateWatch genau ausgeführt wird" and "die transitiven Freigaben noch nicht zuverlässig genug funktionieren").
- **Rationale:** The user already confirmed the job by starting the agent. Asking again, and then refusing because an agent cannot wait, blocks the job that was just confirmed.
- **Fit criterion:** An agent in auto whose job names `ping -c 1 -W 2 192.168.0.101` is allowed to create a watch that runs that command, with no review and no refusal. A watch whose command is not in that job is still reviewed. A vague job does not allow the ping. A workspace thread gets no such grant.
- **Priority:** shall
- **Status:** confirmed

## Design notes (not requirements)

Until the user saves tools on a definition, that definition grants every tool. Saving stores the checked tools. An instance with no narrower set uses that definition. Saving the same set again clears the instance override. REQ-AG-2’s unanswered default is this design, not a stakeholder shall.

Tool choices are shown as a topic table with a read column and a write column. The stored grant is still the individual tool names. A cell that is only partly granted stays mixed until the user clicks it, so saving does not silently add the rest. An instance cell lists only the tools the definition allows.

MCP servers are a second table: one checkbox per server, and that server’s tool names beside it. The stored choice is the server, so tools it exposes later stay included. A saved `mcp` choice from before this table still means every server until the user saves again.

Contacts are set by the user on the instance. Saving the set writes the same link on both instances. A message is allowed when either instance lists the other, so an older one-way link still lets the other side reply. The model is not offered a tool that adds contacts. A definition may store a model id. Empty means Settings. An instance may store its own model id. Empty means the definition. The agent chat shows that model and does not open a picker. The choice is read again for each new message. A run already started keeps the model it started with. Compaction, autopilot, and the review model stay on Settings. The review prompt for an agent definition is REQ-AG-28. A call that is already cleared (routine work inside the workspace, a whitelisted MCP tool, or a tool that never asks) does not go through that prompt. For an agent in auto, every other call does, including file edits and MCP calls that a workspace thread would still force onto the user. Approve runs the tool. Ask, or a missing review, refuses it. The factory prompt still says to ask for privileged, destructive, and secret-sensitive commands, so those are refused unless the saved prompt says otherwise. A saved prompt replaces the factory text in full, as an autopilot prompt does. yolo asks nothing and does not use the prompt. The ask warning sits on the definition's mode control. Reset of the review prompt discards typed text; a confirm belongs on that control as a design default. The refusal is visible on the tool line in the main channel. An unknown id is rejected when saving. If one is already stored and no longer configured, the chat uses Settings and says so. When the same id exists on more than one provider, the Settings provider wins if it has that id.

A message from A to B is a line in A’s channel ("To B") and, when B’s worker runs, a line in B’s channel ("From A"). The host starts B’s worker when a message is waiting. The checkbox screen is the take-back: uncheck and save again. Delete of an instance asks by name and is not beside Save. There is no restore after that confirm.

A conversation reads the starting instructions, saves a definition, and starts an instance with native tools (`agent_instructions`, `save_agent`, `spawn_agent`, `save_team`, `spawn_team`). The stored definition stays the JSON recipe. The saved text is the job; identity and tool rules are still wrapped around it at run time. Empty instructions on create store the starting text. Empty instructions on update keep the saved text. The mode of a new definition is auto, and an update does not change it. Start uses the workspace the call is running in and does not take a workspace argument. An agent whose current workspace is outside its ACL gets an error and no instance. When a host is reachable the host creates the instance and starts its worker. With no host the instance is saved and does not run. A team start creates one instance per definition and lets those instances message each other. The grant table has a Define agents row: read is the starting prompt, write is save and start for agents and teams. Reading the starting prompt does not ask. Save and start follow ask, auto, and yolo like other tools that are not cleared in advance: ask asks, workspace auto reviews, agent auto reviews and refuses when that review would ask, yolo runs. The way back is the existing definition and instance delete in Manage. A definition that still has an instance is still not deleted.

Typing `@` in a workspace thread opens a list of instances that may use that workspace. The list is positioned over the page so the chat's overflow does not clip it. Each row shows the name and the instance id. Choosing one inserts the name into the draft and remembers that id. Nothing is sent until the user sends the draft. Escape or editing the draft drops the choice. A name that matches one instance is enough; a name that matches several is not sent until one id is chosen. The workspace model does not run for that send. The thread shows who received it. The addressed instance is woken with the message text after the name is removed. The wake line is `[From thread_id=<id> via message_agent]`, so the id can be copied exactly and the agent can see the line arrived through message_agent. That id is the workspace thread the message was recorded on. An agent instance keeps its own thread id on the instance. It does not create a workspace thread with that id, and the workspace thread list does not show it. The chat shows the sender without that marker. A call that prefixes `agent-` onto that thread id still reaches the thread. `message_agent` accepts another agent's instance id or a thread id in a workspace this agent may use. A reply into a thread is a line in that thread, not a new run of the workspace model. The agent's own instructions include its name, instance id, and that its chat with the user is not a broadcast. They also include a worked example of answering a thread, answering another agent, and receiving that agent's reply, with sample ids marked as samples. When the user names another agent only roughly, the example calls list_agents before message_agent. set_alarm wakes only that agent. A schedule the user gave it is its own alarm, not a request that the other agent create one. A reply to another agent goes back to that agent. An `[Alarm]` line is a reminder that agent wrote for itself.

Stopping is remembered per agent id and, for a workspace run, per thread id. The chat interrupt and the sidebar stop both cancel that agent's running worker: the model request is aborted, and every command process group that run still has, including one left behind by a shell that already returned, is killed. The chat button says Stopping only until that abort has landed, and it does not offer a second Interrupt. A user stop sets a hold so due alarms and queued peer wakes do not start another run. The alarms stay on the instance. A message the user sends to that agent, from its chat or from a thread, lifts the hold. Reset archives the transcript, clears the charter and the pending queue, and leaves contacts, tools, and alarms. The context dialog's raw section is the instructions, the tool definitions, and the transcript items, with a size on each part.

New ids use a dash before the nanosecond fraction. Ids that already contain a period stay valid. While a workspace run is busy, the user can open another thread, queue a message there, and open settings. The running journal stays the one the run is writing. Opened tool lines and chapter folds stay in the page when a later message is appended. The definition model field suggests ids from the configured providers. Manage tabs use `--fg` on a transparent background, because the default button color `--accent-ink` is the ink for a filled accent button and is dark in the dark theme. Dialog text uses `--fg`. `--ink` is not a theme variable.
