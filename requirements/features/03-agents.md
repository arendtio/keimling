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
| `list_agents` | Discover peer candidates (shared workspace / peers) |
| `allow_peer` | Add instance id to peer ACL |
| `message_agent` | Enqueue text on a peer’s main thread (ACL) |

## API (host)

| Method | Path | Notes |
| --- | --- | --- |
| `GET`/`POST` | `/api/agents/recipes` | List / create |
| `PUT`/`POST` | `/api/agents/recipes/update` | Update definition |
| `POST` | `/api/agents/recipes/delete` | Delete definition (fail if instances use it) |
| `GET`/`POST` | `/api/agents/instances` | List / create |
| `POST` | `/api/agents/instances/attach\|stop\|delete` | Lifecycle |
| `POST` | `/api/agents/instances/message` | Enqueue+wake `{ id, text, from? }` |
| `GET`/`POST` | `/api/agents/teams` | List / create team |
| `POST` | `/api/agents/teams/spawn` | Spawn instances for team |
| `POST` | `/api/agents/teams/delete` | Delete team |

## Non-goals

- Definition marketplace
- Cross-machine peer messaging
