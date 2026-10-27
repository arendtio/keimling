# File shares (capability URLs)

## Goal

Let the agent expose **selected** files over the HTTP agent server so a GUI or browser can open/download them (e.g. behind a TLS reverse proxy), without making the whole workspace public.

## Model

- Tool `share_file` registers an absolute path (must exist, regular file, inside the working directory, not a private path).
- Argument `as`: `auto` (default), `view` (inline), or `download` (attachment). Auto uses inline for common viewable types (HTML, PDF, images, text, …).
- Server issues an opaque **token** (cryptographically random). Knowledge of the token is authorization (capability URL).
- `GET /files/{token}` serves the file **without** the access secret.
- Shares are in-memory for the process; cleared on `/api/newchat` / process exit.
- Re-sharing the same path may reuse or replace the token (implementation: new token each share is fine).
- Agent instructions: when offering a file to the user remotely, call `share_file` and paste the URL.

## URL

Public base (no trailing slash), in order:

1. Config / flag `public_url` if set (e.g. `https://keimling.example.com`)
2. Else last request’s `X-Forwarded-Proto` + `X-Forwarded-Host` (or `Host`)

Tool result includes the full URL when a base is known; otherwise the path `/files/{token}` and a note to open it on the agent host.

## Limits

- Max file size for share/serve: 32 MiB
- No path escape outside working directory; no private credential paths
- `Content-Type` from extension; `Content-Disposition` from `as` or auto heuristic
