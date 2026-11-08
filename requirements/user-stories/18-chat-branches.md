# User stories: Chat branches

## US-51 — Continue from an earlier message

**As** a user, **I want** to edit a message I already sent and send the new text, **so that** the conversation continues from there on another path and the path I had remains.

**Acceptance:** traces REQ-CB-1 and REQ-CB-3. The visible chat follows the new text. The earlier reply is still available. Workspace files are unchanged.

## US-52 — Move between continuations

**As** a user, **I want** to move among every continuation of a message I sent, **so that** I can read and continue any of those chats later.

**Acceptance:** traces REQ-CB-2 and REQ-CB-4. Three continuations of one message can each be shown, including after a restart. The plan and the other compacted memory match the continuation on screen. The next reply follows that continuation.

## US-53 — Keep the past without copying it

**As** a user, **I want** the other continuations kept on disk without a full copy of the earlier chat beside every message, **so that** a long thread can be forked without storing the same history again and again.

**Acceptance:** traces REQ-CB-5. Forking an early message does not duplicate the transcript above it. After a restart the model is loaded with the visible continuation, and another continuation can still be selected.
