# Feature: Composer and workspaces

The records below are the stakeholder ask for a less crowded surface. Which control is drawn as an icon, and the geometry of those icons, are not decided here.

## REQ-CW-1 The composer keeps every action, without a row of them

- **Bin:** explicit requirement
- **Statement:** The composer shall not show every compose action as its own always-visible control. Sending stays immediate. The microphone and a call stay visible with Send. Adding a file and sharing a screen stay reachable in one step. Stopping a run stays available while a run can be stopped. None of these actions are removed.
- **Origin:** User, 2026-11-08 (paraphrase: when nothing is running there are about six buttons beside the composer, which is too many; every function stays important and must not be removed or buried; secondary functions can sit behind one extra button so the composer is not overloaded). Same day (paraphrase: the microphone and call icons belong above Send, where there is still room).
- **Rationale:** The actions are all used, and a column of them crowds the place where the user writes.
- **Fit criterion:** With the agent idle, the composer shows Send and one control that opens File and Screen, each named. The plus stays under the field. Record, call, and Send stack to the right of the field. While a run or a spoken reply can be stopped, Stop is in that stack. During a call, mute and hang-up stay in that stack, and hang-up is its own icon. A user who has not opened the extra control can still send.
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
- **Statement:** The composer field shall grow with the draft until about seven lines, and shall scroll after that. At that cap, the last visible line shall be cut so it is clear the draft continues.
- **Origin:** User, 2026-11-08 (paraphrase: a long draft only shows about two lines; the field should grow until about seven lines and then scroll inside the field; the height should not land on a whole number of lines, so about seven and a half lines stay visible and the scroll is obvious).
- **Rationale:** A two-line box hides the draft the user is still writing. A clean cut on a whole line hides that there is more above or below.
- **Fit criterion:** One line of text is one line tall. Seven lines are fully visible without a scrollbar. Text beyond that scrolls inside the field, and the field stays about seven and a half lines tall.
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
