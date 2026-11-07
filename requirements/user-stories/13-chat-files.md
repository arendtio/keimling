# User stories: Chat files

## US-45 — Refer to a file with @

**As** a user, **I want** to name a file with `@` in the window or the terminal, **so that** the agent reads that file and the project copy stays where it is.

**Acceptance:** traces REQ-CF-2 and REQ-CF-7. `@NOTES.md` keeps the workspace file at its path. `@Entertainer` still addresses that agent.

## US-46 — Drop a file into the chat

**As** a user, **I want** to drop a file into the chat, **so that** the agent can read it without that drop becoming a project file.

**Acceptance:** traces REQ-CF-1 and REQ-CF-3. The dropped file is stored outside the workspace. Removing the chip before send takes the file off the message. Dragging a local file onto the chat stores that same temporary copy and shows the file name. The composer text stays free of `file:///`. The chat stays on screen; the window does not navigate to the file.

## US-47 — Let the model see or hear only when it can

**As** a user, **I want** seeing and hearing to turn on when the model catalog says the model can do that, **so that** I can switch them off and still ask for OCR.

**Acceptance:** traces REQ-CF-4, REQ-CF-5, and REQ-CF-6. A text file is included as text. An image is included only while seeing is on. The path remains either way. A model with no reported capability has no switch.
