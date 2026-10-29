# Feature: Credential Broker (MVP)

## Intent

Keimling verwaltet **Secrets für Tool-/MCP-Aufrufe**, ohne dass das Sprachmodell Passwörter, API-Keys oder Tokens je sieht.

Credentials sind **unabhängig** von den bestehenden Model-/OpenAI-Keys (`api.key`, Settings → Models). Der Broker ist kein Ersatz für Provider-Konfiguration.

Vertrauensmodell v1: **ein Operator** (Access Secret). Keine Multi-User-/Mandantentrennung.

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
3. **Auto-pick:** Ist für das Tool genau ein freigegebenes Credential vorhanden und fehlt `credential_id` → dieses verwenden. 0 → kein Secret (oder Fehler, wenn das Tool eins verlangt). >1 ohne ID → Fehler / Agent muss wählen.
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
