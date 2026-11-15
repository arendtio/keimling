# User stories: Scenario cards

## US-68 — See whether a harness change made a known task worse

**As** a user, **I want** a small set of tasks I can run again after a change to instructions or tools, **so that** I can see whether the outcome still holds and whether the model took more steps, tokens, or time.

**Acceptance:** traces REQ-SC-1, REQ-SC-2, and REQ-SC-18. A finished scenario writes a card with the model, the commit, the outcome, the steps, the tools, the tokens, and the duration. Three runs report the middle step count and the middle token count. An earlier card for the same model stays readable beside the new one. A right outcome with a longer path is kept. A token or time move inside 5% of the baseline, or inside the spread of the runs, is the same result when passes and steps are unchanged.

## US-69 — Find one entry, one hidden value, and one sentence in another chat

**As** a user, **I want** those repeated tasks to cover the cases the harness was built for, **so that** a regression shows up on our own features.

**Acceptance:** traces REQ-SC-3, REQ-SC-4, and REQ-SC-5. The phone-book answer is the number that exists only on the last row. The unknown-file answer is the value past the preview, not the decoy. The other chat's sentence is in the answer and that journal's bytes are unchanged.

## US-70 — Transform a long tool result without a delegate

**As** a user, **I want** a repeated task where a long tool result must be projected in full, **so that** the model writes a data program instead of asking a delegate to extract every row.

**Acceptance:** traces REQ-SC-6. A stored list of 100 rows is projected to key, title, person, and dependencies, and the row bodies stay out of that result. A delegate call on that list marks the path wrong even if the projected text shows up in the answer.

## US-71 — Keep the scenarios out of the binary

**As** a user, **I want** the scenario tasks kept with the developer tools, **so that** the binary I run does not grow every time a scenario is added.

**Acceptance:** traces REQ-SC-7. The shipping binary does not contain the scenario task texts or fixtures. A scenario run starts that binary, and the card is taken from its journal.

## US-72 — Run every scenario on the source I just edited

**As** a user, **I want** one command to build my current source, run every scenario, and tell me what got better or worse, **so that** I can look before a commit without assembling the runs myself.

**Acceptance:** traces REQ-SC-8, REQ-SC-9, REQ-SC-10, REQ-SC-19, and REQ-SC-20. The command builds from the current source and runs all scenarios. A dirty build is named with the commit and a timestamp. The summary compares with the checkpoint when one is set, otherwise with the latest clean commit of the same model. Each run's journal remains on disk after the temporary workspace is gone. Each run reads a copy of the configuration in that directory, and the user's configuration stays unchanged. The runs proceed together unless a maximum is set. The report stays grouped by scenario. One missed repetition does not fail the command. The command fails when the set is worse.

## US-74 — Catch the other ways a tool is chosen

**As** a user, **I want** repeated tasks for a missing file name, a verdict, a count, a join, a decoy in a preview, and a command sitting in the data, **so that** a change that avoids one wrong tool does not create another.

**Acceptance:** traces REQ-SC-11, REQ-SC-12, REQ-SC-13, REQ-SC-14, REQ-SC-15, and REQ-SC-16. A phone book question that omits the file name prefers a search plus a data program and rejects a delegate. A verdict code that is not in the file prefers a delegate, and the opening of the file points the other way. A count prefers a data program that counts or groups. A role that exists only across two lists prefers a lookup and rejects a delegate. A plain-text preview holds a decoy, and the answer is the later code. A row that holds a number and a command returns the number, leaves the file unchanged, and does not run the command.

## US-75 — Read one cell from a column-major table

**As** a user, **I want** a repeated task whose stored table is a large XML document with names in a header and cells listed down each column, **so that** a change handles the same shape as a data-preview response.

**Acceptance:** traces REQ-SC-17. The table has 120 columns. The opening of the document does not contain the asked cell. The answer is the cell that shares its index with a late key, and the first cell of that column is a decoy. A data program is preferred, and a delegate is a wrong path.

## US-78 — Show every service in one table

**As** a user, **I want** a repeated task whose stored service list is past the small inline size, **so that** a table of every service is one read and not a series of narrower ones.

**Acceptance:** traces REQ-SC-21. The list is larger than 4 KiB and its opening does not contain the host. The answer names an early service, the last service, and that host. A data program that asks for inline is preferred, and a delegate is a wrong path.
