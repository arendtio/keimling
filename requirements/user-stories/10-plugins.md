# User stories: Plugins

## US-41 — Add a shared plugin from settings

**As** a user, **I want** to point settings at a Git repository or a checked-out directory, **so that** the team’s skills and MCP servers show up without copying them in by hand.

**Acceptance:** traces REQ-PL-1 and REQ-PL-2. The source can be turned off or removed. An enabled valid plugin contributes its skills and MCP servers.

## US-42 — Use a Cursor or Claude marketplace

**As** a user, **I want** to add a marketplace repository the same way I add a plugin, **so that** every plugin it lists is available here.

**Acceptance:** traces REQ-PL-3, REQ-PL-4, and REQ-PL-5. The kind is recognized from the catalog or the plugin manifest. All listed plugins come on with the marketplace. Rules, skills, slash commands, agents, and MCP servers from those plugins are available.

## US-43 — Grant running code from a marketplace plugin once

**As** a user, **I want** to initialize a marketplace plugin that runs a command, an exec, or a hook, **so that** I have seen what runs and it stays active under that name.

**Acceptance:** traces REQ-PL-6. Until initialization, that code does not run. After activation, the same name keeps running, including when a later update changes the script body. A different name stays inactive.
