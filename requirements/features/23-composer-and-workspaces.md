# Feature: Composer and workspaces

The records below are the stakeholder ask for a less crowded surface. Which control is drawn as an icon, and the geometry of those icons, are not decided here.

## REQ-CW-1 The composer keeps every action, without a row of them

- **Bin:** explicit requirement
- **Statement:** The composer shall not show every compose action as its own always-visible control. Sending stays immediate. The microphone and a call stay visible with Send. Adding a file and sharing a screen stay reachable in one step. Stopping a run stays available while a run can be stopped. None of these actions are removed. Every control that is visible without opening More is an icon with no word on it. Controls inside More may show a word.
- **Origin:** User, 2026-11-08 (paraphrase: when nothing is running there are about six buttons beside the composer, which is too many; every function stays important and must not be removed or buried; secondary functions can sit behind one extra button so the composer is not overloaded). Same day (paraphrase: the microphone and call icons belong above Send, where there is still room). User, 2026-11-11 (quote: „alle Buttons in der Checkbox, die sofort sichtbar sind, sollten keinen Text haben … die, die man über dieses Plus, über den More-Button erreicht, das ist okay … keiner davon soll 'n Text haben“, examples named „Q“ and „Done“).
- **Rationale:** The actions are all used, and a column of them crowds the place where the user writes. A word on those icons, such as Queue or Done, makes the column wider and uneven.
- **Fit criterion:** With the agent idle, the composer shows Send and one control that opens File and Screen, each named. The plus stays under the field. Record, call, and Send stack to the right of the field. While a run or a spoken reply can be stopped, Stop is in that stack. During a call, mute and hang-up stay in that stack, and hang-up is its own icon. While recording, while a message will wait, and while a run is stopping, those same controls stay icons: Queue, Done, and Stopping are the accessible name and the hover text, not a word drawn on the button. A user who has not opened the extra control can still send.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-2 Icons stay small

- **Bin:** explicit constraint
- **Statement:** Where a control is drawn as an icon, the product shall embed only the vector graphics that control uses. It shall not embed an icon font or a collection of unused icons.
- **Origin:** User, 2026-11-08 (paraphrase: text on buttons could be cleaner as icons in places; the binary must not grow because a large icon set is embedded; prefer vectors, and only the ones actually used).
- **Rationale:** Every file under the embedded UI is part of the binary.
- **Fit criterion:** The composer icons are vector graphics for those controls only. The UI assets do not include an icon font or a general icon library.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-3 A workspace is known by a chosen name

- **Bin:** explicit requirement
- **Statement:** The workspace switcher shall show the workspace's name and shall not show its folder path under that name. The name shall be one the user can set, so two folders with the same folder name can be told apart. The user shall be able to add a workspace by choosing a folder and a name, rename it, and remove it from the list. Removing it shall take it off the list and shall leave the directory and its files in place.
- **Origin:** User, 2026-11-08 (paraphrase: the path under the name should not be visible; the name is enough, but today it is the folder name, so two folders with the same name cannot be told apart; a Manage control on the switcher, instead of the open-folder field, can open a dialog to add a name and a folder, edit, and remove from the list without deleting the directory).
- **Rationale:** The switcher is for choosing a workspace. The folder path is how it is stored, not how the user wants to read it. The name has to be choosable or identical folder names stay indistinguishable.
- **Fit criterion:** The closed switcher shows a name and no path beneath it. Adding a workspace stores the name the user typed and the folder they chose. Renaming changes the name the switcher shows and does not move the folder. In the manage list, Edit is the last control on the row. Removing a workspace hides it from the switcher; the directory still exists. Adding that folder again puts the workspace back on the list.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-4 No working-state block above the chat

- **Bin:** explicit requirement
- **Statement:** The product shall not show a working-state block above the conversation. The context remains available in the side area.
- **Origin:** User, 2026-11-08 (paraphrase: the working state above the conversation is superfluous because the full context can already be viewed).
- **Rationale:** The same facts are already in the context view, and the block takes a place above every thread.
- **Fit criterion:** A session that has a working state does not show a Working state disclosure above the thread. The context view still shows the context.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-5 The composer field grows with the draft

- **Bin:** explicit requirement
- **Statement:** The composer shall grow with the draft and then scroll inside the field. On a narrow window the whole composer may reach about 70 percent of the screen height. On a wide window it may reach about 50 percent. Past that cap, the field scrolls and the last visible line is cut so it is clear the draft continues.
- **Origin:** User, 2026-11-08 (paraphrase: a long draft only shows about two lines; the field should grow and then scroll inside the field; the height should not land on a whole number of lines, so the scroll is obvious). User, 2026-11-21 (paraphrase: the transcription field is only about three and a half lines and should be able to grow much taller; the whole composer may reach about 70 percent of the screen height on a phone and about 50 percent on the desktop).
- **Rationale:** A short box hides the draft the user is still writing. A phone can give that draft more of the screen than a desktop, where the conversation above should stay in view. A clean cut on a whole line hides that there is more above or below.
- **Fit criterion:** One line of text is one line tall. On a narrow window the composer, including its controls, stays within about 70 percent of the screen height. On a wide window it stays within about 50 percent. Text beyond that scrolls inside the field, and the field does not end on a whole line. The dictation transcript uses the same cap.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-6 The draft starts under the model line

- **Bin:** explicit requirement
- **Statement:** The composer text shall start at the top of the composer, directly under the model line. A taller stack of controls shall not drop that text lower. The plus stays on the bottom edge of the composer. Typing shall still grow the field downward.
- **Origin:** User, 2026-11-08 (paraphrase: the message sits in the middle of the composer, especially when a live call shows five icons on the right; it should always begin at the top, just after the model, and grow downward while typing. The plus should stay on the bottom edge of the box).
- **Rationale:** The model line is the start of the composer. A short draft that sits halfway down looks like empty space above the words.
- **Fit criterion:** With one line of text and several controls beside it, the text begins directly under the model line and the plus sits on the bottom edge of the composer. Adding lines moves the bottom of the field down. The first line stays under the model line.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-7 Recover and retry a voice recording

- **Bin:** explicit requirement
- **Statement:** The composer More menu shall offer the latest microphone recording for download and accept an audio file for transcription. The recording is kept in the current UI's memory, without requiring persistence across restarts. Transcribing an uploaded file appends the result to the current draft without sending it.
- **Origin:** User, 2026-11-20 (paraphrase: inaccurate or failed transcriptions can lose a long dictation, including when the message was sent immediately; downloading the last recording and uploading audio for transcription is more useful than retry alone, and the recording needs to exist only during the current run).
- **Rationale:** A downloaded copy allows another transcription attempt even after the application is closed. The user can remove a mistaken draft manually.
- **Fit criterion:** After stopping a recording, including one whose transcription fails or was sent, More can download it. A subsequent recording replaces it. Selecting an audio file through More transcribes it and appends the text to the draft, leaving Send to the user. Uploading a file does not replace the latest microphone recording. The retained recording is not restored on a fresh UI load.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-8 A narrow window swipes between writing, dictation, and a call

- **Bin:** explicit requirement
- **Statement:** On a narrow window, the user shall move between writing, dictation, and a live call by dragging the composer sideways. The writing page shall be at least as tall as the dictation and call pages, so the move does not change the composer height. The current page shall be marked without naming the pages in the chrome. The model line shall stay inside the composer. Writing shall keep More at the bottom left and Send along the bottom edge. Dictation shall use the cassette control, show mute only while a recording is open, and offer Cancel and Send as labeled controls. Cancel shall drop the dictation draft and an open recording, and shall leave the typed message on the writing page. A call shall use the phone control to start and to hang up, show screen share and mute beside it only while the call is open, and keep Interrupt across the full bottom edge, inactive until it can stop something. Dragging the composer shall not send or discard. A wide window shall keep the icon stack from REQ-CW-1. While a run can be stopped, Interrupt shall also be on the writing page.
- **Origin:** User, 2026-11-21 (paraphrase of the mobile composer they approved: three dots, swipe on the input box, model line inside the opaque box, cassette in the center, mute only after recording starts, Cancel and Send as matching text buttons, Plus at the bottom left, Send on the bottom edge, Stop full width and grayed until it can act, larger targets on the narrow view, small icons remaining on the desktop). Same day (quote: „Bitte integriere das in unsere bestehende App“). Same day (paraphrase: the writing box should use the height of the call and transcription pages as its minimum, so it does not jump when swiping).
- **Rationale:** A thumb cannot use the desktop icon stack. The actions stay the ones the composer already has. The typed draft and the dictation draft are different pieces of work, so discarding a take must not wipe the message.
- **Fit criterion:** At a wide width the composer still matches REQ-CW-1. At a narrow width, dots sit above the composer with no fill of their own, and a horizontal drag on the composer changes page. With no extra draft, the writing page is at least as tall as the dictation page and the call page, and moving between the three leaves the composer height unchanged. A longer draft may still grow the page that holds it. The composer stays along the bottom of the screen, including when the thread is empty, and it is the same distance from the left edge, the right edge, and the bottom edge. Send sits on the bottom edge of the writing page, with More on its left. The writing field spans the width of the composer. Dictation shows the cassette, mute only during recording, and labeled Cancel and Send. Cancel clears the transcript and drops the open recording; text already in the writing field remains. The call page starts and hangs up from the center control. Share and mute appear only while the call is open. Interrupt fills the bottom of the call page and is inactive until a run or the spoken reply can be stopped. The same Interrupt is on the writing page while a run can be stopped. A drag does not send and does not clear either draft. Screen share still starts from the call page and from More. No camera control is offered.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-11 The page dots stay readable over the chat

- **Bin:** explicit requirement
- **Statement:** On a narrow window the strip behind the three composer page dots shall be transparent, so the chat shows through it. Each dot shall have a ring in the current page background color, so the dot stays readable when text scrolls behind it.
- **Origin:** User, 2026-11-25 (quote: "der Hintergrund hinter den drei Punkten, der ist im Moment in unserer weißen Farbe und ich hätte gern, dass der transparent wird" and "die drei Punkte jeweils außenrum quasi in der Farbe, die aktuelle Hintergrundfarbe ist, ein weiße eine weiße Umrandung haben" and "wenn man irgendwie scrollt und da Text hinter ist, man immer noch sehr klar sehen kann diese grünen Punkte" and "nicht der ganze horizontale Balken 'n weißer Hintergrund sein").
- **Rationale:** A solid bar covers the last lines of the chat. A bare dot disappears into the same text.
- **Fit criterion:** On a narrow window, with text scrolled under the dots, the page background is not a solid bar behind them, and the current dot stays distinguishable from that text. The ring uses the page background of the active theme. Changing page still marks the current dot.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-9 More opens over the chat

- **Bin:** explicit requirement
- **Statement:** On a narrow window, More shall open over the composer and the chat, anchored to the Plus control, so every entry can be reached. The composer shall not clip it.
- **Origin:** User, 2026-11-21 (paraphrase: the Plus menu opens in the wrong place and is cut off at the edge of the chat box, so the entries cannot be selected).
- **Rationale:** More is the way to attach a file, share a screen, and recover a recording. A menu that ends at the composer box hides those actions.
- **Fit criterion:** With the writing page at the bottom of a narrow window, opening More shows File, Screen, Download last recording, Transcribe audio file, Reduce context, and Follow thinking, each clickable. The menu sits by the Plus control and paints over the chat. On a wide window, More still opens from the plus under the field.
- **Priority:** shall
- **Status:** confirmed

## REQ-CW-10 A draft stays with the chat it was typed in

- **Bin:** explicit requirement
- **Statement:** The composer shall keep a draft with the chat it was typed in. An agent chat and a workspace thread shall not show each other draft. Opening a chat again shall show the draft that was left there, and sending shall clear it. The draft shall live in the client only and shall not survive a reload.
- **Origin:** User, 2026-11-22 (paraphrase: a draft typed in an agent chat appears in a workspace thread and the other way round, and it can be lost when switching).
- **Rationale:** An agent chat is an instance, not a thread, so a draft keyed by thread alone lands in the wrong chat. A draft is unfinished writing and belongs where it was written.
- **Fit criterion:** Typing in an agent chat and then opening a workspace thread shows an empty field, and returning to the agent chat shows the typed text. The same holds from a thread to an agent chat. Two chats whose ids match keep separate drafts. Sending clears the draft for that chat. Reloading the page starts with an empty field.
- **Priority:** shall
- **Status:** confirmed
