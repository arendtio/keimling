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

### REQ-VE-12 A narrow window uses one side margin

On a narrow window the page shall meet both screen edges. The transcript and the composer shall share one margin, the same on the left and the right. A user's bubble, a heading, and Keimling's own text all sit inside that margin.

Origin: User, 2026-11-21 (paraphrase: Keimling's own text sticks to the left edge). Same day (paraphrase: the composer has unequal side margins and something gray shows on the right; the chat does not reach the right edge while it does reach the left; headings are still against the edge, and the user's own bubble touches the edge even though the words inside it do not).

Fit: On a narrow window the paper reaches both screen edges, with no strip of the page background on one side. The composer is the same distance from the left edge as from the right, and that distance matches the transcript. A user bubble, a chapter heading, and a line from Keimling each start and end inside that margin. On a wide window the paper keeps its gap beside the sidebar.

### REQ-VE-10 The settled transcript stays put

When a reply finishes, the transcript on screen shall keep the shape it already has. Lists and checkboxes shall keep their spacing. The page shall not draw that reply again in another arrangement, and it shall not open or close a row on its own after the reply is already visible.

Origin: User, 2026-11-19 (quote: "wenn er fertig ist, sieht man kurz die seite und kurz darauf verändern sich dann die abstände von text zu checkbox … und noch einen moment später scheint das automatische ein und ausklappen noch dazu zu kommen" and "immer vor eigenen nachrichten so eine trennlinie … Die Überschrift reicht vollkommen").

Fit: While a reply streams, a task line is already a checkbox with the same gap it has when the reply stops. After the run goes idle, that gap does not change. The run's Thinking rows and
tool cards close once, as REQ-VE-13 says, and a row the user opens after that
keeps its state. A chapter shows its heading and no line before the user's message. Inside one turn the rows share one gap. When the speaker changes, the same larger gap sits before the new voice, under a user message and under a Keimling message.

### REQ-VE-11 Stay with the latest lines

When the page is within 200 pixels of the bottom, growth in the transcript shall move it to the bottom. Growth includes streaming text, a finished message, and a tool result, whether the line is from the user or from the agent. A few pixels above the bottom still count as the bottom. Farther up, growth shall leave the page where it is. Sending one's own message shall move to the bottom even from farther up. Opening another thread shall land at the bottom. Refreshing the current session after the agent finishes shall not jump to the top; readers farther up keep their position. The last line shall sit clear of the composer.

Origin: User, 2026-11-19 (quote: "wenn er neue Nachrichten … dranhängt … vom User kommen oder vom Agenten, dann scrollt er nicht automatisch weiter runter" and "zum Bereich von 200 Pixeln oder so ist völlig in Ordnung, dass er in dem immer nach unten snappt … wenn neue Nachrichten reinkommen"). Same day, the answers were: follow on every growth, including streaming; sending one's own message always jumps. This replaces the earlier same-day wish that a reply stay put after one scroll. The composer clearance from that wish stays (quote: "etwas mehr abstand zwischen chatbox und der untersten Zeile des chatverlaufs"). Follow-up on macOS: "wenn das Gespräch wieder endet … manchmal wieder nach ganz oben".

Fit: With the page 80 pixels above the bottom, a streaming token, a tool card, and a finished line each leave the page at the bottom. With the page 400 pixels above the bottom, those arrivals leave the scroll position unchanged. Sending a message from 400 pixels up lands on the bottom, and the reply that follows stays at the bottom. The gap under the last line stays clear of the composer.

The moment the distance is measured, and a reader who is scrolling away while a run is still going, are REQ-VE-15. The 200-pixel band itself stays.

### REQ-VE-9 Follow the live thinking row

The More menu shall offer Follow thinking, off until the user turns it on. While it is on, the thinking row that is receiving text stays open, and that row closes when the text stops. The next thinking row opens on its own. A thinking row the user opened stays open. Turning Follow thinking off leaves rows closed until the user opens one. The choice is still there on the next visit.

Origin: User, 2026-11-19 (quote: "Das thinking feature bitte auch umsetzen"), after the agreed shape: a control in the chat More menu, off by default, that follows only the row currently receiving tokens and closes it when that row finishes, while a row opened by hand stays open.

Fit: With Follow thinking off, a thinking row stays closed while its text arrives. Turning it on opens the row that is receiving text. When the reply or a tool call starts, that row closes. The next thinking row opens. A row the user opened before that stays open. Turning the control off closes nothing the user opened, and the next thinking row stays closed. Reloading the page keeps the control as the user left it.

### REQ-VE-13 Idle closes the run's detail rows

When a run goes idle, the Thinking rows and tool cards of that run shall be
closed. The lines the agent wrote to the user, and the user's own messages,
keep the shape they have. A row the user opened before the run started keeps
its state, and a chapter fold is not a run row. Closing happens once: after
that the user opens and closes single rows by hand, and a row that was closed
stays closed.

Origin: User, 2026-11-21 (paraphrase: at the end of a run only the messages the
assistant wrote to the user should stay open; Thinking blocks and tool cards
collapse).

Fit: After a run with a tool card and a Thinking row opened by Follow thinking,
the idle transcript shows both closed and the reply unchanged. A tool card the
user opens then stays open across the session refresh that follows idle.
Reloading the thread still shows every run row closed.

### REQ-VE-5 Tool name once

The closed tool line shows the tool name once. Arguments follow that name.
The name may be bold.

Origin: User, 2026-11-12 (quote: "Ich habe das gefühl da steht die der Methodenname immer zweimal. aber einmal würde reichen. Fett gedruck ist ok.").

Fit: A call shown as `RAN mcp_tools mcp_tools {"query":…}` shows `RAN` and `mcp_tools` once, then the arguments. The same for `mcp__Operations__project_status`.

### REQ-VE-14 A call under review says so, and its review time stays visible

While a call waits for the automatic approval review, its tool line shall say
`Reviewing` instead of looking like a running call. When the gate passes, the
line shall say `Running`, and when the gate asks a human it shall say
`Awaiting approval`. After the gate decides, the closed line shall show how long
the review took (`checked 2.3s`), so review time and run time can be compared.
This is progress, not transcript: after a reload the line shows the state the
stored events carry, without the measured time.

Origin: User, 2026-11-23 (quote: "dass man dann an der Oberfläche gar nicht sieht, wie viel Zeit eigentlich mit der Prüfung vergeht … dass man einmal sehen kann, okay, jetzt läuft gerade die Prüfung durch und okay, jetzt läuft der Befehl eigentlich erst los").

Rationale: In auto mode the review is a model call, and its latency was spent
in silence. A call that looks running while it only waits for a reviewer hides
where the time goes.

Fit: In auto mode an `exec` that goes to the reviewer shows `REVIEWING exec …`
on the card while the review runs, then `RUNNING`, and after the result the
closed line shows `checked` with the review duration; the run itself is not
included. A call refused by the reviewer shows `FAILED` with the same duration.
A call that asks the human reads `AWAITING APPROVAL` until it is answered. A
call that never went to the reviewer shows no `checked` note.

### REQ-VE-15 Follow is decided before the new lines arrive

When new transcript content is about to be added, the page shall remember whether it is inside the follow band of REQ-VE-11. If it is, the page shall be at the bottom after those lines are drawn, including when one new message is taller than that band. If it is not, the page shall stay where it was. A scroll the reader makes upward shall keep the page from following for a short pause on the order of one to two seconds. After that pause the page shall follow only when it is inside the band again.

Origin: User, 2026-11-25 (quote: "bevor die Nachricht reinkommt, er guckt, ob er in dem Bereich ist, an dem er Snack to Button machen sollte. Und wenn ja, dann muss er nach links die Nachricht gerendert wurde, Snack to Button machen" and "wenn ich gerade ein Hochscroll Event gemacht hab … für 'ne Sekunde oder zwei oder so erstmal hochscrollt und kein Snack to Button macht").

Rationale: A tall message moves the old bottom farther than 200 pixels away, so a check after the draw treats a reader who was at the bottom as someone who scrolled away. A follow that fights an upward scroll makes Follow thinking unreadable.

Fit: The page is at the bottom and one arriving message is taller than 200 pixels; after it is drawn the page is at the bottom. The page is 400 pixels above the bottom before that message; after the draw it is still about 400 pixels above the new bottom. During a run the reader scrolls upward; for the next one to two seconds new tokens leave the page where the reader put it. After that pause, a page still inside the band follows again, and a page outside the band stays.

### REQ-VE-16 The agent's own sentences stay visible when a run ends

When a run goes idle, Thinking rows and tool cards of that run close, as REQ-VE-13 says. A sentence the agent wrote that is neither Thinking nor a tool card shall stay visible, including a sentence that sits between tool cards and the sentence that ends the run. A user message shall stay visible, as REQ-VE-8 says. Closing shall not put those sentences into a folded stretch.

Origin: User, 2026-11-25 (quote: "wenn er quasi fertig ist mit seinem Lauf und in Idle zurückgeht, dann sollten immer alle Tool-Aufrufe und jedes Thinking … ausgeblendet werden, aber die Nachrichten, die der Bot zwischendurch geschrieben hat, die kein Tool-Aufruf und kein Thinking sind, die sollten angezeigt bleiben, genauso wie User-Nachrichten" and "die Nachricht, die der Bot mir am Ende schickt … wird manchmal beim Ein- und Ausblenden wieder ausgeblendet").

Rationale: The result is the thing the person came to read. Folding it with the tool trace hides the answer at the moment the run finishes.

Fit: A run that writes a sentence, then a tool card, then a closing sentence goes idle with both sentences readable and the tool card and any Thinking row closed. A user message in that stretch stays readable. Opening the tool card afterward shows its input and output. Reloading the thread still shows the sentences and still shows the tool card closed.
