# User stories: Live call

## US-44 — Talk in the open chat

**As** a user, **I want** a live voice call in the thread or agent I already have open, **so that** I can speak instead of only typing.

**Acceptance**

- Settings has a live-call provider, model, and voice, separate from transcription.
- Call is available on a workspace thread and on an agent when that setting is on.
- Hang up ends the call. The transcript stays in the chat.
- Mute turns the microphone off without leaving the call. The open spoken turn still closes.

## US-45 — The chat and the call stay one conversation

**As** a user, **I want** the call to know the chat and to write what was said back into it, **so that** I can keep going in text afterwards.

**Acceptance**

- The call starts from the history the next text turn would see.
- Spoken user and assistant lines show in the thread as they finish, and remain after reload.
- A message sent during the call is stored and handed to the call. The text model does not run until hang up.
- A tool asked for on the call uses that chat's tools and approval.

## US-46 — The call keeps its chat

**As** a user, **I want** a live call to stay on the thread or agent where I started it, **so that** another chat does not look like it is in that conversation.

**Acceptance**

- During a call, another thread, another agent, and another workspace stay closed.
- The control says to hang up first.
- After hang up, that switch opens the chosen chat.
