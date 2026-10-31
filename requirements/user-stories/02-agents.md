# User stories: Agents

## US-10 — Create an agent from a definition

**As** a user, **I want** to pick or create a **definition** then name an **instance**, **so that** I reuse prompts.

**Acceptance:** picker + create definition; instance name; ACL = current workspace; no default agent.

## US-11 — Stable sidebar while chatting with an agent

**As** a user, **I want** workspace threads to stay visible when an agent is open, **so that** I can jump back.

**Acceptance:** Agents under Threads; paper shows agent main thread; thread click returns to workspace; green only while busy.

## US-12 — Manage definitions

**As** a user, **I want** to edit and delete definitions, **so that** I can maintain the catalog.

**Acceptance:** Edit name/instructions; delete fails if instances still use the definition.

## US-13 — Alarms

**As** an agent, **I want** relative and absolute alarms, **so that** I can wake myself later.

**Acceptance:** `set_alarm` / `list_alarms` / `cancel_alarm`; persist across worker restart; a due alarm starts the instance when its process is not running, then the wake runs when idle.

## US-14 — Inter-agent messaging

**As** an agent, **I want** to message peers on my ACL, **so that** agents can collaborate.

**Acceptance:** `list_agents` and `message_agent`; the user sets contacts in Manage agents; the link works both ways; durable queue.

## US-15 — Teams

**As** a user, **I want** to save a set of definitions as a team and spawn them, **so that** I can start a group quickly.

**Acceptance:** create/list/delete team; spawn creates instances and wires peer ACL between them.

## US-23 — Keep working while a run continues

**As** a user, **I want** to change thread, agent, or workspace, open settings and other dialogs, and queue a message in any thread while a run continues, **so that** I am not stuck on that run.

**Acceptance:** traces REQ-AG-11. The run's record stays intact. A message queued on another thread waits there.

## US-24 — Opened messages stay open

**As** a user, **I want** an opened message to stay open when a later message arrives, **so that** I can keep reading it.

**Acceptance:** traces REQ-AG-12.

## US-16 — Change workspace within ACL

**As** an agent, **I want** `change_workspace`, **so that** I can `cd` among allowed roots.

## US-17 — Choose workspaces and tools for an instance

**As** a user, **I want** to grant each agent instance a subset of the existing workspaces and a subset of the tools, **so that** an agent can work across workspaces without having every tool.

**Acceptance:** traces to REQ-AG-1 and REQ-AG-2. The user sees the existing workspaces and which of them this instance may use, and can work in each granted workspace. The user sees which tools this instance may use. A tool that was not granted cannot be called. The default before a choice is not specified.

## US-18 — Open the same main channel from each granted workspace

**As** a user, **I want** an agent's one main channel to appear in every workspace it may use, **so that** I can read the same process wherever I am.

**Acceptance:** traces to REQ-AG-3. Opening the instance from W1 and from W2 shows one conversation, including requests the user put in.

## US-19 — See a message in both agents' channels

**As** a user, **I want** a message from agent A to agent B to show in A's channel as sent to B and in B's channel as received from A, **so that** each main channel shows that part of the process.

**Acceptance:** traces to REQ-AG-4.

## US-20 — Grant tools by topic, reading apart from writing

**As** a user, **I want** tool access grouped by the kind of work, with reading and writing as separate choices, **so that** I can allow an agent to read files without letting it edit or write them, and see which tools a choice covers.

**Acceptance:** traces to REQ-AG-7. File reading is separate from file editing and writing, and those two are one writing choice. Listing alarms is separate from setting and cancelling them. The names of the tools sit beside the choice. An instance cannot turn on a tool its definition does not grant.

## US-21 — Include or leave out a whole MCP server

**As** a user, **I want** each MCP server as its own choice, with that server’s tools listed beside it, **so that** I can allow one server and not another without guessing which of its tools write.

**Acceptance:** traces to REQ-AG-8. Including a server makes its tools available. Leaving it out does not. The tools are not split into reading and writing. An instance cannot include a server its definition does not include.

## US-22 — Choose which agents may contact each other

**As** a user, **I want** to say which instances may contact each other while managing agents, **so that** they can reply to one another and cannot message everyone.

**Acceptance:** traces to REQ-AG-9. Including B on A lets A message B and B message A. Neither can message an instance that was not included. The user changes this set while managing instances.
