# Feature: Chat branches

A chat can leave the linear transcript and continue from an earlier user message on another path. These records are the stakeholder ask. How a path is stored is not decided here.

## REQ-CB-1 A user message can start another continuation

- **Bin:** explicit requirement
- **Statement:** The product shall let the user replace the text of a message they already sent and send that replacement, so the conversation continues from that point as if the replacement had been the message. Only a message the user sent can start such a continuation. The continuation that was already there shall remain.
- **Origin:** User, 2026-11-08 (quote: "zu einer beliebigen Stelle weiter vorne gehen kann, eine Nachricht editieren kann und dann die editierte Nachricht abschicken" and "so zu tun, als würde ich an der Stelle der Chat-Historie wieder sein"). Same day (quote: "es entstehen nur aus Nutzernachrichten Alternativen").
- **Rationale:** A sent message is sometimes the wrong next step. Starting over from there should not throw away the try that already happened.
- **Fit criterion:** Given a thread whose user message U was followed by an assistant reply, the user sends a different text in place of U. The visible chat shows the new text and the reply to that new text. The earlier reply to U is not the visible continuation. A message the assistant wrote cannot be used to start a continuation.
- **Priority:** shall
- **Status:** confirmed

## REQ-CB-2 Every continuation of that message stays reachable

- **Bin:** explicit requirement
- **Statement:** At a user message that has more than one continuation, the product shall let the user move among all of them, including a third and further ones. Moving to one continuation shall show the chat that follows that continuation, and moving back shall show the other again.
- **Origin:** User, 2026-11-08 (quote: "beide Historien tatsächlich später noch durchnavigieren" and "nach rechts oder links navigieren" and "die Historie, die danach kommt, eben immer durchgetauscht"). Same day (quote: "es soll auch mehr als zwei Unterschiede geben können, also quasi beliebig viele").
- **Rationale:** The other try is only useful if it can be opened again later, and a message may be retried more than once.
- **Fit criterion:** A user message has three continuations, each with its own later messages. Choosing each one in turn shows that continuation's later messages and not the others. Choosing one that was already left shows the same later messages as before. The choice is still there after the app is opened again.
- **Priority:** shall
- **Status:** confirmed

## REQ-CB-3 Files stay as they are

- **Bin:** explicit constraint
- **Statement:** Switching or starting a continuation shall not restore, delete, or otherwise change files in the workspace.
- **Origin:** User, 2026-11-08 (quote: "nicht nur die Chat-Historie in einen Baum verwandeln, sondern dass man dann auch den Stand, den die Dateien zu dem Zeitpunkt hatten, wiederherstellen kann" and "Das wäre mir tatsächlich an dieser Stelle nicht so wichtig" and "primär nur um die Chat-Historie").
- **Rationale:** Rolling files back is a separate, heavier change. This feature is the conversation.
- **Fit criterion:** A continuation is started and another continuation is selected. The files in the workspace are the same as before those actions.
- **Priority:** shall
- **Status:** confirmed

## REQ-CB-4 The compacted memory matches that point in the chat

- **Bin:** explicit requirement
- **Statement:** When a continuation is started or selected, the compacted memory the model uses — goal, plan, findings, and the other fields of that memory — shall be the memory as it was when that user message was the point the conversation had reached. The next reply shall follow the visible continuation.
- **Origin:** User, 2026-11-08 (quote: "das, was wir an verdichteter Erinnerung haben, Zielplan, Feststellung und so, soll eigentlich den Stand bekommen, den es hatte, als die Nachricht, die man da ändert, was es zu dem Zeitpunkt war").
- **Rationale:** The model does not reread the whole chat. If the memory still describes a later branch, the new reply continues the wrong conversation.
- **Fit criterion:** After a user message U, the plan names a step that was not in the plan when U was sent. Starting another continuation at U and sending it, the plan the next reply works from does not include that later step. Selecting the original continuation shows that later step in the plan again.
- **Priority:** shall
- **Status:** confirmed

## REQ-CB-5 Continuations are kept without copying the past each time

- **Bin:** explicit constraint
- **Statement:** The product shall keep the continuations on disk. It shall not keep every continuation's memory in memory at once, and it shall not store a full copy of the earlier chat and the compacted memory beside every user message.
- **Origin:** User, 2026-11-08 (quote: "nicht zu jeder Nachricht, die der Nutzer schreibt, die gesamte Chat-Historie irgendwo an die Seite legen" and "auf der Festplatte persistiert, damit man das nicht alles immer irgendwie im Speicher mit sich rumträgt" and "Smartes, halbwegs speichersparendes, aber auch praktikables").
- **Rationale:** A copy of the whole past at every message grows without bound and is mostly repeated text.
- **Fit criterion:** A long thread is forked at an early user message. The stored size added by the fork is the new message and the links needed to rebuild that point, not a second copy of the transcript above it. After a restart only the visible continuation is loaded for the model, and selecting another continuation still shows it.
- **Priority:** shall
- **Status:** confirmed
