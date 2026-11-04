# Feature: Parallel tools and a tighter work prompt

## REQ-PT-1 Independent tools in one response run together

- **Bin:** explicit requirement
- **Statement:** When the model asks for several tools in one response, the ones that can run together shall run together, instead of each one waiting for the previous to finish.
- **Origin:** User, 2026-11-04 (quote: "die parallele Toolausführung finde ich tatsächlich sehr spannend … ist, glaube ich, auch etwas, was wir grundsätzlich umsetzen sollten").
- **Rationale:** Searching, reading, and checks that do not depend on each other are the slow part of a turn when they run one after another.
- **Fit criterion:** Two non-interactive commands issued in the same response overlap in wall time. A command that needs the terminal does not run beside another command. A file edit does not run beside another tool.
- **Priority:** shall
- **Status:** confirmed

## REQ-PT-2 The work prompt states the rules that change what the model does

- **Bin:** explicit requirement
- **Statement:** The standing work instructions shall state instruction priority, smallest scope, search-then-range reading, a failing check before a bugfix, no weakened tests, no claimed pass without tool output, no repeat of a failed approach without new output, and a reversible default instead of a question. They shall stay about that size rather than grow by restating how each tool is called.
- **Origin:** User, 2026-11-04 (quote: "auch die anderen Sachen, die du empfohlen hast, gerne so einmal direkt umsetzen" and, earlier, wanting quality at a similar length rather than always adding text).
- **Rationale:** Those rules change the next action. Tool manuals already sit on the tool definitions.
- **Fit criterion:** The work instructions contain the priority order, the scope rule, the failing-check rule, and the stop-without-new-output rule. They do not contain the exec filtering manual, the sudo invocation, or the artifact-refresh manual.
- **Priority:** shall
- **Status:** confirmed

## Design notes (not requirements)

A call runs beside others only when it is a read (`read_file`, history, lists, `web_search`, `agent_instructions`) or a non-interactive `exec` with no credential placeholder. Edits, writes, the terminal, credential exec, MCP, delegates, and other mutations run one at a time and do not overlap that group. Consecutive calls of the parallel kind form one wave; a mutation breaks the wave, so a later check sees that mutation. Approvals are still asked one at a time before the wave starts. Results are recorded in call order. `parallel_tool_calls` is on so the model may emit the batch. The sentence that the user controls the PTY stays on the workspace thread and is still removed from an agent instance.
