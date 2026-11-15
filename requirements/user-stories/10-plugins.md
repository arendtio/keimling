# User stories: Plugins

## US-41 — Add a shared plugin from settings

**As** a user, **I want** to point settings at a Git repository or a checked-out directory, **so that** the team’s skills and MCP servers show up without copying them in by hand.

**Acceptance:** traces REQ-PL-1 and REQ-PL-2. The source can be turned off or removed. An enabled valid plugin contributes its skills and MCP servers.

## US-42 — Use a Cursor or Claude marketplace

**As** a user, **I want** to add a marketplace repository the same way I add a plugin, and turn on only the plugins I want, **so that** the rest stay off until I activate them.

**Acceptance:** traces REQ-PL-3, REQ-PL-4, and REQ-PL-5. The kind is recognized from the catalog or the plugin manifest. Listed plugins stay off until the user activates each one, and each one can be turned off again. Rules, skills, slash commands, agents, and MCP servers come from the plugins that are on.

## US-43 — Grant running code from a marketplace plugin once

**As** a user, **I want** to initialize a marketplace plugin that runs a command, an exec, or a hook, **so that** I have seen what runs and it stays active under that name.

**Acceptance:** traces REQ-PL-6. Until initialization, that code does not run. After activation, the same name keeps running, including when a later update changes the script body. A different name stays inactive.

## US-81 — Clone a private plugin with the vault key

**As** a user, **I want** a plugin Git URL to use the vault SSH key for that host when the vault is open, **so that** a private repository works in a container and a machine that already has a key still works without a vault.

**Acceptance:** traces REQ-PL-7. An unlocked key whose service names the host is used for that SSH clone and for a refresh. A locked vault, another host, and an HTTPS URL do not receive it. A refused clone shows Git's output.

## US-82 — Give a plugin Git server its SSH port

**As** a user, **I want** to type the SSH port next to the Git address and read Git's full output when the clone fails, **so that** a server on another port is reachable and the failure tells me why.

**Acceptance:** traces REQ-PL-8. Port 22223 on `git@host:path` is used for the clone and a later refresh, and the path stays `path`. An empty port leaves the address alone. The failure shows Git's lines, and the address and port remain so they can be edited.
