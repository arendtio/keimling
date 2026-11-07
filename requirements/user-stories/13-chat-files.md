# User stories: Chat files

## US-45 — Refer to a file with @

**As** a user, **I want** to name a file with `@` in the window or the terminal, **so that** the agent reads that file and the project copy stays where it is.

**Acceptance:** traces REQ-CF-2 and REQ-CF-7. `@NOTES.md` keeps the workspace file at its path. `@Entertainer` still addresses that agent.

## US-46 — Drop a file into the chat

**As** a user, **I want** to drop a file into the chat, **so that** the agent can read it without that drop becoming a project file.

**Acceptance:** traces REQ-CF-1 and REQ-CF-3. The dropped file is stored outside the workspace. Removing the chip before send takes the file off the message. Dragging a local file onto the chat stores that same temporary copy and shows the file name. The composer text stays free of `file:///`. The chat stays on screen; the window does not navigate to the file.

## US-47 — Let the model see or hear only when it can

**As** a user, **I want** seeing and hearing to turn on when the model catalog says the model can do that, **so that** I can switch them off and still ask for OCR.

**Acceptance:** traces REQ-CF-4, REQ-CF-5, and REQ-CF-6. A text file is included as text. An image path remains either way. While seeing is on, the bytes are not in the main input; interpretation is US-50. A model with no reported capability has no switch.

## US-50 — Ask about an image without keeping the pixels

**As** a user, **I want** the agent to look at an attached image through a delegate and keep only the answer, **so that** the picture can be asked about again without its bytes staying in the conversation.

**Acceptance:** traces REQ-CF-8, REQ-CF-9, and REQ-CF-10. The main input shows the path and no image bytes. A delegate call with that path answers the user's question in the user's words and includes an earlier fact the question depends on. It does not replace the question with a request to describe every detail. A second call can use the same path. With seeing off, that call sends no bytes and the path remains for tools.
