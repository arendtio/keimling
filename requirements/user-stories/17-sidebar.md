# User stories: Sidebar

## US-46 — See the run beside the chat

**As** a user, **I want** the agent's plan, its status, and whether the model is still working in the side area, **so that** I can tell what the run is doing without asking.

**Acceptance:** traces REQ-SB-3, REQ-SB-4, and REQ-SB-5. The plan and the performance view are on one view. Recent token speeds are columns about as wide as two capital letters, as many as fit the side area, read against a typical rate, and a model that stays faster than that rate reads as faster. An opening spike does not set the scale. While the model is thinking, a thin light in the interface green sweeps under the speed columns and meets both ends. Columns above the typical rate stay the same colour and are taller. When the run is waiting for the user, the view reads Done, the columns remain with a zero-height column at the end, the time since the model last checked in does not keep increasing, and the speed is not a live rate.

## US-47 — Give the context more room

**As** a user, **I want** the context in the side area, and larger as a focus overlay or fullscreen, **so that** I can read it without a separate dialog.

**Acceptance:** traces REQ-SB-1. Context opens beside the chat, as a focus overlay, and fullscreen.

## US-48 — See whether an MCP server can run

**As** a user, **I want** each MCP connection listed as usable or not, and to start sign-in from that list, **so that** a server that needs authentication can be made ready without editing it there.

**Acceptance:** traces REQ-SB-6. Editing the server opens Manage MCP servers. The list does not change the server's name, command, or address.

## US-49 — Notice the artifact the agent just opened

**As** a user, **I want** an artifact the agent shows to come to the front, **so that** I see it even when I was looking at the plan or the context.

**Acceptance:** traces REQ-SB-7. The previous side view is still reachable after the artifact comes forward.

## US-50 — Open the side area from one entry

**As** a user, **I want** one Sidebar entry instead of More, **so that** context, artifacts, and autopilot instructions are that area rather than separate dialogs.

**Acceptance:** traces REQ-SB-2. Autopilot instructions keep save, undo, and reset on that surface. After the app is opened again, the side area is open or closed as it was, on the same view, at the same width.

## US-51 — Keep the chat readable beside the sidebar

**As** a user, **I want** the chat to stay a steady width, **so that** a long message or a wide sidebar does not stretch or squeeze the conversation.

**Acceptance:** traces REQ-SB-8. A very long line stays inside the chat column. The side area beside the chat stops widening while the chat is still about 600 pixels wide.
