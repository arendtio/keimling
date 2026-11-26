# Delegate and subagents

`delegate` is currently one model call. It has no tools, no MCP, no file
access, and no main conversation history; `source`, `context`, and `image` are
the only inputs. That is why a request to fetch a wiki page through `delegate`
must fail: the delegate cannot call the wiki tool.

The open question is whether `delegate` should also be able to act as a
subagent with the caller's tools, permission-scoped.

## Options

1. **One tool, `tools` parameter.** `tools: "none"` keeps today's bounded
   second opinion. `tools: "all"` runs a nested agent loop with the caller's
   granted tools. This keeps one name and one mental model.
2. **Two tools.** `delegate` stays a model call; `subagent` is a separate tool
   for work that needs tools. Clearer, but the model has to choose between two
   similar names.

## Shape of `tools: "all"`

- The nested loop uses the caller's model profile, instructions, credentials,
  MCP manager, approval channel, and workspace.
- Its tool list is `filterGrantedTools(caller.chatExtraTools(),
  caller.instanceToolGrant())`, minus `delegate`. No recursion.
- Every tool call goes through the normal approval gate. A subagent cannot
  approve an action; a call that would ask the user asks the same user.
- The nested journal is temporary. Only the final answer returns to the parent;
  the parent's history does not receive every subagent tool result. This is the
  token saving.
- A step cap (start with 12) and the caller's spend budget bound the loop.
- The answer says which tools ran and what was verified. Unverified claims stay
  unverified.

## Recommendation

Add the `tools` parameter, keep `"none"` as the default for cost and
predictability, and make `"all"` opt-in. `"all"` is not a wider permission: it
is exactly the caller's grant, and it still asks when the caller would ask.
A separate `subagent` tool only becomes worthwhile if its prompt or budget
should differ from `delegate`.
