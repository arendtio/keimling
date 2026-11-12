# User stories: Reduce context

## US-64 — Shrink internal tools for a small model

**As** a user on a local model with a small context, **I want** Reduce context to show Off or On and to hide internal tool schemas only while it is on, **so that** the ordinary chat keeps those tools and a small model can turn the shrinking on.

**Acceptance:** traces REQ-LC-1. The plus row says Reduce context and Off. Turning it on says On and the request lists internal tool names without their schemas. One search loads an internal tool and an MCP tool. Compaction removes a loaded internal schema. Turning it off says Off and the internal schemas are in the request again. The choice is still the same when the chat is opened again.
