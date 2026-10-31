# User stories: Autopilot instructions

Illustrates [REQ-AP-1](../features/07-autopilot-instructions.md) through [REQ-AP-8](../features/07-autopilot-instructions.md).

## US-30 — Global stance for the autopilot

**As** a user, **I want** to replace the factory autopilot prompt in Settings, and to put that factory prompt back, **so that** the default "do not authorize system changes" stance is not forced on every project.

**Acceptance**

- Settings exposes the global autopilot instruction text, next to the existing Autopilot on/off control.
- Saving a non-empty text persists across restart. Saving an empty or whitespace-only text is refused.
- A reset clears the saved global text. Threads with no thread text then use exactly the factory prompt.
- For a thread with no saved thread text, changing the saved global text changes the next supervision prompt to exactly that new text.
- The approval mode is unchanged.

## US-31 — Different instructions per thread

**As** a user running one thread that must change a remote machine over SSH, and other threads that must not, **I want** that thread's text to replace the autopilot prompt entirely until I reset it, **so that** only that thread keeps the looser stance.

**Acceptance**

- From the thread, the user can read and edit that thread's autopilot instructions. Preferred entry, as named by the user: the top-right More menu (Context, Artifacts).
- While the thread has no saved text, the field shows the effective global text, and a later change of that global text shows up here.
- Saving an empty or whitespace-only text is refused. Saving a single word succeeds and detaches the thread.
- Saving the text that was already shown also detaches the thread. A later global edit does not change that thread.
- Two threads can store different texts at the same time.
- Supervision of a detached thread sends exactly that thread's saved text.
- A reset button clears the saved thread text. The thread follows the global text again, including a global edit made after the reset.
- Reopening the thread shows the text that is in effect.
- The approval mode is unchanged.
