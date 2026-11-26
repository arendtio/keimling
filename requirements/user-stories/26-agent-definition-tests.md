# User stories: Tests for an agent definition

## US-85 — Test the logic of an agent definition

**As** a user, **I want** to attach a script to an agent definition and run it, **so that** I can see whether that definition writes to the right party and says the right thing.

**Acceptance:** traces REQ-ADT-1, REQ-ADT-2, REQ-ADT-3, REQ-ADT-4, REQ-ADT-5, REQ-ADT-7, REQ-ADT-8, and REQ-ADT-9. Two tests on one definition start separately. The other parties are the script. One script expects a reply to a party the latest sender did not name, because the message said so. Another expects the standing recipient named in the definition, including after a different party writes, and including when the script sends nothing. The test adds no addressing instruction of its own. The recipient matches exactly. Two wordings of the same point pass one check prompt, and a miss fails it, naming that expected message. A granted tool returns the simulated result the test supplied, including an empty one, and the host does not run that call. The empty result does not by itself fail the agent. A tool the definition does not grant, and a name that is not a tool, return the same reply they return outside a test.

## US-86 — See the same test three times

**As** a user, **I want** one test to run three times and keep each result, **so that** I can see that it works more than once when the model varies.

**Acceptance:** traces REQ-ADT-6 and REQ-ADT-13. One start runs until a run fails or three runs have passed. The test passes only when three runs passed. A failed run fails the test, stays visible, and no further run starts. Opening a run shows that run's chat, including messages and tool calls, after the test instance is gone.

## US-87 — Write the test in fields, including on a new definition

**As** a user, **I want** to fill in parties, steps, and each tool call as fields while I create the definition, **so that** I can write a test without learning a line format and without saving the definition first.

**Acceptance:** traces REQ-ADT-10, REQ-ADT-11, REQ-ADT-12, REQ-ADT-14, REQ-ADT-15, and REQ-ADT-16. The definition screen shows the test list. New test and Edit open the test in its own dialog. A run’s chat opens in its own dialog. Saving the definition stores a test written before the definition exists. Each party is an id, a name, and a kind. The form says this agent is not a party, that a scripted message is written to this agent, and that the expected line is a message this agent sends. A granted tool has a result for the first call and a separate result for the next call. The example stays closed until it is opened, and the opened text includes one passing example. Manage agents uses most of the window. The Definitions, Instances, and Teams controls stay the size of their labels.
