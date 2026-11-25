# Multi-stage build for the jobs Go service. The build context is
# this service directory plus the `packages` additional context from compose,
# which exposes the project-root packages/ tree so `replace` directives in
# go.mod (../../packages/<name>) resolve inside the builder image.
#
#   /workspace/
#     packages/...                    (from the packages context, when shipped)
#     services/jobs/    (this service — main context)

# --- Builder Stage ---
FROM golang:1.27-alpine AS builder

WORKDIR /workspace/services/jobs

# Module graph first so the dependency layer caches independently of source
# edits. A project generated on a host without Go has no go.sum yet; tidy
# resolves it inside the image in that case.
COPY go.mod go.sum* ./
RUN if [ -f go.sum ]; then go mod download && go mod verify; fi

COPY . .
# Every command ships: server, migrate, and the worker when async work adds it.
RUN if [ ! -f go.sum ]; then go mod tidy; fi \
    && for command in ./cmd/*/; do \
         CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "/out/$(basename "$command")" "$command" || exit 1; \
       done

# --- Runtime Stage ---
FROM alpine:3.22

RUN apk add --no-cache ca-certificates curl \
    && addgroup -g 10001 appuser \
    && adduser -D -u 10001 -G appuser -s /sbin/nologin appuser

WORKDIR /app
COPY --from=builder /out/ /app/
# Layered config profiles (RFC-008) resolve config/<ENV>.yaml relative to the
# working directory, so the runtime image ships them.
COPY config /app/config
RUN chown -R appuser:appuser /app
USER appuser

ENV PORT=5020
# Default to production so the fail-closed auth guard applies; the dev stack
# overrides ENV explicitly.
ENV ENV=production
EXPOSE 5020

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -fsS "http://localhost:5020/api/v1/health/live" || exit 1

CMD ["/app/server"]
