# Feature: Credential Broker (MVP)

## Intent

Keimling verwaltet **Secrets für Tool-/MCP-Aufrufe**, ohne dass das Sprachmodell Passwörter, API-Keys oder Tokens je sieht.

Credentials sind **unabhängig** von den bestehenden Model-/OpenAI-Keys (`api.key`, Settings → Models). Der Broker ist kein Ersatz für Provider-Konfiguration.

Vertrauensmodell v1: **ein Operator** (Access Secret). Keine Multi-User-/Mandantentrennung.

### REQ-CB-2 A remote MCP server takes a bearer token, and its transport is not a guess

- **Bin:** explicit requirement
- **Statement:** Adding a remote MCP server shall accept a bearer token, and shall not ask the user to choose HTTP or SSE.
- **Origin:** User, 2026-11-01 (quote: "http or sse is something that should be auto detected" and "there is no field whatsoever to add a bearer toke nor something similar" and "I thought you could change the form").
- **Rationale:** The server answers `authorization required` when the connection has no token, and the person adding it often does not know which of the two remote transports the server speaks.
- **Fit criterion:** The add form has a bearer-token field and no HTTP/SSE choice. A URL with no declared transport is used as streamable HTTP, and as SSE when that server rejects the HTTP handshake as the older transport. A refusal for a missing token stays a token error. The token is sent as `Authorization` and is not shown in the server list.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-3 An MCP call does not need a vault unless it asks for one

- **Bin:** explicit requirement
- **Statement:** Calling an MCP tool shall succeed without a vault when the call does not ask for a vault credential. A bearer token stored with that MCP server shall be used without creating or unlocking a vault.
- **Origin:** User, 2026-11-01 (quote: "credential error: vault locked or unavailable" and "I placed the token in the mcp settings. But I was never asked to unlok the vault at any time. in fact I did not even create a vault with this client yet.").
- **Rationale:** The connection token and the vault are different secrets. A tool such as listing projects only needs the server connection. Refusing that call because no vault exists blocks the server the person just configured.
- **Fit criterion:** With no vault file, an MCP tool call that does not name a credential is sent to the server and is not answered with "vault locked". A call that names a credential while the vault is locked is still refused. The bearer token on the MCP server is unchanged by either call.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-4 MCP sign-in uses loopback or the server's own address

- **Bin:** explicit requirement
- **Statement:** MCP sign-in shall accept the authorization callback on `127.0.0.1` and, when this Keimling is opened through its own domain, on that domain. It shall not use a private URI scheme.
- **Origin:** User, 2026-11-03 (quote: "klingt für mich als sollten wieder verschieden callback-url anbieten oder nicht? also sowohl 127 als auch custom domain wenn der server über eine eigene domain ausgeliefert wird. Eigenes Schema ist vermutlich nicht nur mit MCP server kompliziert." Also: "Systembrowser ist ok. Token können erstmal im mcp.json landen.").
- **Rationale:** A browser follows `127.0.0.1` to the computer it is running on. When the Keimling process is on another computer, only a callback on that Keimling's own address reaches it. A private scheme is rejected by authorization servers that only allow loopback or HTTPS.
- **Fit criterion:** A sign-in started from a local window or from a page on `127.0.0.1` registers and uses an `http://127.0.0.1` callback. A sign-in started from a page on the configured public domain registers and uses `https://` on that domain. No callback uses a `keimling://` scheme. The resulting tokens are stored in `mcp.json`.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-5 MCP setup offers the usual server credentials

- **Bin:** explicit requirement
- **Statement:** The MCP setup shall accept environment variables for a local command, a bearer token or named headers for a remote URL, and browser sign-in for a remote server that uses OAuth. Stored secrets shall not be shown again. Removing a server shall ask for its name.
- **Origin:** User, 2026-11-03 (quote: "dann gerne alles umsetzen was wir zum Thema MCP authorisierung besprochen haben." The agreed shape was local environment variables, remote bearer or headers, and browser sign-in; client certificates, URL tokens, client-credentials, and a private URI scheme were left out. "Systembrowser ist ok. Token können erstmal im mcp.json landen.").
- **Rationale:** MCP servers do not share one credential shape. A single bearer field blocks local servers and servers that sign the user in.
- **Fit criterion:** A local server saves name/value environment variables and a remote server saves either a bearer token or named headers, without showing those values in the server list. Browser sign-in stores access and refresh tokens in `mcp.json` and shows signed-in or sign-in-again on that row. A saved token is refreshed when it is due. Remove asks for the server name before deleting it, then drops that name from the user `mcp.json`, from `config.json`, and from the workspace `.keimling/mcp.json` and `.cursor/mcp.json`, so the server does not stay listed from one of those files. A server that is only present because a plugin is enabled stays; that plugin's own Remove is how it goes away.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-6 An MCP tool can be marked always allowed

- **Bin:** explicit requirement
- **Statement:** The user shall mark an MCP tool as always allowed, once, on that tool, including a tool on a server that comes from a plugin. In ask and in auto, a marked tool shall run without a question, for workspace threads and for agents. A tool that is not marked shall keep today's question. Clearing the mark shall bring the question back. Injecting a credential outside the model shall not add a question and shall not block a marked tool.
- **Origin:** User, 2026-11-04 (quote: "gerade bei MCP-Servern würde ich mir in Verbindung mit dem Ask-Modus oder auch dem Auto-Modus so eine Art Whitelist von Tools wünschen" and "man sagt, hier, das ist ein gewhitelistetes Tool, das darf immer ausgeführt werden" and "die ist einmal zentral am Tool"). Same day, on credentials (quote: "Ein Aufruf mit Zugang zu Geheimnissen, so ist unser Geheimniskonzept ja gar nicht aufgebaut. Also es ist ja so, dass der Bot niemals Zugriff auf die Geheimnisse kriegt oder das Model, sondern die Geheimnisse außerhalb stattfinden und dementsprechend ist das aus meiner Sicht gar keine Ausnahme"). User, 2026-11-12 (quote: "ich möchte auch einzelne methoden von servern die aus einem plugin kommen whitelisten können").
- **Rationale:** Read-only MCP calls stop an autonomous run on a question even though the user already trusts that tool. The mark is a property of the tool, not of one agent, and not of where the server was installed. The model never receives the secret, so the injection is not a second approval.
- **Fit criterion:** In the MCP setup, one tool of a server is marked and another is not. The same mark works on a server the user added and on a server that comes from a plugin. Marking the plugin server does not change the plugin's own files. In ask and in auto, the marked tool runs with no approval question, in a workspace thread and in an agent. The unmarked tool still asks in a workspace thread, still asks in an agent whose mode is ask, and is still refused without waiting in an agent whose mode is auto (REQ-AG-29). Clearing the mark makes the next call of that tool ask again. A marked call that injects a vault credential still does not show the secret to the model and is not held back for a question because of that injection. A tool the server adds later is unmarked. yolo is unchanged.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-7 The credential form explains the case

- **Bin:** explicit requirement
- **Statement:** The credential form shall explain, in the words of the job, what the chosen type is for and when each field is filled. A service may name several hosts in that one field. An SSH key shall explain the private-key file, the shell command, and the hosts. A password, API key, or token shall explain the environment variable for a shell command and the argument for an MCP tool.
- **Origin:** User, 2026-11-15 (quote: "wenn ich den Typ auswähle. Dann, ja, ich weiß nicht, danach habe ich das Gefühl. Ähm, ich brauche eigentlich erstmal 'ne Erklärung, was alles möglich ist. Also dass man bei der Domain mehrere, äh, Felder angeben kann. Ähm, dass, was ist jetzt ein Inject-as-SEP-Argument, Inject-as-Exec-EnfName, was ist. Secret oder. Also das, das verstehe ich alles nicht so richtig, wie ich das zu bedienen habe als Enduser.").
- **Rationale:** The field names name the mechanism. A person filling the form needs the case: which hosts, which command, which tool argument, and what to paste.
- **Fit criterion:** Choosing SSH key shows that the value is a private key file loaded for one shell command, that several hosts go in the one service field, and that the MCP argument and the environment variable are not part of that case. Choosing Password, API key, or Token shows that a shell command receives an environment variable and an MCP tool receives one argument. Each of those fields has that sentence next to it.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-8 A server has its own details, log, and reload

- **Bin:** explicit requirement
- **Statement:** Each MCP server shall open a details view from Manage MCP servers. That view shall show the server log, and it shall reload that server alone. A server the user added by hand shall be editable from that view. A local command shall take a directory it is started from. Reloading every server shall be a separate control, labeled as such. The side list shall stay the ready-check and the sign-in. A server that comes from a plugin is not edited in details.
- **Origin:** User, 2026-11-19 (quote: "Ich find den Button noch nicht, um mir das Log anzuzeigen" and "nicht nur den Dialog Allow Tools nennt, sondern irgendwie Details" and "auch 'n Reload-Button hätte pro Server" and "Server auch editieren können möchte" and "bei den Local Command Servern … ein Directory angeben" and "Reload-Button irgendwie auf der Seite mit der Liste … ob das den MCP-Server reloadet oder einfach die Liste neu lädt" and "auch in der Sidebar … keine Absprungmöglichkeit"). Same day (quote: "in der Sidebar ist mir der Details button bei den MCP zu groß" and "da reicht eigentlich der sign in button wenn relevant und sonst kann man ja über den manage MCP dialog gehen").
- **Rationale:** Connection failures and the process error stream were invisible. The list control restarts every server, which is a different job from restarting the one that failed. A hand-added command often needs a working directory. Plugin servers stay defined by the plugin.
- **Fit criterion:** Details on a server in Manage MCP shows Log, Reload, and Close. Log is a large read-only view of that server's recent errors and, for a local command, its error stream. Reload reconnects that server and leaves the others running. Reload all is the control that reconnects every server. For a server the user added, Edit can change the command or the URL, and a local command can set the directory it starts in. Saving that directory starts the next run there. A plugin server has Log and Reload and no Edit. The side list shows usable or not, and Sign in when a server needs it. It has no Details control. Remove still asks before deleting.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-9 A new sign-in uses the scopes from that attempt

- **Bin:** explicit requirement
- **Statement:** Signing in to an MCP server again shall request the scopes that server states for this attempt. It shall not request a scope list saved from an earlier sign-in.
- **Origin:** User, 2026-11-19 (quote: "wenn ich ihn wieder neu authentifizieren wollte, also Sign-in machen wollte, dann habe ich immer so eine Fehlermeldung bekommen … dass die Scopes, die angegeben worden sind, nicht valide waren, also irgendwie Invalid Scope" and "in der Zwischenzeit bei dem MCP-Server die erlaubten Scopes geändert worden sind, er aber irgendwo sich abgespeichert hatte, welche Scopes denn zur Authentifizierung genutzt werden sollten und er, wenn ich ihn jetzt erneut über Sign-in geschickt hab, immer noch wieder die alte Liste genommen hat und nicht die neue Liste").
- **Rationale:** A server can drop a scope after the first sign-in. Sending that old list back makes the new sign-in fail even though the server has already named the scopes it accepts now.
- **Fit criterion:** A server that was signed in with `old:read` answers the next sign-in with `fresh:read` and still lists `old:read` as a supported scope. The authorization request asks for `fresh:read` and does not ask for `old:read` or for the scope stored from the earlier sign-in.
- **Priority:** shall
- **Status:** confirmed

### REQ-CB-10 An expired access token is renewed without a new sign-in

- **Bin:** explicit requirement
- **Statement:** When an MCP server rejects a call because the short-lived access token no longer works, Keimling shall renew that token with the refresh token stored for that server and retry the call. The renewal request shall include the scope stored for that server. A browser sign-in shall be asked only when the refresh token itself is rejected.
- **Origin:** User, 2026-11-20 (quote: "der kurzfristige Token abgelaufen ist und der erneuert werden muss. Und das macht er im Moment nicht automatisch" and "wenn ich bei einem Service diesen Login wieder mache, dann sind auf einmal vier Services, alle die sich über Entra authentifizieren, alle wieder authentifiziert").
- **Rationale:** The access token lasts about an hour. The refresh token is the grant the user already gave. Asking for a password when that grant is still valid blocks every Entra server at once, and signing in to one of them only happens to renew the others as a side effect.
- **Fit criterion:** A call sent with an expired access token and a valid refresh token completes with the renewed access token and does not open a browser sign-in. The renewal request contains the scope stored for that server. When the refresh token is rejected, the call reports that sign-in expired and is not repeated in a loop.
- **Priority:** shall
- **Status:** confirmed

## Nicht-Ziele (v1)

- External vaults for MCP connection tokens. Those tokens stay in `mcp.json` (REQ-CB-4, REQ-CB-5).
- Client certificates, tokens in the URL, OAuth client credentials, and a private URI scheme.
- Pro-Agent-ACL auf einzelne Credentials
- Verschieben der Model-Provider-Keys in den Broker
- Ausbau langlebiger MCP-/XMPP-Sessions als Broker-Feature (siehe Hinweis unten)

## Datenmodell

Jedes Credential:

| Feld | Bedeutung |
| --- | --- |
| `id` | Stabil, eindeutig (vom Nutzer oder generiert) |
| `name` | Anzeigename |
| `description` | Optional |
| `type` | `api_key`, `password`, `token`, `ssh_key` |
| `service` | Optional: Domain/Host/Service-Hinweis für den Agenten |
| `allowed_tools` | Tool-Namen (`exec`, `mcp__…`, …) |
| `inject_arg` | MCP: Argumentname fürs Secret |
| `inject_env` | Exec: Env-Variablenname aus dem Vault (nicht vom Modell gewählt) |
| `secret` | Geheimer Wert — nur Laufzeit, nie API-GET/LLM |

Mehrere Credentials für denselben Service sind erlaubt.

### Exec: Placeholder + Env / SSH-Agent

- Im `exec`-Befehl referenziert der Agent Credentials nur als Placeholder **`{{keimling:<id>}}`** (ID aus `list_credentials`).
- **Env-Injection:** Vault-Feld `inject_env` setzt den Env-Namen. Placeholder wird durch `$<inject_env>` ersetzt; Secret nur in `cmd.Env`.
- Mehrere Credentials in einem Befehl: mehrere Placeholder mit unterschiedlichen IDs.
- **Exfiltration-Check:** Bevor Env-Secrets gesetzt werden, prüft ein zusätzlicher Model-Call den Befehl (ohne Secret-Werte, nur Env-Namen + Command) auf Secret-Exfiltration. Ablehnung → kein Inject, Befehl wird nicht mit Secrets ausgeführt. Fail-closed bei Review-Fehler.
- **SSH-Agent (`type: ssh_key`):** Private Key kommt in einen ephemeren `ssh-agent` für diesen Lauf (`SSH_AUTH_SOCK`); Placeholder wird entfernt. Kein Key im Modellkontext. Das Secret ist die private Schlüsseldatei, Zeilenumbrüche eingeschlossen. Ein Schlüssel, der ohne Zeilenumbrüche gespeichert wurde, wird vor `ssh-add` wieder in PEM-Zeilen gelegt.

## Speicherung (KeePass-artig)

- Eigene Datei neben der Config, z. B. `~/.config/keimling/credentials.vault` (Format intern, verschlüsselt).
- **Master-Passwort** schützt die Datei.
- Beim Start (GUI/Server): wenn die Vault-Datei existiert → Master-Passwort verlangen (oder aus Env).
- Env für Deployments: `KEIMLING_VAULT_PASSWORD` / `KEIMLING_VAULT_PASSWORD_FILE` (bevorzugt vor interaktivem Prompt).
- **Unlock gilt für die Lebensdauer des Keimling-Backend-Prozesses** (einmal entsperrt → Vault bleibt im Speicher bis Prozessende). Neustart erfordert erneutes Unlock (oder erneut Env).
- Ohne Vault-Datei startet Keimling wie bisher (Broker idle); Anlegen des ersten Credentials erzeugt die Vault und verlangt ein Master-Passwort.
- Keine Secrets in Logs, Errors, Tool-Results, SSE, Compaction.

## Laufzeitfluss

1. **Discovery:** eingebautes Tool `list_credentials` mit Pflichtargument `tool` → nur Metadaten der Credentials, deren `allowed_tools` dieses Tool enthalten.
2. **Selection:** Tool-Aufruf darf `credential_id` tragen (interne Referenz, kein Auth-Beweis).
3. **Auto-pick:** Ist für das Tool genau ein freigegebenes Credential vorhanden und fehlt `credential_id` → dieses verwenden. 0 → kein Secret. >1 ohne ID → Fehler / Agent muss wählen. Kein Vault, oder ein gesperrter Vault ohne `credential_id` → kein Secret und kein Fehler (die Verbindung nutzt ihr eigenes Token). `credential_id` bei gesperrtem Vault → Fehler.
4. **Authorization:** Laufzeit prüft: Credential existiert; Tool ∈ `allowed_tools`; (v1) Operator = Prozessbesitzer.
5. **Injection:** Secret nur an den **vertrauenswürdigen** Tool-Adapter / MCP-Connector der Laufzeit — nie zurück an das Modell. Konkrete Form (Header, Env für stdio-MCP, …) pro Connector; v1 mindestens ein klarer Pfad für MCP dokumentieren.
6. **Ergebnis:** nur bereinigtes Tool-Ergebnis an den Agenten.

Ungültige oder nicht erlaubte `credential_id` → Ablehnung.

Manipulierte Tool-Argumente dürfen Secrets **nicht** an beliebige URLs/Ziele umleiten: Injection nur in vom Connector festgelegte Auth-Slots, nicht in freie Modell-Strings.

## Verwaltung

- CRUD über Settings-UI (eigener Bereich, z. B. Tab **Secrets** / Credentials).
- Dieselbe Oberfläche über API (Access Secret), analog anderer Settings-Ressourcen; Secret-Wert write-only (GET nur Metadaten + `secret_set`).
- Vault entsperren: UI-/Start-Prompt und/oder Env.

## Agenten vs. Credentials

v1: Agenten werden auf **Tools/MCP-Server** berechtigt (bestehend bzw. wie heute).  
Credential-Freigabe läuft über **`allowed_tools`**, nicht über eine separate Agent↔Credential-Matrix.

## Hinweis: langlebige Verbindungen

Session-Haltung (z. B. XMPP über MCP) ist primär **MCP-Server-/Connector-Lifecycle**, nicht Broker-Kern.

Credentials betreffen das nur insofern, dass beim **Connect** einmal injiziert werden kann und die Session danach ohne erneute Secret-Übergabe an das Modell weiterläuft. Das Datenmodell braucht dafür keinen Extra-Typ in v1; spätere Connectoren können „credential at connect“ nutzen. Publish/Subscribe auf einer bestehenden MCP-Connection bleibt unabhängig vom Broker.

## Akzeptanz (MVP)

- Operator legt mehrere Credentials an und bindet sie an Tools.
- Agent listet per `list_credentials` nur erlaubte Metadaten.
- Tool-Lauf mit gewählter oder auto-gepickter `credential_id` funktioniert; unautorisierte IDs scheitern.
- Secrets erscheinen nicht in Modellkontext, Logs oder Tool-Ergebnissen.
- Model-/OpenAI-Keys bleiben unverändert außerhalb des Brokers.
