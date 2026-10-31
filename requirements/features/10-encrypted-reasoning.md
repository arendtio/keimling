# Feature: Encrypted reasoning across a model change

The chat model for a thread is [feature 08](08-thread-model.md). These records are the stakeholder ask. How a request is filtered, and which control shows the notice, are not decided here.

## REQ-ER-1 Blobs stay with the model that produced them

- **Bin:** explicit requirement
- **Statement:** Encrypted reasoning shall stay in the thread's active context when the chat model changes. A later chat call shall include a blob when the chat model in effect is the provider and model that produced it, and shall omit it when the chat model is a different provider or model. A model change shall not itself remove the blob. If that span has already left the active context, switching back shall not put the blob into the call.
- **Origin:** User, 2026-10-31 (quote: "dass grundästzlich diese Reasoning blobs nicht verlorengehen, also wenn ich mit dem original model wieder arbeite sollten sie wieder verfügbar sein (sofern nicht zwischendurch ihr bereich eh rausgeflogen ist)" and "wenn an anderes Modell geshcickt wird, dann eben ohne die verschlüsselten Reasoning blobs").
- **Rationale:** The blob is only readable by the model that produced it. The user still wants it when they return to that model, including after working with another model in between.
- **Fit criterion:** The active context contains an encrypted reasoning blob produced while the chat model was provider P1 model M1. The user sets the chat model to P2/M2. The next chat call to P2/M2 does not contain that blob. The user sets the chat model back to P1/M1. The next chat call to P1/M1 contains that blob. A blob produced later under P2/M2 is in the call to P2/M2 and absent from the call to P1/M1. After compaction or another history reduction has dropped the span that held the P1/M1 blob, a chat call to P1/M1 does not contain it.
- **Priority:** shall
- **Status:** confirmed

## REQ-ER-2 Notice at the switch

- **Bin:** explicit requirement
- **Statement:** When the user changes the chat model and the active context contains encrypted reasoning that the newly chosen model will not receive, the product shall tell the user at that change. The notice shall still be visible when the user returns to the thread later, before sending a message. The product shall not wait until the next message is sent to say this.
- **Origin:** User, 2026-10-31 (quote: "Der Hinweis sollte beim model wechsel kommen, wenn solche blobs vorhanden sind, nicht erst beim abschicken an der neue model (dann erinnert sich der user vielleicht nicht mehr daran dass er das model vor der kaffeepause gewechselt hat)").
- **Rationale:** The user may switch, leave, and later not remember that this model is running without the earlier encrypted reasoning.
- **Fit criterion:** The active context contains an encrypted reasoning blob for P1/M1. The user selects P2/M2. Before any new message is sent, the thread shows that encrypted reasoning is not sent to this model. The user leaves and reopens the thread. The notice is still there, and no message has been sent yet. Selecting a model when the active context has no encrypted reasoning adds no such notice. Selecting P1/M1 again, so that the P1/M1 blob will be sent, adds no omission notice.
- **Priority:** shall
- **Status:** confirmed

## REQ-ER-3 Readable reasoning text

- **Bin:** explicit requirement
- **Statement:** The product should include readable reasoning text in a chat call when the active context has it.
- **Origin:** User, 2026-10-31 (paraphrase of "grundsätzlich ist es glaube ich auch sinnvoll immer den reasoning kontext mit zu schicken").
- **Rationale:** A later model can read the text. The encrypted blob is a separate item and follows REQ-ER-1.
- **Fit criterion:** The active context contains reasoning text from an earlier turn. The next chat call includes that text. This holds for the model that produced it and for a different model.
- **Priority:** should
- **Status:** confirmed

## Design notes (not requirements)

The chat call filters encrypted blobs for the model in effect and leaves them in the active context. `/continue-new-model` is removed; it checkpointed those blobs out, so switching back could not restore them.

The model menu closes after a choice. A notice that exists only there is gone when the user returns from a pause, so it would fail REQ-ER-2. A line that stays in the thread matches the fit criterion. The user did not require that control.
