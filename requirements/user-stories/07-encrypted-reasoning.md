# User stories: Encrypted reasoning across a model change

Illustrates [REQ-ER-1](../features/10-encrypted-reasoning.md) and [REQ-ER-2](../features/10-encrypted-reasoning.md).

## US-34 — Leave and come back to the same model

**As** a user who changed the chat model, **I want** the encrypted reasoning of the original model to be used again when I work with that model, **so that** the switch did not throw it away.

**Acceptance**

- After the chat model changes, a call to the new model omits encrypted reasoning produced by the previous model.
- Switching back to the original provider and model includes that reasoning in the next chat call.
- Encrypted reasoning produced by the other model is included only while that model is the chat model.
- If the span that held the blob has already left the active context, switching back does not put it into the call.

## US-35 — Hear about it at the switch

**As** a user who changes the chat model while encrypted reasoning is still in the thread, **I want** to be told then, **so that** I still know after a pause that this model does not receive those blobs.

**Acceptance**

- The notice appears when the model changes, before the next message is sent.
- Reopening the thread later still shows the notice, with no message sent in between.
- No such notice appears when the thread has no encrypted reasoning, or when the chosen model will receive the blobs that are there.
