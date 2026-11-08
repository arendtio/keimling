# User stories: Themes, background tools, agent overview

## US-57 — Keep one palette for light and dark

**As** a user, **I want** a theme to be a light palette and a dark palette I can edit in Settings, **so that** the gray chrome and the sheet stay in one family and I can switch with the system scheme.

**Acceptance:** traces REQ-TH-1 and REQ-TH-2. Keimling is the built-in theme. In the dark palette the main background stays the KDE gray and the chat sheet is a darker well, with bubbles and the composer lighter on top. Settings lists themes, and a new theme can be edited, used, and deleted after a confirm. Keimling can be reset and cannot be deleted. The choice survives a restart.

## US-58 — Keep talking while tools run

**As** a user on a live call, **I want** tool calls to run in the background, **so that** the conversation does not wait on them and the assistant comments once they are back.

**Acceptance:** traces REQ-BG-1. Tools from one turn start together. A result within about five seconds is spoken as the result. Only a longer run is described as still waiting, and those results are handed back when they arrive.

## US-59 — See every agent on one page

**As** a user, **I want** an agent page that shows who is running, waiting, or idle and which definition they use, **so that** I can open the right chat when there are many agents.

**Acceptance:** traces REQ-AO-1. The Agents page shows a wrapping grid of blocks, one per agent, each at most 400 pixels wide, with definition and phase. Choosing one opens that agent's chat. Focus and Fullscreen enlarge the page.
