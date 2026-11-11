# User stories: MCP tools

## US-61 — Keep unused MCP tools out of the context

**As** a user with many MCP servers, **I want** the model to see what each server is for and to load only the tools it is about to use, **so that** the context is not already full of tools it will not call.

**Acceptance:** traces REQ-MCP-1. A large catalog shows one short line per server. Loading tools for one job returns at most five, and those can be called. Compaction removes them. A small catalog stays loaded. Built-in tools stay loaded.
