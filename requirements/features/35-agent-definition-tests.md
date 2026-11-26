# Feature: Tests for an agent definition

A test checks the logic in one agent definition. The other parties are a script. The wording of an outgoing message may vary. Whom it is addressed to is exact.

## REQ-ADT-1 A definition carries its tests

- **Bin:** explicit requirement
- **Statement:** The user shall attach one or more tests to an agent definition and run a test from that definition. Each test shall run on its own.
- **Origin:** User, 2026-11-26 (quote: "unter Managed Ratings 'ne Definition des Ratings anlegen" and "zu dieser Definition auch Tests anlegen" and "Ich möchte Agenten-Definitionen testen können"). Spoken "Ratings" is the agent definition edited in Manage agents.
- **Rationale:** The logic under test is the definition. A test that lives somewhere else is no longer about that definition.
- **Fit criterion:** One definition has two tests. Starting one runs that test and leaves the other unstarted. A definition with no test still starts an instance as it does today.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-2 A test plays the other parties from a script

- **Bin:** explicit requirement
- **Statement:** A test shall start one instance of its definition. The other parties shall be the script, not further model instances. The script shall send the messages it names, and it shall expect the messages the instance sends. An expected message may be addressed to a party other than the sender of the message that came before it.
- **Origin:** User, 2026-11-26 (quote: "ich habe eine Initiierung des Agenten" and "dann kriegt er teilweise Nachrichten und er sendet weitere Nachrichten" and "Das muss nicht sein, dass quasi die Antwort genau an den gleichen Empfänger zurückkommt"). Same day, on the counterpart (quote: "Skript reicht").
- **Rationale:** A second live agent would make the result depend on a second prompt. The definition is the thing being tested.
- **Fit criterion:** A script sends a message as party A and expects the next outgoing message to be addressed to party B. A later step sends as party C. No second model instance is started for A, B, or C. The instance under test is an instance of the definition the test belongs to.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-3 The recipient is named by the message or by the definition

- **Bin:** explicit requirement
- **Statement:** During a test, the instance shall learn whom to write to only from the definition instructions, from a scripted message, or from both. A scripted message may name a recipient other than its sender. The definition may name a standing recipient for the messages the instance sends. A later message from a different party shall not replace a standing recipient that the definition names. The instance may send when the script has no further message. The test shall not add its own instruction about whom to write to.
- **Origin:** User, 2026-11-26 (quote: "kann es sein, dass die Message kommt und sagt, wem der Agent antworten soll? Oder es könnte sozusagen im Initial-Prompt mit drinstehen, an wen er immer Nachrichten schreiben soll" and "beides möglich sein" and "wenn er eine Nachricht vorschickt und danach schickt jemand anders nochmal eine Nachricht, dann könnte das, dass er eigentlich zu bezwecken, der steht es im wichtigsten Punkt" and "Oder es kommt keine Nachricht mehr").
- **Rationale:** Some jobs answer the sender. Some jobs always report to one party, even after someone else writes, and some send because the definition said so before anyone wrote.
- **Fit criterion:** In one test the definition names no recipient and the scripted message tells the instance to answer party B; the outgoing message is addressed to B. In a second test the definition says to write status to party A, a message arrives from party B and does not name a recipient, and the outgoing message is addressed to A. In a third test the definition says to report results to party A; the instance writes to party B, party B replies, and the following message is addressed to A. In a fourth test the definition says to send a status to party A at the start, the script sends nothing, and the instance sends that status to A. None of these tests inserts an extra instruction that names the recipient.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-4 The address is exact and the wording is judged

- **Bin:** explicit requirement
- **Statement:** Each expected message shall name its recipient, and that recipient shall match exactly. The same expected message shall carry a check prompt. The check shall accept different wordings that satisfy the prompt and reject a message that misses what the prompt asks. A failure shall say which expected message failed.
- **Origin:** User, 2026-11-26 (quote: "dann erwarte ich, dass er irgendeinen Schritt, Adresse, irgendeinen Schritt Nachricht schickt" and "wie man dann validiert, ob die Nachricht, die er schickt, korrekt ist" and "Weiche Instruktionen geben, also auch einen Prompt, mit dem validiert werden kann" and "von der Formulierung her immer unterschiedlich").
- **Rationale:** The address is the part that does not vary. The sentence does. A single verdict over the whole run hides which step failed.
- **Fit criterion:** Two outgoing messages with different wording and the same required point both pass one check prompt. A message that lacks that point fails it. A message addressed to a party other than the expected recipient fails even when the wording would have passed. The failure names that expected message.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-5 A test supplies tool results

- **Bin:** explicit requirement
- **Statement:** A test shall be able to supply the result of a tool the instance calls. That call shall receive the supplied result and shall not run the tool on the host.
- **Origin:** User, 2026-11-26 (quote: "Tools können wir gerne simulieren. Das finde ich völlig in Ordnung").
- **Rationale:** A prompt test should fail because the definition did the wrong thing, not because a file or a command was missing.
- **Fit criterion:** The instance calls a tool for which the test supplies a result. The instance's next step can use that result. The host command or file named by the call is not executed and not read. A granted tool with no supplied result, an empty result, and a tool the definition does not grant are covered by REQ-ADT-7, REQ-ADT-8, and REQ-ADT-9.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-7 A granted tool returns a simulated result

- **Bin:** explicit requirement
- **Statement:** During a test, every tool the definition grants should return a simulated result when the instance calls it.
- **Origin:** User, 2026-11-26 (quote: "Alle Werkzeuge die für den agenten erlaubt sind, sollten in der simulation etwas zurückgeben").
- **Rationale:** The simulation stands in for the tools the definition is allowed to use. A granted tool the test never answers is a hole in the test. An empty answer is still an answer.
- **Fit criterion:** The instance calls each tool the definition grants. Each call receives a simulated result, and an empty result counts as that result. The host does not run the call.
- **Priority:** should
- **Status:** confirmed

## REQ-ADT-8 An empty tool result is allowed

- **Bin:** explicit requirement
- **Statement:** An empty simulated result of a tool the definition grants shall be delivered to the instance. The run shall continue. That emptiness alone shall not record the agent as having failed.
- **Origin:** User, 2026-11-26 (quote: "Wenn die Rückgabe eines Werkzeugs leer ist, weil der test schreiber es schlecht definiert hat, dann ist das nicht die Schuld des agenten"). Same day, later (quote: "ein leeres Werkzeug ergebnis sollte erlaubt sein").
- **Rationale:** An empty result is a result the author may mean, including when a lookup finds nothing. The agent is judged on the messages it sends after that, not on the fact that the tool came back empty.
- **Fit criterion:** A granted tool whose simulated result is empty delivers that empty result, and the run continues. The agent is not failed for the emptiness. A later message that misses its check prompt still fails the agent. The host does not run the call.
- **Priority:** should
- **Status:** confirmed

## REQ-ADT-9 A forbidden or unknown tool gets the ordinary reply

- **Bin:** explicit requirement
- **Statement:** When the instance calls a tool the definition does not grant, or a name that is not a tool, the instance shall receive the same reply it would receive for that call outside a test.
- **Origin:** User, 2026-11-26 (quote: "Wenn er ein nicht erlaubtes werkzeug aufruft, bekommt der die normale reaktion die ein agent bekommt der ein Werkzeug aufruft, für das er nicht authoriziert ist oder das es gar nciht gibt").
- **Rationale:** The test is about the definition's logic. A special test error for a forbidden call would hide the reply the definition actually has to deal with.
- **Fit criterion:** The definition grants a set of tools. The instance calls a tool outside that set and receives the reply a normal instance of that definition receives for the same call. The instance calls a name that is not a tool and receives the reply a normal instance receives for that name. Neither call runs on the host.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-6 A test runs until it fails or three times

- **Bin:** explicit requirement
- **Statement:** A test shall run up to three times. A run that fails shall stop the test, and the remaining runs shall not start. Each run that did start shall keep its own result. The test as a whole shall pass only when three runs have passed.
- **Origin:** User, 2026-11-26 (quote: "so einen Test auch dreimal durchlaufen lässt, reicht" and "Alle läufe müssen bestehen"). Same day, later (quote: "Von mir aus muss er auch nicht dreimal durchlaufen, wenn er schon beim ersten Mal fehlschlägt" and "sobald er fehlschlägt, er abbricht").
- **Rationale:** One pass does not show that the definition holds. A failed run already shows that it does not. Further runs after that failure do not change the result.
- **Fit criterion:** A test whose first run fails stores that run and does not start a second. A test whose second run fails stores the first two and does not start a third. The test passes only when three runs have each passed the recipient and the check prompt. A failed run stays visible. An empty tool result under REQ-ADT-8 does not by itself fail a run.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-10 A test can be written while the definition is created

- **Bin:** explicit requirement
- **Statement:** The user shall write a test on the same screen where a definition is first created, before that definition has been saved. Saving the definition shall store those tests. A saved definition shall still be where its tests are edited and run.
- **Origin:** User, 2026-11-26 (quote: "warum kann ich nicht bei der initialen Erstellung auch schon Tests anlegen?").
- **Rationale:** The definition and the test are one job. Saving the definition first, then coming back to add the test, splits that job.
- **Fit criterion:** On the new-definition screen the user can enter a test. Saving the definition stores that test on it. Opening the definition afterwards shows the test, and Run is available there. Leaving the new-definition screen without saving does not store the test.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-11 The parts of a test are entered in their own fields

- **Bin:** explicit requirement
- **Statement:** The user shall enter each party as an id, a name, and a kind, each in its own field, and shall add another party without a line format. The user shall enter each script step as who writes, that message, who should receive the agent's message, and the check, each in its own field. A step may omit the incoming message, the expected message, or neither. The user shall enter tool results from the tools the definition grants, and for each such tool shall set the result of the first call, the second call, and further calls, each in its own field.
- **Origin:** User, 2026-11-26 (quote: "das einfach in ein Textarea reinzuknallen, wo es doch irgendwie was ist, was aus drei Komponenten besteht, einer ID, einem Namen und dann einem Typ" and "viel einfacher, wenn man das in ein vernünftiges, tabellenartiges Formular packt" and "das nächste genau das Gleiche" about the script). Same day, later (quote: "das mit den komischen Formularen, alles in einem Area, gilt für die Tools unten auch" and "die Liste an Tools und könnte dann eintragen, was es beim ersten Tool aufruft, beim zweiten, dritten und so weiter, was es da jeweils immer zurückgibt").
- **Rationale:** A party, a step, and a tool call are fixed shapes. A line the user has to punctuate correctly is where tests fail before the agent runs.
- **Fit criterion:** Two parties are added as separate id, name, and kind fields. One step has an incoming message from one party and an expected message to another, entered as fields, with no `send` or `expect` line syntax. Another step has only an expected message. For one granted tool, the first call receives one result and the second call receives a different result. A tool the definition does not grant is not offered as a result to fill in.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-12 The explanation stays out of the form and shows a passing example

- **Bin:** explicit requirement
- **Statement:** The explanation of how a test works shall stay closed until the user opens it from the Tests heading. That explanation shall include one concrete example of a definition and a test that passes.
- **Origin:** User, 2026-11-26 (quote: "da drunter gibt's zwar so eine kleine Beschreibung, die würde ich aber tatsächlich auch eher hinter irgendeinem Info-I verstecken. Und da sollte aber ein konkretes Beispiel drin sein, wie so ein Test erfolgreich aussehen könnte" and "vielleicht machen wir's auch übergreifend für die Testkategorie").
- **Rationale:** The form is the job. The explanation is reference, and an example is what makes a first test possible.
- **Fit criterion:** Opening a definition does not show the long explanation under the fields. Opening it from the Tests heading shows one example that names instructions, parties, steps, and a tool result, and that example is one that the checks would pass.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-13 Each run can be opened as the agent's chat

- **Bin:** explicit requirement
- **Statement:** After a test has run, the user shall open each run and see that run's chat as an agent's chat is shown: the messages, the tool calls, and what came back.
- **Origin:** User, 2026-11-26 (quote: "wenn so ein Test gelaufen ist, dass ich das genaue Chat-Verlaufsfenster sehen kann, von dem Agent selber, also seinen internen Zustand quasi, und sehen kann, wie er sich verhalten hat, was er gemacht hat").
- **Rationale:** A pass or a one-line failure does not show what the agent did. The chat is the record of that.
- **Fit criterion:** A finished test shows three runs. Opening one run shows that run's transcript in the same form as an agent chat, including a scripted message, a tool call, the result that call received, and the outgoing message. The transcript is still available after the test instance is gone. A run from before this record was kept says that its chat was not kept.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-14 Manage agents has room for these forms

- **Bin:** explicit requirement
- **Statement:** Manage agents shall use a large surface, comparable to the sidebar focus view, so the definition and its tests can be laid out as forms.
- **Origin:** User, 2026-11-26 (quote: "dieser ganze Dialog für Manage Agents, der ist ja eher so von der Größe etwa wie der Settings Dialog" and "ob das nicht hier auch sinnvoller wäre, einfach den größeren Platz zu haben" like "die Sidebar zum Beispiel so in Fokus Mode").
- **Rationale:** The definition form plus parties, steps, and per-call tool results do not fit a settings-sized dialog.
- **Fit criterion:** Manage agents occupies most of the window, on the order of the sidebar focus overlay, and is visibly larger than the settings dialog. The test form's party, step, and tool fields are usable in that surface without a line format.
- **Priority:** should
- **Status:** confirmed

## REQ-ADT-15 The test list, the test, and a run’s chat are separate

- **Bin:** explicit requirement
- **Statement:** On a definition, the user shall see the list of tests and shall run a test from that list. Creating or editing a test shall happen in its own dialog. Opening a run shall show that run’s chat in its own dialog. The definition’s fields shall stay out of that chat.
- **Origin:** User, 2026-11-26 (quote: "wenn ich einen neuen Test anlege, dass da ein neuer Dialog aufgeht" and "in der übergreifenden Ansicht nur die Liste der Tests" and "wenn ich jetzt auf Chat klicke und dann wird nur der Chat angezeigt, aber der obere Teil des Dialogs ist immer noch die Agent Definition").
- **Rationale:** The definition, the test being written, and one run’s chat are three jobs. One scrolling pile makes it unclear which fields belong to the test.
- **Fit criterion:** The definition screen shows the test list, Run, and each run. New test opens a dialog that contains the test fields and Save. Closing it returns to the list. Chat opens a dialog whose content is that run’s transcript. The definition’s name and instructions are not part of that chat dialog.
- **Priority:** shall
- **Status:** confirmed

## REQ-ADT-16 The form says who is under test

- **Bin:** explicit requirement
- **Statement:** The test form shall say that the agent under test is not a party. It shall say that a scripted message is written to that agent, and that an expected message is a message that agent sends. The kind of a party shall be described as how that party is introduced, not as which agent is under test.
- **Origin:** User, 2026-11-26 (quote: "muss ich den Agent mit als Party definieren oder nicht?" and "Woher weiß man, welche Party jetzt der Agent ist?" and "Expect muss ja eigentlich nur etwas sein, wo der Agent etwas getan hat und eine Nachricht geschickt hat").
- **Rationale:** A test that does not say who is being tested cannot be filled in. Expect read as a check of the incoming message tests the wrong thing.
- **Fit criterion:** The test dialog states that parties are the other side and that this agent is not a party. A step’s incoming line is labelled as writing to this agent. A step’s expected line is labelled as this agent writing to a party. The party kind is labelled as thread or agent for how that party writes, and the text says it does not choose the agent under test.
- **Priority:** shall
- **Status:** confirmed

## Design notes

These are not requirements. They are the choices that currently satisfy the fit criteria above.

- A tool result listed once is what every call receives. A second result is what the second call receives. A call after the last listed result receives that last result again.
- `message_agent` is the script, not a tool-result row. Its reply stays the ordinary sent-message text.
- On the new-definition screen, tests stay on that screen until Save definition. Run waits until the definition exists. On a saved definition, Save test stores the test immediately.
- Each run stores the chat view the product already uses for an agent. Opening a run renders that view. The temporary test instance is still removed afterwards.
