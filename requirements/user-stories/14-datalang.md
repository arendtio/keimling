# User stories: DataLang

## US-48 — Filter a tool result without shell or Python

**As** a user, **I want** the agent to filter and reshape a tool result in a language that cannot touch the system, **so that** a large MCP or CLI result can be reduced without a script.

**Acceptance:** traces REQ-DL-1, REQ-DL-2, and REQ-DL-3. A stored issue list is filtered to the open bugs. The model record of the original call does not contain the full list when it is large. A program that asks for a file or a process fails. The filtered rows can be passed into a second transform by reference.

## US-49 — Combine two stored results

**As** a user, **I want** the agent to join or compare two tool results in one transform, **so that** a lookup or a set check does not require a script.

**Acceptance:** traces REQ-DL-5 and REQ-DL-6. Two stored lists are combined by reference: rows whose key is absent from the second list drop out, and a field from the matching row is attached. The default timeout is 30 seconds, and a longer requested timeout is kept.

## US-65 — Name a file or a chat section instead of copying it

**As** a user, **I want** a transform to take a file path or a section of this chat, **so that** the agent does not copy that text into the conversation in order to filter it.

**Acceptance:** traces REQ-DL-8 and REQ-RF-1. A large whole-file read shows a preview and the path. A named line range is exact. A transform given that path returns a value from the file. A transform given a history section id returns a value from that section. The program does not open the file or the journal itself.

## US-66 — See the shape, then ask about the file

**As** a user, **I want** the agent to see how a large file is built and to ask a question about it without keeping the file in the chat, **so that** the answer comes back without a series of guesses.

**Acceptance:** traces REQ-DL-9, REQ-RF-2, REQ-DD-1, and REQ-DD-2. The shape of a large JSON file names a field and a list length and leaves out a value that lives only in the list. A delegate call names the file and returns an answer. The file text is on the delegate request and not in the tool arguments. A model window of 1,000,000 tokens can take a source larger than 400 KiB in one call.

## US-67 — Search a file of any size

**As** a user, **I want** a search over a phone-book-sized file to return one entry, **so that** the size of the file does not block the search.

**Acceptance:** traces REQ-DL-10. A CSV larger than 8 MiB, searched for one name, returns that row and leaves the other rows out of the result. The run does not fail because of the file's size. The shape of that file names the header and the row count and leaves out a cell value.
