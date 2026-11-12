# Feature: Visible run outcomes

## Intent

When a run stops, a tool fails, autopilot decides, or compaction fails, that
outcome is visible in the chat. It may sit in a collapsed row. It must not
exist only in a channel the GUI ignores.

Origin: the user saw a tool call and then silence, and asked that every such
event be visible on the surface, collapsed if needed, without a new panel.

## Requirements

### REQ-VE-1 Transcript

Outcomes that explain what happened or why a run ended are messages in the
same transcript as chat and tool cards.

Fit: after a model call fails following a tool call, the thread shows the tool
card and a line containing the error. Reopening the thread still shows both.

### REQ-VE-2 Tool state

A failed tool is labeled Failed. A tool cut off by Stop is labeled Stopped.
The card stays collapsed. Stop adds a `Stopped` line.

Fit: a non-zero exec and a user stop are distinguishable with the card closed.

### REQ-VE-3 Next turn

The next model call receives the same outcome note.

Fit: the API input for the following turn contains the note text.

### REQ-VE-4 What stays ephemeral

Progress such as thinking, generating, and tokens per second is visible while
the run is going and does not have to remain afterward. The CLI text stream
is not rendered as a second chat log.

### REQ-VE-5 Tool name once

The closed tool line shows the tool name once. Arguments follow that name.
The name may be bold.

Origin: User, 2026-11-12 (quote: "Ich habe das gefühl da steht die der Methodenname immer zweimal. aber einmal würde reichen. Fett gedruck ist ok.").

Fit: A call shown as `RAN mcp_tools mcp_tools {"query":…}` shows `RAN` and `mcp_tools` once, then the arguments. The same for `mcp__Operations__project_status`.
