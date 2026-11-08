# Chat branches — storage

Traces to: REQ-CB-1, REQ-CB-2, REQ-CB-4, REQ-CB-5.

The journal already stores each message once, append-only. A continuation is a path through those lines, not a second transcript.

## Path

Each journal item on a path is a node: an id, the parent node, and the line it points at. A user node may have several children that are alternative user messages. Each of those remembers the tip of its own continuation. The active tip is the path on screen and the path the next reply extends. Switching moves the active tip. The message text is not copied.

## Compacted memory

The charter (goal, plan, findings, questions, constraints, recent notes) is small and changes only when the model updates it or a compaction rewrites it. A revision is written only when the charter bytes change. A user node stores the revision id that was current when that message was accepted, before the reply to it updates the charter. Many messages share one revision.

If the goal was empty and would be filled from the user text at the start of the run, that fill runs again from the text actually sent on this continuation.

## What the model sees

The model sees the charter plus the tail since the last compaction, not the whole journal. A user node stores the id of that compaction and the ids of the items after it, up to this message. Those items are already in the journal. Rebuilding the tail reads them. It does not store them again.

Only the active path's tail and charter are loaded. Another continuation is read from disk when it is selected.

## Files

Beside the session journal: a branch index (nodes, parent links, revision ids, active tip) and a charter-revision log that grows only when the charter changes. Existing checkpoint lines stay the compaction snapshots they already are.
