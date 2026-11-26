# Delegate and subagents

`delegate` has two modes.

`tools: "none"` (the default) is one model call. It has no tools, no MCP, no
file access, and no main conversation history; `source`, `context`, and `image`
are the only inputs. This is the bounded second opinion.

`tools: "all"` runs a nested agent loop with the caller's tools. It takes a
task and context only; a source or image belongs to the tool-less mode.

## The nested loop

- It uses the caller's model profile, workspace, credentials, MCP manager, and
  approval channel.
- The tool list is `filterGrantedTools(caller.chatExtraTools(),
  caller.instanceToolGrant())`, with `delegate` and `finish_task` removed.
  `delegate` would recurse; `finish_task` belongs to the parent task. A caller
  that grants nothing still grants nothing.
- Every tool call goes through the normal approval gate. A subagent cannot
  approve an action; a call that would ask the user asks the same user.
- The subagent writes a temporary journal. Only its final answer returns to the
  parent, so its tool results do not flood the parent transcript.
- The loop stops after 12 steps. A missing final answer is an error, not a
  half-finished transcript.
- Tool findings are skipped inside the subagent: the raw result is already in
  its temporary transcript, and a finding would cost one more model call per
  tool.

## Why not always `all`

The tool-less call is cheaper, predictable, and has no side effects. A
subagent that can edit files or run commands should be a deliberate choice.
