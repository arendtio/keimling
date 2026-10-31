# User stories: Thread model

Illustrates [REQ-TM-1](../features/08-thread-model.md) through [REQ-TM-5](../features/08-thread-model.md).

## US-32 — A different model for this thread

**As** a user with several providers and models in Settings, **I want** to pick the provider and model for the thread I am in, **so that** this chat does not have to use the global default.

**Acceptance**

- From the thread, the user can choose a provider and one of that provider's saved models.
- The choice is stored for that thread only and applies to the main chat model.
- Compaction, autopilot, delegates, and review stay on their Settings models.
- The next message sent in that thread uses that provider, model, and that model's reasoning and context limit.
- Another thread is unchanged.
- A thread with no choice of its own uses the current Settings default. A later change of that default changes that thread's chat model.
- Selecting the provider and model that Settings currently use clears the thread choice.

## US-33 — Switch model mid-thread

**As** a user in a thread that has already been talking, **I want** to switch its chat model, **so that** the next message I send uses the new one.

**Acceptance**

- A run already started finishes on the model it started with.
- The next message the user sends uses the newly chosen model, its reasoning, and its context limit.
- Earlier messages stay in the thread.
- Reopening the thread shows the new choice.
- If that new choice is the current Settings pair, the thread has no saved choice and follows later Settings changes.
