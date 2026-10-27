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
# Persist at /data (uid 65532). Named volumes inherit image ownership; bind mounts
# must be writable by 65532 (or chown). Do not mount over /home/nonroot.
# Secrets: OPENAI_API_KEY[_FILE], KEIMLING_SECRET[_FILE] (also KEIMLING_ACCESS_SECRET / ACCESS_SECRET).
# Override listen / flags after the image name; ENTRYPOINT is the binary.

# ---- build (static, server-only) ----
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -tags nogui -o /out/keimling .

# Empty state dir owned by distroless nonroot (65532) so named volumes stay writable.
RUN mkdir -p /data && chown 65532:65532 /data

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="keimling" \
      org.opencontainers.image.description="Keimling HTTP agent server (nogui)"

COPY --from=build /out/keimling /keimling
COPY --from=build --chown=65532:65532 /data /data

# Sessions / index under $HOME; mount a volume on /data for persistence.
ENV HOME=/data

EXPOSE 7420

# Bind all interfaces so a Traefik/Caddy/nginx proxy on another container can reach us.
# Access secret and API key come from env / *_FILE (Compose secrets), not CMD.
ENTRYPOINT ["/keimling"]
CMD ["--server", "--listen", "0.0.0.0:7420"]
