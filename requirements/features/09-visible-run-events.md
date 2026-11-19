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

Progress such as generating and tokens per second is visible while the run is
going and does not have to remain afterward. The CLI text stream is not
rendered as a second chat log. The thinking text itself is REQ-VE-6.

### REQ-VE-6 Thinking text

The readable thinking text shall appear in the transcript while it arrives,
closed until the user opens it, and shall still be there after the run. It
shall not be sent back to the model.

Origin: User, 2026-11-18 (quote: "Gerne per default zugeklappt, aber ich möchte die Möglichkeit haben live das thinking zu sehen").

Fit: While reasoning text is streaming, the transcript has a closed Thinking row whose body grows with that text. Opening it shows the text so far. After the run goes idle, reopening the thread still shows that row closed, and the next model request does not contain the thinking text.

### REQ-VE-7 Tool input and output

Every tool call shall keep its full arguments and its full output available, closed until the user opens the row.

Origin: User, 2026-11-18 (quote: "ich möchte auch bei tool calls immer die Möglichkeit haben input und output mir vollständig anzusehen (auch gerne per default zugeklappt, aber eben möglich)").

Fit: A closed tool row shows the short label. Opening it shows the arguments that were sent and the output that came back, including an output longer than a short preview. A read_file row shows the path and the line numbers, not only the path.

### REQ-VE-8 User messages stay visible

A user message stays outside a folded stretch of a chapter.

Origin: User, 2026-11-19 (quote: "Beim ausblenden/zusammenklappen von Nachrichten, werden auch Nachrichten des Nutzers ausgeblendet. Ich glaube Nutzernachrichten sollte immer eingeblendet bleiben.").

Fit: In a chapter overview, the text of a user message is visible while the fold is closed. Opening the fold is not required to read it. The chapter heading is the only division above that message; there is no horizontal line before it.

### REQ-VE-10 The settled transcript stays put

When a reply finishes, the transcript on screen shall keep the shape it already has. Lists and checkboxes shall keep their spacing. The page shall not draw that reply again in another arrangement, and it shall not open or close a row on its own after the reply is already visible.

Origin: User, 2026-11-19 (quote: "wenn er fertig ist, sieht man kurz die seite und kurz darauf verändern sich dann die abstände von text zu checkbox … und noch einen moment später scheint das automatische ein und ausklappen noch dazu zu kommen" and "immer vor eigenen nachrichten so eine trennlinie … Die Überschrift reicht vollkommen").

Fit: While a reply streams, a task line is already a checkbox with the same gap it has when the reply stops. After the run goes idle, that gap does not change and a row that was open or closed stays that way. A chapter shows its heading and no line before the user's message. Inside one turn the rows share one gap. When the speaker changes, the same larger gap sits before the new voice, under a user message and under a Keimling message.

### REQ-VE-11 Scroll once for a sent message

When a user message is shown, the transcript shall scroll to the bottom once. Lines that arrive after that message shall not pull the page down again. The last line of the transcript shall sit clear of the composer.

Origin: User, 2026-11-19 (quote: "wenn ich eine nachricht abschicke scheint er im chat noch ganz nach unten zu scrollen, das ist etwas anstrengend" and "wenn er eine neue nutzernachricht gerendert hat er immer nach ganz unten im chat scrollt" and "etwas mehr abstand zwischen chatbox und der untersten Zeile des chatverlaufs").

Fit: After sending, the user message is at the bottom of the page. As the reply grows, the scroll position stays. The gap under the last line is wider than the gap inside a turn.

### REQ-VE-9 Follow the live thinking row

The More menu shall offer Follow thinking, off until the user turns it on. While it is on, the thinking row that is receiving text stays open, and that row closes when the text stops. The next thinking row opens on its own. A thinking row the user opened stays open. Turning Follow thinking off leaves rows closed until the user opens one. The choice is still there on the next visit.

Origin: User, 2026-11-19 (quote: "Das thinking feature bitte auch umsetzen"), after the agreed shape: a control in the chat More menu, off by default, that follows only the row currently receiving tokens and closes it when that row finishes, while a row opened by hand stays open.

Fit: With Follow thinking off, a thinking row stays closed while its text arrives. Turning it on opens the row that is receiving text. When the reply or a tool call starts, that row closes. The next thinking row opens. A row the user opened before that stays open. Turning the control off closes nothing the user opened, and the next thinking row stays closed. Reloading the page keeps the control as the user left it.

### REQ-VE-5 Tool name once

The closed tool line shows the tool name once. Arguments follow that name.
The name may be bold.

Origin: User, 2026-11-12 (quote: "Ich habe das gefühl da steht die der Methodenname immer zweimal. aber einmal würde reichen. Fett gedruck ist ok.").

Fit: A call shown as `RAN mcp_tools mcp_tools {"query":…}` shows `RAN` and `mcp_tools` once, then the arguments. The same for `mcp__Operations__project_status`.
