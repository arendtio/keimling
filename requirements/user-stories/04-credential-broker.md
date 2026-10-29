# User stories: Credential Broker

## US-30 — Secrets verwalten

**As** Operator, **I want** Credentials in Settings anzulegen/zu bearbeiten/zu löschen (Name, Typ, Service, erlaubte Tools, Secret), **so that** Tools sie nutzen können ohne das Modell einzuweihen.

**Acceptance**

- Vault-Datei verschlüsselt mit Master-Passwort; Unlock per Prompt und/oder Env.
- GET/API und UI zeigen nie den Secret-Wert (nur Metadaten / „gesetzt“).
- Mehrere Credentials pro Service möglich.
- Model-Provider-Keys sind davon unberührt.

## US-31 — Credentials für ein Tool entdecken

**As** Agent, **I want** `list_credentials` mit Ziel-Tool aufzurufen, **so that** ich eine erlaubte `credential_id` wählen kann.

**Acceptance**

- Antwort enthält nur id/name/description/type/service (keine Secrets).
- Nur Credentials mit diesem Tool in `allowed_tools` erscheinen.

## US-32 — Tool mit Credential ausführen

**As** Agent, **I want** ein Tool mit `credential_id` (oder Auto-Pick bei genau einem Treffer) auszuführen, **so that** der Connector authentifiziert, ohne dass ich das Secret sehe.

**Acceptance**

- Laufzeit prüft Freigabe und injiziert nur in den vertrauenswürdigen Adapter.
- Ungültige/fremde IDs werden abgelehnt.
- Secret erscheint nicht in Ergebnis, Logs oder Folge-Prompts.

## US-33 — Exec mit Env- und SSH-Credentials

**As** Agent, **I want** in `exec`-Befehlen `{{keimling:<id>}}` zu nutzen, **so that** Tokens per Vault-`inject_env` und SSH-Keys per ephemerem ssh-agent ankommen.

**Acceptance**

- Env-Name kommt aus dem Vault; Placeholder enthält die Credential-ID.
- Vor Env-Inject: Exfiltration-Review (Model-Call); fail-closed.
- `ssh_key`: Agent-Bridge, kein Key im Prompt; Output redacted.
