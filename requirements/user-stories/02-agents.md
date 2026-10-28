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

**Acceptance:** `set_alarm` / `list_alarms` / `cancel_alarm`; persist across worker restart; wake message queued and run when idle.

## US-14 — Inter-agent messaging

**As** an agent, **I want** to message peers on my ACL, **so that** agents can collaborate.

**Acceptance:** `list_agents`, `allow_peer`, `message_agent`; durable queue; peer ACL enforced.

## US-15 — Teams

**As** a user, **I want** to save a set of definitions as a team and spawn them, **so that** I can start a group quickly.

**Acceptance:** create/list/delete team; spawn creates instances and wires peer ACL between them.

## US-16 — Change workspace within ACL

**As** an agent, **I want** `change_workspace`, **so that** I can `cd` among allowed roots.
