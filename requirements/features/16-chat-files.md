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
- **Fit criterion:** With seeing on or off, the path stays available for tools such as OCR. With seeing on, interpretation follows REQ-CF-8. With seeing off, the image bytes are not sent.
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

## REQ-CF-8 An image is interpreted aside from the main conversation

- **Bin:** explicit requirement
- **Statement:** The main conversation shall keep a reference to an attached image and shall not keep the image bytes. A question about that image shall be answered by a separate request that receives the question and the image and returns only text. The same reference shall be enough to ask again.
- **Origin:** User, 2026-11-07 (quote: "das Bild weiterhin, also eine Referenz auf das Bild weiterhin in unserem Kontext bleibt, damit wir auch eine zweite Anfrage an das gleiche Bild stellen können … aber dass der eigentlichen Bilddaten aus dem Kontext bleiben" and "der Vision-Teil, also das Interpretieren des Bildes immer … durch einen Delegate … sodass quasi unser Hauptmodell eine Anfrage definiert, was möchte ich aus diesem Bild erfahren").
- **Rationale:** Image bytes are large and would be sent again on every later turn. A later question still needs to name the same image.
- **Fit criterion:** After a message attaches an image and seeing is on, the main model input contains the path and does not contain the image bytes. One delegate call with that path sends the bytes on that call only and the text answer is what the main conversation keeps. A later main model input still contains the path and still does not contain the bytes, so a second delegate call can use the same path. With seeing off, a delegate call with an image sends no bytes.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-9 Image interpretation uses delegate

- **Bin:** explicit constraint
- **Statement:** Interpreting an attached image shall go through the existing delegate, extended so that call can carry the image.
- **Origin:** User, 2026-11-07 (quote: "ok, dann bitte delegate entsprechend erweitern und auf verarbeitung über delegate umstellen").
- **Rationale:** The separate look should be the delegate the product already has, not a second kind of side request.
- **Fit criterion:** The call that receives the image bytes is a delegate call. Its arguments name the path and the question. The tool result contains the text answer and does not contain the image bytes.
- **Priority:** shall
- **Status:** confirmed

## REQ-CF-10 The user's question is the question that is answered

- **Bin:** explicit requirement
- **Statement:** The delegate shall be asked the user's question in the user's words. Facts from the conversation that the question depends on shall be included. The question shall not be replaced by a broader description of the image.
- **Origin:** User, 2026-11-07 (quote: "die Anfrage über delegate verfälscht die Abfrage" and "Was siehst Du auf dem Bild?" became "Describe everything visible in this image in detail…". "es ist wichtig, dass die Anfrage des Users auch richtig an Delegate mit allem relevanten Kontext übergeben wird", example: "Meine Lieblingsfarbe ist blau" and then "Wieviele Sachen in meiner Lieblingsfarbe siehst Du?").
- **Rationale:** A rewritten question produces a different answer. A short question expects that level of answer, and a later question can depend on something said earlier.
- **Fit criterion:** When the user asks "Was siehst Du auf dem Bild?" the delegate request carries that question and does not carry a replacement request to describe every detail. When an earlier user message states a fact the new question depends on, that fact is in the same delegate request.
- **Priority:** shall
- **Status:** confirmed
