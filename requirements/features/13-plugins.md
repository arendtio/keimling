# Feature: Agent Plugins

## REQ-PL-1 A settings source is a Git URL or a directory

- **Bin:** explicit requirement
- **Statement:** The user shall be able to add a plugin source in settings as either a Git repository URL or a directory where that plugin is already checked out, and shall be able to turn that source off or remove it.
- **Origin:** User, 2026-11-04 (quote: "Plugin-Support einfach implementieren und kann man irgendwo in den Einstellungen, 'ne URL … für ein Git-Repo hinterlegen, wo 'n Plugin drin ist oder auch alternativ ein Verzeichnis, in dem das Plugin ausgecheckt ist, angeben").
- **Rationale:** A team shares one checkout or one repository. Settings is where the other install-wide choices already live.
- **Fit criterion:** Adding an absolute directory keeps that path. Adding an `https`, `http`, `ssh`, or `git@` URL stores that URL and checks the repository out under Keimling's data directory. Turning the source off leaves the checkout in place and stops using its components. Removing a Git source deletes that checkout. Removing a directory source leaves the directory on disk.
- **Priority:** shall
- **Status:** confirmed

## REQ-PL-2 An enabled source contributes its skills and MCP servers

- **Bin:** explicit requirement
- **Statement:** An enabled source that is an Agent Plugin shall make that plugin's skills and MCP servers available the same way skills and MCP servers from the existing configuration are available.
- **Origin:** User, 2026-11-04 (quote: "Plugin-Support einfach implementieren" after the discussion of sharing skills and MCP servers across a team through the portable plugin package).
- **Rationale:** The package exists so a team does not re-enter each skill and each server by hand.
- **Fit criterion:** A directory with a valid `plugin.json`, a skill under `skills/`, and an `mcp.json` server is listed with that plugin name. The skill appears in the skill registry. The server appears in the MCP configuration. A source that is off, or a directory without a valid manifest, contributes neither.
- **Priority:** shall
- **Status:** confirmed

## Design notes (not requirements)

One source is one plugin root, the directory that contains `plugin.json`. A marketplace catalog is not read. The manifest `$schema` must be `https://agent-plugins.org/schemas/1.0.0/plugin.schema.json`. Skills are the immediate children of `skills/`. MCP servers come only from `mcp.json` with the matching MCP schema. `${PLUGIN_ROOT}` and `${PLUGIN_DATA}` expand in stdio `args`, `env`, and `cwd`. `./` commands are resolved inside the plugin root. The subprocess working directory is the plugin root when `cwd` is omitted. Unknown servers are skipped and reported on the source. A skill or server name the user already has stays the user's; the plugin fills in names that are not already configured. The user types the URL or path and presses Add; that is the confirmation. The row's checkbox and Remove are the take-back. The MCP list does not remove a server that exists only because the plugin is on.
