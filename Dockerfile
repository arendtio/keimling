# Keimling HTTP agent server image (no native GUI).
# TLS terminates at your reverse proxy; this process speaks plain HTTP.
#
#   docker build -t keimling .
#   docker run --rm -p 7420:7420 \
#     -e OPENAI_API_KEY_FILE=/run/secrets/openai_api_key \
#     -e KEIMLING_SECRET_FILE=/run/secrets/keimling_secret \
#     -v keimling-data:/data \
#     keimling
#
# Persist at /data (uid 65532): HOME, WORKDIR, and named-volume mount. Bind mounts
# must be writable by 65532 (or chown). Agent-created relative paths stay on the volume.
# Secrets: OPENAI_API_KEY[_FILE], KEIMLING_SECRET[_FILE] (also KEIMLING_ACCESS_SECRET / ACCESS_SECRET).
# Override listen / flags after the image name; ENTRYPOINT is the binary.
#
# Runtime is Debian slim (not distroless): the agent exec tool needs /bin/bash and
# basic Unix utilities inside the container.

# ---- build (static, server-only) ----
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -tags nogui -o /out/keimling .

# ---- runtime ----
FROM debian:bookworm-slim

LABEL org.opencontainers.image.title="keimling" \
      org.opencontainers.image.description="Keimling HTTP agent server (nogui)"

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      bash \
      ca-certificates \
      coreutils \
      curl \
      findutils \
      gawk \
      git \
      grep \
      procps \
      sed \
 && rm -rf /var/lib/apt/lists/* \
 && groupadd --gid 65532 nonroot \
 && useradd --uid 65532 --gid 65532 --home-dir /data --no-create-home --shell /bin/bash nonroot \
 && mkdir -p /data \
 && chown 65532:65532 /data

COPY --from=build /out/keimling /keimling

# Sessions / index under $HOME; CWD is /data so relative file writes hit the volume.
ENV HOME=/data
WORKDIR /data
USER nonroot:nonroot

EXPOSE 7420

# Bind all interfaces so a Traefik/Caddy/nginx proxy on another container can reach us.
# Access secret and API key come from env / *_FILE (Compose secrets), not CMD.
ENTRYPOINT ["/keimling"]
CMD ["--server", "--listen", "0.0.0.0:7420"]
