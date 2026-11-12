# User stories: MCP tools

## US-61 — Keep unused MCP tools out of the context

**As** a user with many MCP servers, **I want** the model to see what each server is for and to load only the tools it is about to use, **so that** the context is not already full of tools it will not call.

**Acceptance:** traces REQ-MCP-1. Every connected server appears as one short line, and its tool schemas stay out until the one search loads them. That search also finds an internal tool, returns at most five, and a loaded MCP tool can be called. Compaction removes it from that chat. A new chat does not include it. Built-in tools stay loaded while Reduce context is off. There is no separate MCP search.

## US-62 — Allow a tool program once, before it runs

**As** a user in ask mode, **I want** a program that calls tools to be allowed or questioned once before it starts, **so that** a run of already-trusted tools does not stop, and a run that needs permission does not start halfway.

**Acceptance:** traces REQ-MCP-2. A program of only whitelisted or non-asking tools runs with no question. A program that includes one tool that would ask shows one question with those tools and calls none until allow. After allow it does not ask again. A tool that was not part of the decision is not called.

## US-63 — Run MCP calls together without crowding one server

**As** a user, **I want** chat MCP calls to run together in the background up to a limit I can set, **so that** independent calls do not wait in a single line and one stdio server is not driven twice at once.

**Acceptance:** traces REQ-MCP-3 and REQ-MCP-4. Several MCP calls from one chat response run together, at most the saved limit at once, starting from ten. Two calls to the same stdio server do not overlap. Calls to two stdio servers may overlap. The limit is in settings and survives a restart.
