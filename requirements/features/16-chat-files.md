# Feature: Chat files

## REQ-CF-1 A file in the chat can be of any type

- **Bin:** explicit requirement
- **Statement:** The user shall be able to add a file of any type to a chat message.
- **Origin:** User, 2026-11-06 (quote: "the user should be able to add files to the chat (no restrictions)").
- **Rationale:** The user already has the file and wants the agent to work from it.
- **Fit criterion:** A text file, an image, and a file with an uncommon extension can each be added to one message. The message keeps the file name after it is sent.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-2 A referenced workspace file stays where it is

- **Bin:** explicit requirement
- **Statement:** When a file is already in the workspace and the user refers to it in the chat, the file shall remain at that path.
- **Origin:** User, 2026-11-06 (quote: "wenn eine Datei schon im Workspace liegt … und wir sie quasi nur irgendwie erwähnen im Chat, dann kann sie da ja ruhig liegen bleiben").
- **Rationale:** Mentioning a project file is not a request to copy or move it.
- **Fit criterion:** After a message refers to an existing workspace file, that file is still at the same path and no second copy of it has been written into the workspace.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-3 A dropped file is temporary until the user asks to keep it

- **Bin:** explicit requirement
- **Statement:** A file newly dropped into the chat shall be stored as a temporary file outside the workspace. The agent shall copy it into the workspace only when the user asks.
- **Origin:** User, 2026-11-06 (quote: "Wenn wir jetzt eine neue Datei per drag and drop in den Chat einfügen … standardmäßig sollte das eher sowas wie eine temporäre Datei sein … wenn er die Anweisung hat, sie irgendwie in den Workspace zu kopieren, kann er das auch gerne machen"). User, 2026-11-06 (quote: "lasse ich das Bild fallen und dann öffnet der Webview dieses Bild und die eigentliche Applikation verschwindet … als wenn er die Location von der Website in dem Moment replaced durch die URL der Datei").
- **Rationale:** A drop is material for this message, not yet a project file. Opening that file in the window replaces the chat.
- **Fit criterion:** After a drop, the stored file is outside the workspace and the workspace does not contain a new copy. The agent can read the temporary path. A later request to copy it can place a copy in the workspace. A drag from the file manager does the same as the File control: the message gets a temporary copy named with the file’s own name, and the composer does not receive the `file:///` text. The window stays on the chat; it does not open the dropped file in place of the UI.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-4 The model reads text directly and uses tools for the rest

- **Bin:** explicit requirement
- **Statement:** The model shall receive the contents of a text-like file. For a file it cannot read that way, it shall be able to use tools on the file.
- **Origin:** User, 2026-11-06 (quote: "The model should either use command line tools to make use of those file or read them directly (when they are text like txt, md, html, ...)").
- **Rationale:** Text is useful immediately. Other formats stay on disk for tools such as OCR.
- **Fit criterion:** A short `.md` file attached to a message is present as text in the model input, together with its path. A non-text file is present as a path the tools can open.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-5 Seeing and hearing follow the model catalog

- **Bin:** explicit requirement
- **Statement:** When a model is added, Keimling shall detect whether that model can see images and hear audio. A detected capability shall be on by default and the user shall be able to turn it off. A capability the catalog does not report shall not be offered.
- **Origin:** User, 2026-11-06 (quote: "wenn ich ein Model hinzufüge, dass er dann so eine Art … Capability Detection macht … man sollte das vielleicht deaktivieren können … wenn das Model die Capability nicht hat, dann macht es ja auch keinen Sinn, das überhaupt zu aktivieren. Standardmäßig sollte es … aktiviert sein").
- **Rationale:** The switch is only meaningful for a model that has the sense.
- **Fit criterion:** A catalog entry that reports image input stores seeing as available and on. A catalog entry that reports neither image nor audio stores neither switch. Turning seeing off keeps the capability recorded and stops sending the picture.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-6 The user can ask for OCR whether or not the model can see

- **Bin:** explicit requirement
- **Statement:** The user shall be able to ask the agent to OCR an image both when model seeing is on and when it is off.
- **Origin:** User, 2026-11-06 (quote: "selbst wenn das Model die Capability hat, trotzdem dem Agenten als User sagen, dass er OCR machen soll … die Möglichkeit gibt es ja sowieso immer").
- **Rationale:** Seeing is an extra channel. The tool path remains an instruction the user can give.
- **Fit criterion:** With seeing on, the image is included for the model and the path is still available for tools. With seeing off, the image bytes are not included and the path is still available.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-7 @ refers to a file and still addresses an agent

- **Bin:** explicit constraint
- **Statement:** In the window and in the terminal, `@` followed by a path shall refer to that file. `@` followed by an agent name shall still address that agent.
- **Origin:** User, 2026-11-06 (quote: "Im Terminal würde ich erwarten, dass man … eine Datei … referenzieren kann … ok, dann lass uns auch ein @ nutzen, um Dateien zu referenzieren").
- **Rationale:** One mark is enough when the name and the path are different.
- **Fit criterion:** `@NOTES.md` on an existing file refers to that file and leaves it in place. `@Entertainer` still addresses the agent of that name. A token inside a word, such as an email address, is neither.
- **Priority:** shall
- **Status:** confirmed
