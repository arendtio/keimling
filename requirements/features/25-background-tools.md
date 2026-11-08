# Feature: Background tools on a call

## REQ-BG-1 A live call does not wait on tools

- **Statement:** While a live call is open, tool calls shall run in the background so the conversation can continue. A result that arrives within about five seconds is used directly. Only a call that is still running after that wait is described as still running. When every tool from that batch has finished or timed out, the call shall hear the remaining results and comment on them before treating the topic as done.
- **Origin:** User, 2026-11-08 (paraphrase: a live call blocks while tools run; run them like a shell background job and wait for the batch before the agent comments; MCP calls on a call should be background by default). Same day: two seconds is too short; wait about five seconds before treating the call as still running and continuing the conversation.
- **Fit criterion:** Several tools requested in one spoken turn start together. If they finish within about five seconds, the call comments on the real results and does not say it is still waiting. If any are still running after that, the call may say so, and a later follow-up carries those results. A tool that fails is included as an interruption, not omitted.
- **Priority:** shall
- **Status:** confirmed
