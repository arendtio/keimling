# Feature: DataLang

## REQ-DL-1 Tool output can be transformed without host code

- **Bin:** explicit requirement
- **Statement:** The agent shall be able to filter, parse, search, aggregate, and transform results of other tools without generating Python, shell, or other host code.
- **Origin:** User, 2026-11-07 (quote: "damit die Modelle nicht immer python aufrufen müssen um daten zu filtern oder zu transformieren. Also Input sollen beliebige tool aufrufe dienen, also beispielweise soll man den Output eines MCP server aufrufes mit dem Tool direkt filtern können").
- **Rationale:** Data work is common, and a shell or Python program is the wrong tool for it.
- **Fit criterion:** `data_processing` filters a stored MCP-shaped object and parses a CLI transcript into typed values. The program does not invoke a shell or Python.
- **Priority:** shall
- **Status:** confirmed

## REQ-DL-2 Transformation has no host side effects

- **Bin:** explicit requirement
- **Statement:** A transformation shall not read or write files, open a network connection, start a process, read environment variables, or call other tools.
- **Origin:** User, 2026-11-07 (quote: "diese Sprache relativ gefahrlos einfach immer von einem LLM genutzt werden kann, ohne das die GEfahr besteht, dass Dateien manipuliert oder das System verändert wird").
- **Rationale:** The model may run a transform on every tool result, including untrusted ones.
- **Fit criterion:** A program that names a file, a network, or a process operation fails with an unknown-function error and does not change the host. A non-terminating program stops with a timeout.
- **Priority:** shall
- **Status:** confirmed

## REQ-DL-3 Large tool results stay out of the model context

- **Bin:** explicit requirement
- **Statement:** A large tool result shall be stored in the harness and passed on as a reference. The model shall be able to transform that result without the original output having been placed in full into the model context.
- **Origin:** User, 2026-11-07 (quote: "Result Store und Referenzübergabe zuerst bauen. Ohne diese Architektur wäre DataLang zwar ein sicherer Parser, würde aber den entscheidenden Vorteil verlieren: große MCP-Ergebnisse verarbeiten, ohne sie vollständig in den Modellkontext zu laden").
- **Rationale:** Transforming a result after it was already sent to the model does not save the context.
- **Fit criterion:** After a tool result larger than the inline limit, the text recorded for the model contains a reference and a preview and does not contain the full payload. `data_processing` with that reference still returns the filtered rows.
- **Priority:** shall
- **Status:** confirmed

## REQ-DL-4 The binary does not grow by a large dependency

- **Bin:** explicit constraint
- **Statement:** The implementation shall not pull a large dependency into the Go binary just to host this language.
- **Origin:** User, 2026-11-07 (quote: "darauf zu achten, dass Umsere Implementierung die Go-Binary nicht unnötig vergrößert in dem große dependencies reingezogen werden").
- **Rationale:** Keimling stays a single small binary.
- **Fit criterion:** The nogui binary grows only by the language implementation itself. No new module is added for the interpreter, the grammar engine, or a sandbox runtime.
- **Priority:** shall
- **Status:** confirmed

## REQ-DL-5 One transform can use several stored results

- **Bin:** explicit requirement
- **Statement:** One transformation shall be able to use more than one stored tool result, so the agent can join, look up, or compare them.
- **Origin:** User, 2026-11-07 (quote: "mehrere input references finde ich durchaus wichtig und sollten jetzt bereits eingebaut werden, um z.B. Dinge wie einen join/lookup oder Mengenoperationen zu ermöglichen").
- **Rationale:** A filter of one list is not enough when the answer depends on a second tool result.
- **Fit criterion:** Given two stored lists, a program keeps the rows of the first whose key occurs in the second and attaches a field from the matching row of the second. Neither list is copied into the program text.
- **Priority:** shall
- **Status:** confirmed

## REQ-DL-6 The timeout defaults to 30 seconds and has no artificial cap

- **Bin:** explicit requirement
- **Statement:** The default execution limit shall be 30 seconds. The agent shall be able to request any other limit that fits in the integer used for that limit.
- **Origin:** User, 2026-11-07 (quote: "5 Sekunden finde ich als timeout zu kurz. 30 als STandard ist okay und es sollte kein hartes Limit geben, also das LLM darf einne beliebigen wert angeben (maximum definiert durch datentyp der Zahl)").
- **Rationale:** A large parse can need longer than a few seconds, and the agent is the one who knows how long this run should be allowed to take.
- **Fit criterion:** Omitting the timeout uses 30 seconds. A requested timeout of 45 seconds is not reduced. A request at the top of the integer range is accepted up to the longest duration the clock type can represent.
- **Priority:** shall
- **Status:** confirmed

## REQ-DL-7 The program that ran stays visible

- **Bin:** explicit requirement
- **Statement:** The tool card shall show the DataLang program and the references it used, including when the program fails.
- **Origin:** User, 2026-11-07 (quote: "man gar nicht sieht, welche parameter an data_processing übergeben wurden, nicht mal im fehlerfall").
- **Rationale:** A failed attempt cannot be told apart from the next one when the card only shows the tool name.
- **Fit criterion:** A `data_processing` call whose arguments are longer than 160 characters still shows the program text in the command field of the tool card, on success and on a syntax error.
- **Priority:** shall
- **Status:** confirmed

## REQ-DL-8 A run can take a file or a chat section

- **Bin:** explicit requirement
- **Statement:** A transformation shall be able to take a file path or a section of this chat as an input, the same way it takes a tool result. The program shall still not open a file or the journal itself. A large result of that run shall be returnable as a reference.
- **Origin:** User, 2026-11-12 (quote: "dass man Referenzen reintun kann, Referenzen damit produzieren kann, dass man aus der Chat-Historie Referenzen ziehen kann"). Same day, confirming the shape: the harness resolves the address before the program runs.
- **Rationale:** The model should name an address it already has. Copying a file or a stretch of the chat into the program text puts that text into the conversation.
- **Fit criterion:** A run whose input is a file path returns a value taken from that file, and the program text does not contain the file. A run whose input is a history section id from search returns a value taken from that section. A program that calls a file or journal operation fails and does not read either. A large result can come back as a reference.
- **Priority:** shall
- **Status:** confirmed
