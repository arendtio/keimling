# Feature: Autopilot instructions

Standing text the autopilot supervisor follows. The factory text lives in `src/autopilot.go`. These records are the stakeholder ask. Storage and API shape are not decided here.

Three levels. The text that is in effect is the **entire** autopilot prompt. No fragment of a less specific text remains, including the factory JSON contract and the factory ban on authorizing system changes.

| Level | What the user called it | When it is in effect |
| --- | --- | --- |
| Factory | in der Software hinterlegt | No saved global override |
| Global | Settings, globaler Prompt, System-Prompt | Saved in Settings, and the thread has no saved text of its own |
| Thread | Thread-Prompt | Saved on that thread, until reset |

A thread with no saved text follows the global level live: a later change in Settings applies to it. Saving a thread text detaches that thread. It keeps that text until the user resets it. Reset clears the saved thread text so the thread follows the global level again.

## REQ-AP-1 Global autopilot instructions

- **Bin:** explicit requirement
- **Statement:** The product shall let the user set the autopilot's standing instructions in Settings. A saved global text shall be the entire autopilot prompt, replacing the factory text, for every thread that has no saved thread text.
- **Origin:** User, 2026-10-30 (quote: "in den Settings die Anweisungen an den Autopiloten anpassen kann. Also global."). Composition, same day (quote: "in den Settings, der Text überschreibt den in der Software hinterlegten Text" and "die Texte immer den kompletten Autopiloten Prompt ersetzen"). Live follow, same day (quote: "wenn kein Thread-spezifischer Prompt gesetzt ist, dann nutzt er eben den System-Prompt und wenn der geändert wird, gilt dann der auch.").
- **Rationale:** The factory ban on authorizing system changes is right in general, and wrong for a project whose task is to change a second machine over SSH.
- **Fit criterion:** A distinctive sentence saved as the global autopilot instructions is still that text after the process restarts. Supervision of a thread with no saved thread text sends exactly that sentence and does not contain the factory text. After the user changes the saved global text to a second sentence, the same thread's next supervision sends exactly the second sentence.
- **Priority:** shall
- **Status:** confirmed

## REQ-AP-2 Per-thread autopilot instructions

- **Bin:** explicit requirement
- **Statement:** The product shall let the user save autopilot instructions for one thread. A saved thread text shall be the entire autopilot prompt for that thread, replacing the global text and the factory text, and the thread shall keep that text when the global text later changes.
- **Origin:** User, 2026-10-30 (quote: "pro Thread … abweichende Anweisungen reinschreibt" and "der Text, den man im Thread festlegt, der ersetzt den Supervisor Prompt"). Detach, same day (quote: "jeder Thread behält den Prompt, den er hat. Also wenn er quasi sich einmal gelöst hat vom globalen Prompt, dann hat er seinen eigenen.").
- **Rationale:** One thread may need system changes on a remote machine. Other threads keep following the global text. The thread text is the most specific.
- **Fit criterion:** Thread A has saved text T-a. Thread B has no saved text. The global text is then changed to G2. Supervision of A sends exactly T-a and does not contain G2 or the factory text. Supervision of B sends exactly G2. Reopening thread A shows T-a. Saving the text that was shown, including a text that still equals the current global text, counts as setting a thread prompt.
- **Priority:** shall
- **Status:** confirmed

## REQ-AP-3 Unset thread text shows the global text

- **Bin:** explicit requirement
- **Statement:** While a thread has no saved autopilot text, the product shall show the effective global autopilot text as that thread's autopilot text.
- **Origin:** User, 2026-10-30 (quote: "Standardmäßig sollte im Thread Text aber genau der gleiche Text stehen, der im globalen Autopiloten steht, also in den Settings festgelegt ist.").
- **Rationale:** The user edits the stance that is already in effect, and does not start from a blank field.
- **Fit criterion:** With global text G saved and no saved thread text, opening the thread's autopilot instructions shows G. After a settings reset to the factory text (REQ-AP-8), the same opening shows the factory text.
- **Priority:** shall
- **Status:** confirmed

## REQ-AP-4 Empty thread text cannot be saved

- **Bin:** explicit constraint
- **Statement:** The product shall reject saving a thread autopilot text that is empty. The product shall accept a non-empty text, including a single word.
- **Origin:** User, 2026-10-30 (quote: "wenn im Thread ein leerer Text gespeichert wird, das finde ich komisch … Das sollte man so eigentlich nicht speichern können." and "Wenn da nur ein Wort drin steht oder so, dann ist das halt so … Anwenderfehler.").
- **Rationale:** An empty save would wipe the stance. Clearing the thread text is a separate reset (REQ-AP-6). A short text is the user's choice.
- **Fit criterion:** Saving "" or a whitespace-only string is refused, the thread stays in its previous state (still following global, or still on its previously saved text), and that previous text is still what supervision uses. Saving "x" succeeds, detaches the thread, and supervision uses exactly "x".
- **Priority:** shall
- **Status:** confirmed

## REQ-AP-5 Approval mode stays

- **Bin:** explicit constraint
- **Statement:** The worker approval mode (`ask` / `auto` / `yolo`) shall stay as it is. Autopilot instruction text shall not change it.
- **Origin:** User, 2026-10-30 (quote: "der Freigabemodus ist auch, wie du schreibst, bleibt, wie er ist.").
- **Rationale:** Editable instructions change what the supervisor steers toward. They do not change who may approve a tool call.
- **Fit criterion:** Saving or resetting a global or thread autopilot text leaves the stored approval mode unchanged. A tool call that required approval before still requires the same approval after.
- **Priority:** shall
- **Status:** confirmed

## REQ-AP-6 Reset a thread back onto the global prompt

- **Bin:** explicit requirement
- **Statement:** The product shall offer a button that clears the saved thread autopilot text, so the thread follows the global prompt again.
- **Origin:** User, 2026-10-30 (quote: "es soll auch 'n Button geben, mit dem man den Thread-spezifischen Prompt wieder resetten kann, sodass man sich quasi wieder an den System-Prompt hängt" and "man muss irgendwie diesen Thread-spezifischen Prompt wieder löschen oder zurücksetzen … können.").
- **Rationale:** Empty save is refused, so the user needs another way to drop a thread text and pick up later global edits again.
- **Fit criterion:** Thread A has saved text T-a. The user activates the reset button. Thread A no longer has a saved text. Supervision of A sends exactly the current global text. After the global text is then changed, supervision of A sends exactly the new global text. The control is a button.
- **Priority:** shall
- **Status:** confirmed

## REQ-AP-7 Empty global text cannot be saved

- **Bin:** explicit constraint
- **Statement:** The product shall reject saving an empty global autopilot text in Settings.
- **Origin:** User, 2026-10-30 (quote: "in den Settings, auch da soll 'n leerer Text abgelehnt werden").
- **Rationale:** Same failure as an empty thread save. Returning to the factory text is a separate reset (REQ-AP-8).
- **Fit criterion:** Saving "" or a whitespace-only string in Settings is refused. The previous effective global text stays in effect for threads that have no saved thread text.
- **Priority:** shall
- **Status:** confirmed

## REQ-AP-8 Reset Settings back to the factory prompt

- **Bin:** explicit requirement
- **Statement:** The product shall let the user reset the global autopilot text so that the factory prompt is in effect again.
- **Origin:** User, 2026-10-30 (quote: "auch da muss man die Möglichkeit haben, den wieder zurückzusetzen auf den in der Software hinterlegten Prompt.").
- **Rationale:** Empty save is refused, so the user needs another way to drop a global override.
- **Fit criterion:** After the reset, no saved global override remains. Threads with no saved thread text are supervised with exactly the factory text. A thread that still has a saved thread text keeps that text. If the factory text in the software later differs, a thread with no saved thread text and no saved global override uses the new factory text.
- **Priority:** shall
- **Status:** confirmed. The user required the capability, not a specific control. REQ-AP-6 names a button; this one does not.

## Explicit scenario (rationale, not a separate capability)

The user described one project where the worker must change a second machine over SSH, so the factory line about not authorizing system changes does not fit. The asked capability is a full replacement of the autopilot prompt at global and thread level.

## Design notes (not requirements)

Preferred place for the per-thread text, hedged with "glaube ich": the top-right **More** menu that already contains Context and Artifacts. Global text: **Settings**, next to the existing Autopilot on/off control (REQ-AP-1).

Reset throws away a text the user typed. The thread reset returns to the global text; the settings reset returns to the factory text. A confirm or an undo belongs on those controls as a design default, not as something the user asked for.
