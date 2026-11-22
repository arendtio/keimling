# User stories: Composer and workspaces

## US-54 — Write without a crowd of buttons

**As** a user, **I want** Send in front of me and the other compose actions one step away, **so that** the composer stays clear while I can still attach, share a screen, use the mic, call, and interrupt.

**Acceptance:** traces REQ-CW-1, REQ-CW-2, and REQ-CW-5. Idle, the composer shows Send and one More control under the field. File and Screen are named inside More. Record, call, Stop, and Send stack to the right of the field as icons with no word on them, including while recording, while a message will wait, and while a run is stopping. Mute and hang-up stay in that stack during a call, and hang-up has its own icon. The field starts directly under the model line and grows with the draft up to about seven and a half lines, then scrolls inside the field. The plus stays on the bottom edge of the composer. The icons are only the vectors those controls use.

## US-55 — Name a workspace

**As** a user, **I want** to name a workspace and manage the list from the switcher, **so that** I can tell two folders with the same folder name apart without a path under the switcher.

**Acceptance:** traces REQ-CW-3. The switcher shows the name and no path under it. Manage adds a workspace with a name and a folder, renames it, and removes it from the list. The directory remains after removal. Adding the folder again restores the list entry.

## US-56 — Read the thread without a working-state header

**As** a user, **I want** the working state gone from above the conversation, **so that** the thread starts at the conversation and I read context in the side area.

**Acceptance:** traces REQ-CW-4. No Working state disclosure sits above the thread. Context still opens in the side area.

## US-57 — Read a chat from another workspace

**As** a user, **I want** to name a chat in another workspace, **so that** this workspace can review that chat without copying it and without changing it.

**Acceptance:** traces REQ-WS-4 and REQ-WS-5. A read of `chat:<workspace>/<thread>` returns text that exists only in that other journal. Editing that journal through the reference leaves the file unchanged. Now shows the open chat's workspace and thread at the bottom and copies `chat:` plus the two ids. An agent chat does not show that footer.

## US-58 — Compose with a thumb

**As** a user on a narrow window, **I want** to swipe the composer between writing, dictation, and a call, **so that** each job has large controls and I do not lose the desktop composer on a wide window.

**Acceptance:** traces REQ-CW-8. A wide window keeps the icon stack. A narrow window shows dots and three pages, and the composer stays at the bottom edge of the screen. With no extra draft, the writing page is at least as tall as dictation and the call, and swiping does not change the composer height. More and Send sit on the bottom of the writing page. The writing field spans the composer. Dictation uses the cassette, mute only while recording, and labeled Cancel and Send. Cancel drops the take and leaves the typed message. The call starts and hangs up from the center, with share and mute only while live. Interrupt is full width and inactive until it can stop something, and it is also on the writing page while a run can be stopped. Swiping does not send or discard.

## US-59 — Keep a long draft in reach

**As** a user, **I want** the composer to grow with what I am writing, **so that** a transcription or a long message stays readable until the screen needs the conversation back.

**Acceptance:** traces REQ-CW-5 and REQ-CW-9. On a narrow window the composer stays within about 70 percent of the screen height, and on a wide window within about 50 percent. Beyond that, the field scrolls. More on a narrow window opens over the chat, by the Plus control, with every entry clickable.

## US-60 — Keep a draft in the chat where it was written

**As** a user, **I want** my draft to stay in the chat I typed it in, **so that** I do not find half a message in another chat and lose it when I switch.

**Acceptance:** traces REQ-CW-10. A draft typed in an agent chat does not appear in a workspace thread, and the other way round. Returning to a chat shows its draft again. Sending clears it. A reload starts empty.
