# User stories: Watch commands

## US-51 — Wake only when a command matches

**As** an agent, **I want** a command to run on an interval without a model call, **so that** I am started only when its result matches.

**Acceptance:** traces REQ-WC-1 through REQ-WC-8. The tools are `create_watch`, `list_watches`, and `cancel_watch`. A time with no condition is the former alarm. A shell command, an MCP tool, and a chain that ends in data processing can each be the check. The model stays idle while the result does not match. A value that still matches wakes the agent on every run. A match starts the agent with that result, and that watch stays until the agent turns it off. A file or directory watch and a long-running command wake the agent with no interval. A workspace thread does not set a watch. It knows its own thread id. It tells the user an agent can, and after the user agrees it creates that agent with a job that names the concrete check and that thread id, starts it, and queues the job. A match appears in the agent channel as a Watch report and stays in the open chapter. Deleting the instance removes its watches and stops their processes. Another instance still has its own.
