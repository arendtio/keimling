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
