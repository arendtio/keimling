# Keimling HTTP agent server image (no native GUI).
# TLS terminates at your reverse proxy; this process speaks plain HTTP.
#
#   docker build -t keimling .
#   docker run --rm -p 7420:7420 \
#     -e OPENAI_API_KEY \
#     -v keimling-data:/home/nonroot/.local/state/keimling \
#     keimling --access-secret "$SECRET"
#
# Override listen / flags after the image name; ENTRYPOINT is the binary.

# ---- build (static, server-only) ----
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -tags nogui -o /out/keimling .

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="keimling" \
      org.opencontainers.image.description="Keimling HTTP agent server (nogui)"

COPY --from=build /out/keimling /keimling

# Sessions / index under $HOME; mount a volume if you want persistence.
ENV HOME=/home/nonroot \
    OPENAI_API_KEY=

EXPOSE 7420

# Bind all interfaces so a Traefik/Caddy/nginx proxy on another container can reach us.
ENTRYPOINT ["/keimling"]
CMD ["--server", "--listen", "0.0.0.0:7420"]
