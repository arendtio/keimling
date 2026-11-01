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

## Nicht-Ziele (v1)

- OAuth-Flows, Token-Refresh, externe Vaults
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
