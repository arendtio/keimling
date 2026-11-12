# MCP calls

Traces to REQ-MCP-2, REQ-MCP-3, and REQ-MCP-4.

Chat and a DataLang program share one scheduler. The program is the earlier
tool program: it calls MCP tools, its approval is decided before it starts, and
its calls use this same scheduler.

## Scheduler

A slot is one call that is talking to a server. The cap is how many slots may
be taken. It is stored in the config as `mcp_parallel`, starts at 10, and is
edited in settings the same way as `max_steps`. A waiting call does not take a
slot.

Stdio is one process on one connection, keyed by server name. That key is
exclusive: the next call to the same server starts only after the current one
finishes. Two stdio servers have two keys, so they may each hold a slot.
HTTP and SSE have no per-server lock. Two writes to the same HTTP server may
overlap; only stdio is serialized.

Chat tool waves and the program both submit here. A wave cannot exceed the cap
by starting more; further calls wait for a free slot. Stop cancels the calls
that have started and the ones still waiting.

## Chat

MCP calls from one model response join the scheduler instead of the serial tool
path. Reads and a plain shell keep their existing parallel wave. Each MCP call
is still approved on its own before it starts. Allowed calls then share the
slots. A denied call does not start; the others that were allowed do.

The next model step waits until those calls have finished. The results are the
reason for the step. The live call's five-second release, where speech
continues and the result arrives later, stays on the call. The chat shows each
call as running, and the composer can still queue, steer, and stop.

## Program

A sequential call submits one job and waits for it. A parallel group submits
its calls and waits for the group. The group is written in the program, so the
tools are known before the start and REQ-MCP-2 applies to the whole program.
Inside the group the scheduler still applies the cap and the stdio key. A later
step that needs a returned value stays sequential.

DataLang writes a sequential call as `mcp(name, args)` and a parallel group as
`all(...)`. Those two operations are the program side of this design. They do
not add a second runtime. A name that is not a string literal in the program
is not part of the approval and does not run.
