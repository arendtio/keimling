# Feature: Watch commands

Time alarms stay. A watch is an additional way for an agent to be woken.

## REQ-WC-1 A check runs without the model

- **Bin:** explicit requirement
- **Statement:** An agent shall be able to run a check at a regular interval without starting the model. The model shall start only when the check matches.
- **Origin:** User, 2026-11-07 (quote: "nicht jedes Mal das LLM starten, um diesen Befehl auszuführen, sondern man könnte diesen Befehl im Hintergrund irgendwo laufen lassen als deterministisches Programm und erst wenn es einen bestimmten Wert hat, dann wird das LLM informiert" and "führe alle 10 Sekunden diesen Befehl aus und wenn er sich ändert, dann sag mir Bescheid").
- **Rationale:** A clock alarm starts the model on every ring so it can look whether anything happened. The look itself does not need the model.
- **Fit criterion:** A watch with a 10 second interval runs its check on that interval while the model is idle. Intervals whose result does not match do not start a model request. A matching result starts the agent and includes that result.
- **Priority:** shall
- **Status:** confirmed

## REQ-WC-2 The check can be any granted tool, including a chain

- **Bin:** explicit constraint
- **Statement:** The check shall be a shell command, an MCP call, or any other tool the agent may use, and it shall be possible to chain tools, including data processing.
- **Origin:** User, 2026-11-07 (quote: "Die Befehle, die dafür genutzt werden können, sollen sowohl MCP-Befehle sein können … als auch Befehle, die auf der Shell ausgeführt werden" and "eigentlich alle Tools sollen möglich sein" and "eine Verkettung von Tools, beispielsweise mit dem Data Processing").
- **Rationale:** Shell and MCP are both tools. A chain lets the check reduce a large result before the comparison.
- **Fit criterion:** One watch can run a shell command. Another can run an MCP tool. Another can run a tool and then data processing on that result. The condition sees the chain's result. A tool the agent is not allowed to use is not a watch step.
- **Priority:** shall
- **Status:** confirmed

## REQ-WC-3 The result is compared before the agent is told

- **Bin:** explicit requirement
- **Statement:** A watch shall compare the result with a condition. Two conditions are a result that equals a given value, and a result that changed.
- **Origin:** User, 2026-11-07 (quote: "wenn der einen Rückgabewert X hat. Dann bitte, ja, starte mich" and "wenn er sich ändert, dann sag mir Bescheid" and "es muss natürlich irgendwie eine Form von Rückgabewertprüfung geben").
- **Rationale:** Without a comparison, every interval would wake the model, which is the cost this watch exists to avoid.
- **Fit criterion:** A watch waiting for a value X wakes the agent on every run whose result is X, including when X was already the result last time. A watch waiting for a change wakes the agent when the result differs from the previous result. A result that does not match does not wake the agent.
- **Priority:** shall
- **Status:** confirmed

## REQ-WC-4 A watch stays until the agent turns it off

- **Bin:** explicit requirement
- **Statement:** A watch shall stay active until the agent that created it deactivates it.
- **Origin:** User, 2026-11-07 (quote: "eine Überwachung sollte so lang aktiv bleiben, bis sie vom agenten wieder deaktiviert wird").
- **Rationale:** A match is a signal, not the end of the watch.
- **Fit criterion:** After a match, the same watch is still active. It ends when that agent deactivates it, and then it no longer runs.

## REQ-WC-8 A watch belongs to one instance

- **Bin:** explicit requirement
- **Statement:** A watch belongs to the agent instance that created it. That instance lists only its own watches. Deleting the instance removes them and stops their file watches and command processes.
- **Origin:** User, 2026-11-07 (quote: "watches agent instanz spezifisch sind, also jede agent instanz sieht nur ihre watches und diese werden automatisch bereinigt wenn man den agent löscht").
- **Rationale:** A deleted instance must not leave a check running, and one instance must not see another instance's watches.
- **Fit criterion:** `list_watches` on an instance returns only the watches stored for that instance. After the instance is deleted, its watch file is gone and a command it started is no longer running. Another instance's watches are unchanged.
- **Priority:** shall
- **Status:** confirmed

## REQ-WC-5 A watch can wait for an event instead of an interval

- **Bin:** explicit requirement
- **Statement:** An agent shall be able to set a watch with no interval. A watch on a file or directory shall notify the agent when that path changes. A command-line command shall be able to run the same way, staying open and reporting when it produces a result.
- **Origin:** User, 2026-11-07 (quote: "es sollte auch die Möglichkeit geben, Async Befehle zu bauen" and "eine Überwachung auf ein Verzeichnis oder eine Datei" and "sobald sich was ändert und man kein fixes Intervall festlegen muss" and "dafür könnte man fsnotify fix mit einbauen, aber auch andere Befehle (auf der CLI) sollen funktionieren"). This replaces the earlier decision to leave filesystem notifications out of the first step.
- **Rationale:** Polling a file on a timer spends runs that an event can replace. The same shape should fit a command that waits and then speaks.
- **Fit criterion:** A directory watch has no interval and wakes the agent when the directory changes. A long-running command has no interval and wakes the agent from its output when the condition matches. Both stay until the agent deactivates them.
- **Priority:** shall
- **Status:** confirmed

## REQ-WC-6 Watch replaces alarm

- **Bin:** explicit constraint
- **Statement:** The agent shall set, list, and turn off watches, and shall not have separate alarm tools. A watch with a time and no condition shall wake the agent the way an alarm did.
- **Origin:** User, 2026-11-07 (quote: "können wir das nicht vereinfachen und alarm streichen und nur noch watch nutzen? ein set_watch ohne bedingung wäre dann praktisch das gleiche wie bisher alarm und wir sparen uns ein paar kleine manage tools" and "sollten wir es vielleicht create_watch nennen, sonst könnte man auch denken man stellt damit eine Uhrzeit?" and "alarms.json müssen wir nicht weiter supporten. stattdessen gerne eher sowas wie watch/watcher/watchers.json" and "Eine dauerhafte Prüfung alle 10 Sekunden mit Nachrichten hier kann ich nicht zuverlässig ausführen" and "hast Du nicht die Möglichkeit create_watch zu nutzen?").
- **Rationale:** A time wake and a conditional wake are the same job. Two tool sets make the agent choose a name for one job.
- **Fit criterion:** The tools are `create_watch`, `list_watches`, and `cancel_watch`. There is no `set_alarm`, `list_alarms`, `cancel_alarm`, or `set_watch`. A watch with a time and no condition wakes once for a time such as "in 2m" or "at 15:04", and wakes on each interval for "every 30s" until the agent cancels it. The instructions tell the agent that `create_watch` wakes only itself. The list is stored as `watches.json` in the instance directory. An existing `alarms.json` is not read. A workspace thread is not offered these tools. The check runs while a model reply is still in progress.
- **Priority:** shall
- **Status:** confirmed

## REQ-WC-7 A workspace chat offers an agent for a watch

- **Bin:** explicit requirement
- **Statement:** A workspace chat shall not set a watch. It shall know that an agent can, tell the user that, and after the user agrees create an agent, start an instance, and hand that instance the requested job.
- **Origin:** User, 2026-11-07 (quote: "ein normaler Chat keine Watch einstellen sollte, sondern dass das eben genau die Funktion eines Agenten ist" and "der Thread weiß, welche zusätzlichen Möglichkeiten ein Agent hat" and "dass er selbst das nicht kann, weil er kein Agent ist und dass er aber einen Agenten erstellen kann" and "wenn der User das möchte, dann sollte er in dem Moment auch einen Agent erstellen, der genau diese Funktion auch direkt umsetzt" and "erstellen und instanziieren können und auch damit starten" and "das sollten wir in das agentische Muster bringen und nicht an den Workspace hängen" and "die instruktionen sind beim anlegen des agenten noch nicht automatisch spezifisch genug" and "ob der thread eigentlich seine eigene id kennt?").
- **Rationale:** A watch that keeps running is what an agent is for. Giving the workspace that loop would make the workspace an agent. A vague job is not a purpose the later approval can match, and the workspace can name the thread only when it knows that thread's id.
- **Fit criterion:** A workspace thread has no `create_watch`. Its instructions say a watch belongs to an agent, this chat cannot set one, and they include this thread's id. After the user agrees, the chat saves a definition whose job names the concrete check and that thread id as the place to message, starts an instance, and queues that job as the instance's first message. The workspace does not run the check. A match is shown in the agent channel as a Watch report, with the result, and not as a message from the user. It does not open a new chapter.
- **Priority:** shall
- **Status:** confirmed
