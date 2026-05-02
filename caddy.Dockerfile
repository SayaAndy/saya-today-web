FROM docker.io/library/caddy:2.11.2-builder-alpine AS builder

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    xcaddy build v2.11.2 \
    --with github.com/mholt/caddy-ratelimit

FROM docker.io/library/caddy:2.11.2-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
