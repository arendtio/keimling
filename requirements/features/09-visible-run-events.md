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

Every tool call shall use the same card. Closed, it shows the short label. Opened, it shows an Input section and an Output section. The Output section is the full text that came back. When the model received a shorter copy, that copy is shown as well, and the full output starts closed inside the card.

Origin: User, 2026-11-18 (quote: "ich möchte auch bei tool calls immer die Möglichkeit haben input und output mir vollständig anzusehen (auch gerne per default zugeklappt, aber eben möglich)"). User, 2026-11-19 (quote: "bei Tool Calls auch die Parameter sehen kann … die aufzuklappen, sowohl die Eingangsparameter als auch das Ergebnis … für alle Tools gleichermaßen" and the answer "Input, die volle Ausgabe (zugeklappt) und der gefilterte Text wenn es einen gibt").

Fit: A closed tool row shows the short label. Opening a read_file row, an exec row, and a row whose arguments are empty each shows an Input section and an Output section, including an output longer than a short preview. The read_file arguments include the path and the line numbers. An exec row whose model saw a shorter text shows that shorter text when the row is opened, and the full output is a closed section in that same row.

### REQ-VE-8 User messages stay visible

A user message stays outside a folded stretch of a chapter.

Origin: User, 2026-11-19 (quote: "Beim ausblenden/zusammenklappen von Nachrichten, werden auch Nachrichten des Nutzers ausgeblendet. Ich glaube Nutzernachrichten sollte immer eingeblendet bleiben.").

Fit: In a chapter overview, the text of a user message is visible while the fold is closed. Opening the fold is not required to read it. The chapter heading is the only division above that message; there is no horizontal line before it.

### REQ-VE-10 The settled transcript stays put

When a reply finishes, the transcript on screen shall keep the shape it already has. Lists and checkboxes shall keep their spacing. The page shall not draw that reply again in another arrangement, and it shall not open or close a row on its own after the reply is already visible.

Origin: User, 2026-11-19 (quote: "wenn er fertig ist, sieht man kurz die seite und kurz darauf verändern sich dann die abstände von text zu checkbox … und noch einen moment später scheint das automatische ein und ausklappen noch dazu zu kommen" and "immer vor eigenen nachrichten so eine trennlinie … Die Überschrift reicht vollkommen").

Fit: While a reply streams, a task line is already a checkbox with the same gap it has when the reply stops. After the run goes idle, that gap does not change and a row that was open or closed stays that way. A chapter shows its heading and no line before the user's message. Inside one turn the rows share one gap. When the speaker changes, the same larger gap sits before the new voice, under a user message and under a Keimling message.

### REQ-VE-11 Stay with the latest lines

When the page is within 200 pixels of the bottom, growth in the transcript shall move it to the bottom. Growth includes streaming text, a finished message, and a tool result, whether the line is from the user or from the agent. A few pixels above the bottom still count as the bottom. Farther up, growth shall leave the page where it is. Sending one's own message shall move to the bottom even from farther up. Opening another thread shall land at the bottom. Refreshing the current session after the agent finishes shall not jump to the top; readers farther up keep their position. The last line shall sit clear of the composer.

Origin: User, 2026-11-19 (quote: "wenn er neue Nachrichten … dranhängt … vom User kommen oder vom Agenten, dann scrollt er nicht automatisch weiter runter" and "zum Bereich von 200 Pixeln oder so ist völlig in Ordnung, dass er in dem immer nach unten snappt … wenn neue Nachrichten reinkommen"). Same day, the answers were: follow on every growth, including streaming; sending one's own message always jumps. This replaces the earlier same-day wish that a reply stay put after one scroll. The composer clearance from that wish stays (quote: "etwas mehr abstand zwischen chatbox und der untersten Zeile des chatverlaufs"). Follow-up on macOS: "wenn das Gespräch wieder endet … manchmal wieder nach ganz oben".

Fit: With the page 80 pixels above the bottom, a streaming token, a tool card, and a finished line each leave the page at the bottom. With the page 400 pixels above the bottom, those arrivals leave the scroll position unchanged. Sending a message from 400 pixels up lands on the bottom, and the reply that follows stays at the bottom. The gap under the last line stays clear of the composer.

### REQ-VE-9 Follow the live thinking row

The More menu shall offer Follow thinking, off until the user turns it on. While it is on, the thinking row that is receiving text stays open, and that row closes when the text stops. The next thinking row opens on its own. A thinking row the user opened stays open. Turning Follow thinking off leaves rows closed until the user opens one. The choice is still there on the next visit.

Origin: User, 2026-11-19 (quote: "Das thinking feature bitte auch umsetzen"), after the agreed shape: a control in the chat More menu, off by default, that follows only the row currently receiving tokens and closes it when that row finishes, while a row opened by hand stays open.

Fit: With Follow thinking off, a thinking row stays closed while its text arrives. Turning it on opens the row that is receiving text. When the reply or a tool call starts, that row closes. The next thinking row opens. A row the user opened before that stays open. Turning the control off closes nothing the user opened, and the next thinking row stays closed. Reloading the page keeps the control as the user left it.

### REQ-VE-5 Tool name once

The closed tool line shows the tool name once. Arguments follow that name.
The name may be bold.

Origin: User, 2026-11-12 (quote: "Ich habe das gefühl da steht die der Methodenname immer zweimal. aber einmal würde reichen. Fett gedruck ist ok.").

Fit: A call shown as `RAN mcp_tools mcp_tools {"query":…}` shows `RAN` and `mcp_tools` once, then the arguments. The same for `mcp__Operations__project_status`.
