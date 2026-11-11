# User stories: Screen control

## US-60 — Let the assistant use the shared screen

**As** a user, **I want** to add the keyboard and mouse to a screen I am already sharing, **so that** the assistant can click and type there, and I can take that back.

**Acceptance:** traces REQ-SS-4, REQ-SS-5, and REQ-SS-6. The picture can be shared on its own. The picture and the keyboard and mouse switch are in the Now view beside the chat. Keyboard and mouse are a separate switch on that share. While the switch is off, a click or key from the model does nothing to the desktop. While it is on, a click and typed text land in the shared picture. If the picture is a window, turning the switch on keeps that window. Turning it off stops the next action. This works on Linux, macOS, and Windows.
