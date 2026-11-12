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
- **SSH-Agent (`type: ssh_key`):** Private Key kommt in einen ephemeren `ssh-agent` für diesen Lauf (`SSH_AUTH_SOCK`); Placeholder wird entfernt. Kein Key im Modellkontext.

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
