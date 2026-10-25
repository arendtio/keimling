# Concept and guiding principles

## What Keimling is

Keimling is a minimalist, general-purpose terminal agent for Linux. It is built
to run on old x86-64 machines with little RAM: model inference happens remotely
through APIs, while the local binary handles agent orchestration, terminal
integration, file operations, context management, and security decisions.

Keimling grew out of an earlier project named *tinyagent*. The name changed; the
goals did not. Everything below uses the current name, and configuration paths,
session locations, and the binary are all `keimling`.

## Guiding principles

- **A small, statically linked Go binary.** No agent framework, no mandatory
  external language servers, parser executables, or companion services.
- **Model and provider independence** through Responses-compatible endpoints.
- **Lean, language-agnostic tools** instead of many specialised tools competing
  for context.
- **Transparent terminal output** and automation the user can control.
- **A complete, persistent original history.** Only the active model context may
  be compressed; the journal is never rewritten.
- **Capability through tools, skills, and techniques**, not through an
  ever-growing core binary.

## Platform

| Aspect | Decision |
| --- | --- |
| Operating system | Linux, x86-64 |
| Runtime footprint | Low RAM and CPU; the original target was a netbook with roughly 1 GiB RAM |
| Language | Go, standard library preferred, statically linked |
| Companion processes | None required; voice capture may use system tools such as `arecord` |
| Configuration | `~/.config/keimling/config.json` |
| Session data | `~/.local/state/keimling/` |

## Providers and APIs

- The main model is reached through an OpenAI-compatible **Responses API**,
  selected with `--model` and `--base-url`.
- OpenAI, OpenRouter, and compatible local endpoints such as llama.cpp are
  supported. A general Chat Completions adapter is not part of the design;
  `--api-mode` accepts `responses` only.
- API keys come from files or environment variables. Credentials for external
  OpenAI services (web search, transcription) are kept separate from the model
  provider key, so the provider never receives the OpenAI key.
- External services can be disabled with `--disable-external-services`.
- OpenRouter routing suffixes such as `:floor`, `:nitro`, or `:exacto` must not
  break model-catalog validation: the catalog is checked against the base model
  ID, while the full suffixed ID is sent in the actual request.
- Model validation can be skipped with `--check-model=false`.
- The context window can be discovered from provider metadata where possible;
  `--context-limit` remains the explicit override and fallback.
- Compaction can use a separate model via `--compaction-model`, on the same
  endpoint and credentials by default.

## Reliability and retries

- Transient failures are retried automatically: HTTP 429 and 5xx, a Responses
  status of `failed` with a transient `server_error`, network timeouts, and
  connectivity errors such as `network is unreachable`, `no route to host`,
  connection reset or refused, and temporary DNS failures.
- Five retries follow the initial attempt, with exponential backoff of 1, 2, 4,
  8, and 16 seconds. Waits stay cancelable.
- Non-retryable errors, such as invalid parameters, are reported immediately.
- Broken IPv6 connectivity is not treated as a permanent request error. An
  explicit IPv4 fallback was discussed but is not a mandatory core feature.

## Where the project deliberately stays small

- No embedded Tree-sitter, ctags, or LSP stack, and no symbol tools in the core.
  An experimental branch showed roughly a sevenfold binary-size increase.
  Language-aware analysis belongs in an optional external integration such as an
  MCP server; MCP itself is a possible extension, not a current requirement.
- No bundled debugging methodology skill.
- No vector index or embedding service for history search.

See [security-and-quality.md](security-and-quality.md) for the full list of
deferred items and the acceptance criteria for further development.
