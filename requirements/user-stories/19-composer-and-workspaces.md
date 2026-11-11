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
