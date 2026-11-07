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

## REQ-PL-3 Adding a source recognizes a marketplace or a plugin

- **Bin:** explicit requirement
- **Statement:** When the user adds a Git URL or a directory, Keimling shall look in the marketplace and plugin locations Cursor and Claude use, and shall add a marketplace when a catalog is there and a single plugin when only a plugin manifest is there.
- **Origin:** User, 2026-11-07 (quote: "dass wir es automatisch erkennen. Das wäre wahrscheinlich noch sinnvoller, dass man einfach an den entsprechenden Orten guckt, ob man da was findet und je nachdem, was man findet, fügt man es entsprechend hinzu").
- **Rationale:** The same marketplace repositories other coding agents already use should work here, without a separate choice at add time.
- **Fit criterion:** A checkout that contains `.cursor-plugin/marketplace.json` or `.claude-plugin/marketplace.json` with a name, an owner, and a plugin list is stored as a marketplace. A checkout that contains only a plugin manifest (`plugin.json` at the root, `.cursor-plugin/plugin.json`, or `.claude-plugin/plugin.json`) and no marketplace catalog is stored as one plugin. The add action does not ask which kind it is.
- **Priority:** shall
- **Status:** confirmed

## REQ-PL-4 An enabled marketplace turns on every listed plugin

- **Bin:** explicit requirement
- **Statement:** An enabled marketplace shall enable every plugin in its catalog. There is no separate switch per plugin.
- **Origin:** User, 2026-11-07 (quote: "wenn ich 'n Marketplace habe, dann sollten automatisch alle Plugins an sein").
- **Rationale:** The marketplace is one source, the same way one plugin source is on or off.
- **Fit criterion:** Adding a marketplace whose catalog lists three plugins makes all three contribute without a further per-plugin choice. Turning that source off stops all three. Turning it on again resumes all three.
- **Priority:** shall
- **Status:** confirmed

## REQ-PL-5 A marketplace plugin contributes the components Cursor and Claude ship

- **Bin:** explicit requirement
- **Statement:** A plugin from a marketplace shall contribute the component kinds those plugins carry: rules, skills, commands, agents, and MCP servers. Keimling shall accept slash commands so a command component can be invoked. Rules apply as instructions in the same role as `AGENTS.md`.
- **Origin:** User, 2026-11-07 (quote: "sollen die Plugins vom Marketplace auch alle Komponenten laden die es in Cursor und Claude gibt. Rules sind ähnlich wie Agents.md und skills müssen wir ggf. aber auch noch implementieren, damit es dann zusammen passt" and "wir in der GUI aktuell keine Slash commands verfügbar haben sehe ich sowieso noch als konzeptionslücke an die gefixt werden sollte").
- **Rationale:** A marketplace is the same package as today's plugin, listed from one catalog. The composer has to be able to run the commands those packages contain.
- **Fit criterion:** An enabled plugin that contains a rule, a skill, a command file, an agent definition, and an MCP server contributes each of them. The rule is available as instructions. The skill appears in the skill registry. The command can be started by its slash name from the composer. The agent definition can be started. The MCP server appears in the MCP configuration. A component kind the product has no place for (language-server integration, color themes, editor-tab hooks) contributes nothing.
- **Priority:** shall
- **Status:** confirmed

## REQ-PL-6 Running code from a marketplace plugin is granted once, by name

- **Bin:** explicit requirement
- **Statement:** A marketplace plugin that needs a command, an exec, or a hook shall not run that code until the user initializes it once, sees what it does, and activates it. That activation grants that named command, exec, or hook. A later update that changes the script or hook shall keep the grant while the name stays the same.
- **Origin:** User, 2026-11-07 (quote: "die Marketplace-Plugins, die eben ein Command oder ein Exec oder ein Hook brauchen, dass man die initial einmal initialisieren muss und dabei quasi freigibt und aktiviert" and "ein Update des skripts/hook braucht keine erneute freigabe. So lang der name gleich bleibt, kann es aktiviert bleiben").
- **Rationale:** The user looks once at what will run. The name is the identity of that grant, so an update of the same hook does not ask again.
- **Fit criterion:** Before initialization, a hook, exec, or command entry in the plugin does not run. After the user activates it, that entry runs under its name without a further grant. Replacing the body of a hook or script whose name is unchanged does not ask again, and the new body runs. An entry whose name was not activated does not run.
- **Priority:** shall
- **Status:** confirmed

## Design notes (not requirements)

REQ-PL-1 and REQ-PL-2 describe a single Agent Plugin. For that layout, one source is the directory that contains `plugin.json`. The manifest `$schema` must be `https://agent-plugins.org/schemas/1.0.0/plugin.schema.json`. Skills are the immediate children of `skills/`. MCP servers come only from `mcp.json` with the matching MCP schema. `${PLUGIN_ROOT}` and `${PLUGIN_DATA}` expand in stdio `args`, `env`, and `cwd`. `./` commands are resolved inside the plugin root. The subprocess working directory is the plugin root when `cwd` is omitted. Unknown servers are skipped and reported on the source. A skill or server name the user already has stays the user's; the plugin fills in names that are not already configured. The user types the URL or path and presses Add; that is the confirmation. The row's checkbox and Remove are the take-back. The MCP list does not remove a server that exists only because the plugin is on.

REQ-PL-3 supersedes the earlier note that a marketplace catalog is not read. The catalog files are `.cursor-plugin/marketplace.json` and `.claude-plugin/marketplace.json`. When both a catalog and a plugin manifest exist, the catalog is the source. Language-server integration, color themes, and editor-tab hooks have no place in this product; the user accepted that boundary on 2026-11-07. Slash-command files are prompts. The one-time grant in REQ-PL-6 covers entries that run: hooks, exec, and a command that executes.
