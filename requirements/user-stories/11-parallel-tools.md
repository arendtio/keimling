# User stories: Parallel tools

## US-42 — Independent checks at the same time

**As** a user, **I want** reads and checks the model asks for together to run together, **so that** a turn does not wait out each command in sequence.

**Acceptance:** traces REQ-PT-1 and REQ-PT-2. Two non-interactive commands in one response overlap. A terminal command and a file edit stay on their own. The work instructions state priority, scope, the failing check, and stopping without new output, and they no longer restate the exec, sudo, and artifact manuals.
