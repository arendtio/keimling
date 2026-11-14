# User stories: Scenario cards

## US-68 — See whether a harness change made a known task worse

**As** a user, **I want** a small set of tasks I can run again after a change to instructions or tools, **so that** I can see whether the outcome still holds and whether the model took more steps, tokens, or time.

**Acceptance:** traces REQ-SC-1 and REQ-SC-2. A finished scenario writes a card with the model, the commit, the outcome, the steps, the tools, the tokens, and the duration. Three runs report the middle step count and the middle token count. An earlier card for the same model stays readable beside the new one. A right outcome with a longer path is kept.

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

**Acceptance:** traces REQ-SC-8, REQ-SC-9, and REQ-SC-10. The command builds from the current source and runs all scenarios. A dirty build is named with the commit and a timestamp. The summary compares with the checkpoint when one is set, otherwise with the latest clean commit of the same model. Each run's journal remains on disk after the temporary workspace is gone. One missed repetition does not fail the command. The command fails when the set is worse.
