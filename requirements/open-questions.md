# Open questions

Unresolved points from stakeholder conversations. A requirement does not invent an answer for these.

## 2026-11-25 — agent spend ceiling (REQ-AG-46, REQ-AG-47)

Answered the same day: the ceiling lasts for the life of the instance, a child's spend does not count against the parent, and the human's own chat has no ceiling. The user changes it where the instance is edited. Spawn may set it at the start. Edit may change it later. Delete does not. The instance cannot change its own.

## 2026-11-25 — purposes (REQ-PM-12, REQ-PM-14)

Answered the same day: review, the output filter, and autopilot are one side-task purpose. Delegates use the main chat model, and that main model is itself a purpose. A purpose points only at a saved model. Transcription stays off until a transcription model is saved and chosen. Speech synthesis is a purpose only; how speech is produced is later. There is no global context length.

## 2026-11-25 — conditional instructions

The user described instructions that would apply when a file type, a project, or a workspace is recognized, and asked whether that belongs on the model and the MCP server or in a separate condition list. REQ-PM-13 and REQ-MCP-8 record the two cases they described with an owner. A condition list is not a requirement until they ask for it.

## 2026-11-25 — idle notice (REQ-IA-1)

Does the CLI have the same notice?

Answered the same day: sound and notification stay quiet while the window is focused. The desktop ping uses the call's audio-element playback, not an audio-context output (REQ-IA-2).
