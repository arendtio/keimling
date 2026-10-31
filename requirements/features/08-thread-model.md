# Feature: Thread model

The chat model for one thread. Settings stay the global default. These records are the stakeholder ask. Control shape, storage, and API shape are not decided here.

A configured choice is a provider plus one of that provider's saved models. The upstream catalog is not a source of choices.

Precedence for the **main chat model** only: a saved thread choice, otherwise the current Settings default (active provider and its active model), live. Compaction, the autopilot supervisor, delegates, and the review model stay on their Settings models.

Selecting the provider and model that Settings currently use clears the thread choice. The thread then follows Settings again, including a later Settings change. Selecting any other configured pair saves it, and the thread keeps that pair when Settings later change.

Reasoning and context limit come from the model that is in effect for the chat. A model with no context limit still uses the global context-limit fallback from [feature 06](06-providers-and-models.md).

A run already started keeps the chat model it started with. The thread choice is used when a message is sent.

## REQ-TM-1 Choose the model for this thread

- **Bin:** explicit requirement
- **Statement:** While in a thread, the product shall let the user choose which configured model, on which configured provider, that thread's main chat model is.
- **Origin:** User, 2026-10-31 (quote: "dass man im Chat auswählen kann, mit welchem Modell der Chat, also der Thread eigentlich laufen soll" and "welches Modell bei welchem Provider" and "dass das dann auch wirklich nur für den Thread ausgewählt ist"). Scope, same day (quote: "wenn ich im Thread etwas auswähle, dann gilt das nur für das Chat-Modell. Die anderen Modelle sind darin unberührt.").
- **Rationale:** Settings hold several providers and, on each, several models. Different threads need different chat models. Other model roles stay global.
- **Fit criterion:** Thread A is set to provider P1 model M1. Thread B has no thread choice, and Settings use P2 model M2. The next message sent in A is answered by P1 as M1, with M1's reasoning and M1's context limit. Compaction, autopilot, delegates, and review for A still use their Settings models. The next message sent in B is answered by P2 as M2. Reopening A still shows M1 on P1. A model id that is not saved on that provider cannot be chosen.
- **Priority:** shall
- **Status:** confirmed

## REQ-TM-2 Settings stay the live default

- **Bin:** explicit requirement
- **Statement:** A thread with no saved model choice shall use the current Settings default for its chat model, including after that default changes.
- **Origin:** User, 2026-10-31 (quote: "Das, was ich in den Settings mache, ist ja quasi die globale Voreinstellung" and "ein Thread ohne eigene Wahl, der orientiert sich immer an der Einstellung der Systemsettings.").
- **Rationale:** Threads that were not given their own model stay on whatever Settings say now.
- **Fit criterion:** Thread B has no saved choice. Settings use M2. B's chat model is M2. Settings then change to M3. The next message sent in B is answered as M3. Changing A's saved choice does not change the Settings default.
- **Priority:** shall
- **Status:** confirmed

## REQ-TM-3 A thread choice stays until the user picks the current default

- **Bin:** explicit requirement
- **Statement:** A thread with a saved model choice shall keep that choice when Settings change. Selecting the provider and model that Settings currently use shall clear the thread choice, so the thread follows Settings again.
- **Origin:** User, 2026-10-31 (quote: "Wenn ein Thread ein eigenes Modell hat, dann bleibt er dabei." and "in dem Moment, wo ich auf den Wert, den ich in den System Settings aktuell habe, soll es quasi ein automatisches Resetten sein, dass er wieder den System Settings Wert nutzt.").
- **Rationale:** There is no separate reset control. Picking the current default is how the thread lets go.
- **Fit criterion:** Thread A has saved P1/M1. Settings change from P2/M2 to P3/M3. A's chat model stays P1/M1. The user then selects P3/M3 in A. A no longer has a saved choice. Settings then change to P4/M4. The next message sent in A is answered as P4/M4. Selecting P1/M1 while Settings are some other pair saves P1/M1 again.
- **Priority:** shall
- **Status:** confirmed

## REQ-TM-4 Switch applies on the next send

- **Bin:** explicit requirement
- **Statement:** The product shall let the user switch the thread's chat model during the thread. A run already started shall keep the model it started with. The new choice shall be used when a message is sent.
- **Origin:** User, 2026-10-31 (quote: "innerhalb eines Threads in der Lage sein, das Modell zu wechseln" and "ein Lauf, der schon gestartet ist, wird nicht beeinflusst, sondern erst mit dem Absenden einer Nachricht wird dann das neue Modell eingesetzt").
- **Rationale:** The model is not fixed for the life of the thread, and an in-flight run is not retargeted.
- **Fit criterion:** A run that started on M1 finishes on M1 after the user selects M2 during that run. The next message the user sends is answered as M2. Earlier messages stay in the thread. Reopening the thread shows M2.
- **Priority:** shall
- **Status:** confirmed

## REQ-TM-5 Reasoning and context come from the chat model

- **Bin:** explicit requirement
- **Statement:** The chat call shall use the reasoning and context limit stored on the model that is in effect for that thread.
- **Origin:** User, 2026-10-31 (quote: "Reasoning und Context Limit sind die Werte am gewählten Modell.").
- **Rationale:** Those values belong to the model, not to a second global override.
- **Fit criterion:** M1 has reasoning R1 and context limit C1. M2 has R2 and C2. A thread whose chat model is M1 sends the chat call with R1, and context handling for that chat uses C1. After the thread's chat model is M2, the next sent message uses R2 and C2. When the model in effect has no context limit, the global context-limit fallback from feature 06 still applies.
- **Priority:** shall
- **Status:** confirmed

## Explicit bounds

The choice is among models already stored on a provider in Settings. The user described that catalog as what they want to pick from (quote: "in den Einstellungen verschiedene Provider anlegen" and "bei den verschiedenen Providern kann man verschiedene Modelle hinterlegen").

## Design notes (not requirements)

The composer already shows the current model next to the approval mode, with the title "Model for this thread (change in Settings → Models)". That label describes the outcome. The user did not require that control. They also said they do not want a separate reset control; picking the current Settings pair is the reset.
