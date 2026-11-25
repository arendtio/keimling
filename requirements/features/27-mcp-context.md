# Feature: MCP tools in context

## REQ-MCP-1 Load a server's tools when the job needs them

- **Statement:** The model shall see one short line per connected MCP server and not that server's tool definitions. The line shall name the server and say what it is for, in at most about 200 characters, using the server's own instructions when it sends them. One search shall find those tools and the internal tools, and it shall load up to five matches. A loaded MCP tool shall then be callable in that chat, and it shall leave again when that chat's context is compacted. Another chat shall not receive it. The built-in tools shall stay loaded unless Reduce context is on. This shall hold for every connected server, including one with few tools.
- **Origin:** User, 2026-11-11 (quote: "wenn ich jetzt viele MCP-Server hab, die viele Tools enthalten, dann ist es häufig so, dass der Kontext allein schon durch die ganzen MCP-Server-Tools relativ voll ist" and "nur die MCP-Server-Tool-Definitionen auch wirklich in den Kontext mit reinnimmt, die der Agent gerade nutzen möchte" and "jeder MCP-Server hat eine Kurzbeschreibung, die nicht mehr als 200 Zeichen sein darf" and "eine Funktion, mit der man sich die Tools für den MCP-Server" and "beim Komprimieren des Kontextes fliegt es dann wieder raus"). Same day, after the comparison with Claude Code, Cursor, and OpenCode: load a few matching tools rather than a whole server, keep that set out of the system prompt, and drop it when the context is compacted. User, 2026-11-12 (quote: "die 10% regelung finde ich blöde, verursacht ja trotzdem kosten" and "dynamisches laden bitte für alle MCP Tools" and "Was wir aus meiner Sicht immer mitschicken könnten wäre eine einfache liste aller verfügbaren MCP server"). Same day (quote: "nur eine Suche und die sollte eigentlich immer alles finden, sowohl die eingebauten Tools als auch die MCP Tools" and "woher kommen die anderen drei, die jetzt zumal auch noch MCP Server Tools sind? Ich sehe aber nicht, dass er in irgendeiner Art und Weise hier einen MCP Server gerade genutzt hat").
- **Rationale:** A full schema for every tool is paid on every turn, including turns that never call one. A small server still costs that on every turn. The system prompt is kept across compaction, so a load that lives there would not leave. A second search would make the model choose and would cost another definition.
- **Fit criterion:** With one connected MCP server and one tool, the prompt names that server in one short line and does not include the tool schema. One search, given that server's name, returns the tool, and that tool can be called by name. After compaction it is gone and the server line remains. The same search also returns an internal tool. Built-in tools such as reading a file are present when Reduce context is off. A tool loaded in one chat is absent from a new chat's request. With no server connected, the prompt has no server list. The request does not advertise a separate MCP search.
- **Priority:** shall
- **Status:** confirmed

## REQ-MCP-2 Approve a tool program before it starts

- **Statement:** When the model runs a program that calls tools, the product shall allow or ask once, before that program starts. The decision for each tool the program will call shall be the same as for a single call: a tool on the whitelist, or a tool ask mode already runs without a question, does not make the program ask. If every such tool would run, the program runs. If any would ask, the program is the question and does not start until it is allowed. After that answer, it does not stop to ask again. A call to a tool that was not part of that decision does not run.
- **Origin:** User, 2026-11-12 (quote: "Prüfen und Stehenbleiben im Ask-Modus, dass man das an den Werkzeugen festmacht" and "wenn alle Werkzeuge, die verwendet werden, eh in der Whitelist drin sind oder von Ask als unkritisch eingestuft werden, dann darf das ganze Programm durchlaufen" and "wenn's aber einen Fall gibt, wo 'n Tool genutzt wird, was nicht einfach laufen darf, was von Ask um 'ne Erlaubnis bittet, dann würde man das ganze Programm zum Ask stellen" and "die Freigabe zur Durchführung muss vor dem Start des Programmes getätigt werden und nicht während das Programm läuft").
- **Rationale:** One program is one intention. Stopping it in the middle leaves a half-finished run and asks again for tools the user already has in view.
- **Fit criterion:** A program whose tools are all whitelisted or already non-asking runs with no question and performs those calls. A program that includes one tool ask mode would question shows one question naming the tools it will call, and performs none of them until allow. After allow, those calls run without a further question. Deny runs none. A call whose tool was not in that decision does not run.
- **Priority:** shall
- **Status:** confirmed

## REQ-MCP-3 Run chat MCP calls together, up to a chosen limit

- **Statement:** In a normal chat, MCP calls shall run in the background and in parallel. The user shall set how many MCP calls may be in progress at once. That limit starts at ten.
- **Origin:** User, 2026-11-12 (quote: "auch ein normaler Chat, MCP-Aufrufe im Hintergrund machen können und Parallelität" and "in den Einstellungen aber gern irgendwie 'n Limit einstellen können, dass man sagt, maximal zehn parallele Aufrufe gleichzeitig im Hintergrund laufen können sollen").
- **Rationale:** Several independent calls should not wait out each other's full duration, and the chat should stay usable while they run. The user wants a ceiling they can change.
- **Fit criterion:** A chat response that requests several MCP calls starts them together, and never more than the saved limit are in progress at once. The setting shows the limit, starts at ten, and a change survives a restart. The user can interrupt the run while those calls are in progress.
- **Priority:** shall
- **Status:** confirmed

## REQ-MCP-4 One stdio server at a time, several servers together

- **Statement:** Calls to one stdio MCP server shall not run in parallel. Calls to different stdio servers may run at the same time.
- **Origin:** User, 2026-11-12 (quote: "'n SCDIO-Server kann ich nachvollziehen, der sollte nicht unbedingt jetzt mit Parallelität belästigt werden" and "wenn man mehrere SCDIO-Server hat, vielleicht will man ja mehrere verschiedene Server parallel bedienen, das wäre dann wieder in Ordnung").
- **Rationale:** One stdio server is one process on one connection. A second call on that connection is a different matter from calling two servers.
- **Fit criterion:** Two calls to the same stdio server do not overlap. A call to one stdio server and a call to another may overlap. Both still count toward the limit in REQ-MCP-3.
- **Priority:** shall
- **Status:** confirmed

## REQ-MCP-5 A start script runs before any server connects

- **Statement:** Keimling shall run one shell command, chosen in settings, before it connects MCP servers. The command shall run when Keimling starts and when every server is reloaded. When the command fails, no MCP server shall be connected. A later reload shall try the command again.
- **Origin:** User, 2026-11-20 (quote: "manche MCP-Server, für die muss ich vorher so 'n Proxy aufbauen" and "so was wie 'n Startskript … immer, wenn er startet, er dieses Skript ausführt" and "Die betroffenen Server nicht verbinden, bis das Skript durchläuft"). Same day, the user chose both a start script and a script per server.
- **Rationale:** Some servers are unreachable until a proxy process is up. The proxy is a shell script the user already has.
- **Fit criterion:** With a start command that exits non-zero, every enabled MCP server stays disconnected and reports that failure. A command that exits zero is followed by the servers connecting. Reload all runs the command again. A disabled server stays disabled. The command is allowed five minutes and then stopped.
- **Priority:** shall
- **Status:** confirmed

## REQ-MCP-6 A server can run a command before it connects

- **Statement:** Each MCP server, including one that comes from a plugin, shall be able to run a shell command before that server connects. When that command fails, that server shall stay disconnected and the others shall still connect. Reloading that server shall try the command again. The command for a plugin server shall not change the plugin's files.
- **Origin:** User, 2026-11-20 (quote: "bevor ein MCP-Server aus diesem Plugin gestartet wird, muss vorher folgendes Skript ausgeführt werden" and "Das ist normalerweise 'n Shell-Skript"). Same day, the user chose a script per server in addition to the start script, and chose not to connect a server whose script fails.
- **Rationale:** One proxy belongs to one server. A plugin must stay as it was shipped.
- **Fit criterion:** A server whose command exits non-zero stays disconnected, and another server whose command exits zero still connects. Saving the command on a plugin server leaves the plugin files unchanged. Reload of that server runs the command again. The command is allowed five minutes and then stopped.
- **Priority:** shall
- **Status:** confirmed

## REQ-MCP-7 A script that is still running or has failed stays on the chat header

- **Statement:** While the start script or a per-server script is running, the chat header shall show that it has not finished and that the affected servers are not connected. After the script fails, that warning shall stay until a later run succeeds. A finished successful script shall leave the header quiet.
- **Origin:** User, 2026-11-20 (quote: "irgendwo anzeigt, dass dieses Skript noch nicht beendet ist. Oder eben auch fehlgeschlagen" and "oben in der Mitte 'ne große Warnung sehr deutlich" and "nichts ärgerlicher, als wenn man denkt, alle MCP-Server laufen normal und dann laufen sie aber gar nicht"). Same day, the user set the script limit at five minutes because a script waits on a sign-in.
- **Rationale:** The script can wait for a confirmation for minutes. A failure that only appears inside MCP settings looks like the servers are up.
- **Fit criterion:** While a script command is still running, the header between Chapters and Sidebar says it is still running. After a command exits non-zero, the header names the failure and offers reload. After a command exits zero and the servers are not blocked by a script, the header warning is gone.
- **Priority:** shall
- **Status:** confirmed

## REQ-MCP-8 A server can carry a note for the model

- **Bin:** explicit requirement
- **Statement:** Each MCP server shall be able to carry a note the user writes. The note shall be sent with the information the model already receives about that server, and only while that server is connected. A server with an empty note shall add nothing. A disconnected server shall not send its note.
- **Origin:** User, 2026-11-25 (quote: "auf Pro-MCP-Server, dass ich auch Pro-MCP-Server nochmal System Prompt Instructions mitgeben kann" and "die aber natürlich nur ausgespielt werden sollen, wenn der MCP-Server irgendwie aktiv ist" and "wenn da auch Informationen abgerufen werden an Infos zu dem MCP-Server" and "manche MCP-Server, die bestimmte Macken haben").
- **Rationale:** One server has a quirk. The model needs that note when it is looking at that server, and the note is noise on every other turn.
- **Fit criterion:** A connected server with the sentence "Page numbers are 1-based" includes that sentence next to that server's line. A second connected server with an empty note does not. Disconnecting the first server removes its sentence from the next request. The note is still there after a restart.
- **Priority:** shall
- **Status:** confirmed
