# Multi-stage build for the cv Go service. The build context is
# this service directory plus the `packages` additional context from compose,
# which exposes the project-root packages/ tree so `replace` directives in
# go.mod (../../packages/<name>) resolve inside the builder image.
#
#   /workspace/
#     packages/...                    (from the packages context, when shipped)
#     services/cv/    (this service — main context)

# --- Builder Stage ---
FROM golang:1.27-alpine AS builder

WORKDIR /workspace/services/cv

# In-tree SDK source for the go.mod replace directives. Only copied when a
# project-root packages/ tree ships (auth); the compose `packages` context is
# gated on the same condition.
COPY --from=packages . /workspace/packages/

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
# Node, because the service runs the Claude Code CLI to write CVs (docs/claude-setup.md).
FROM node:26-slim

# Pinned; bump deliberately: the service reads the CLI's flags and result format.
ARG CLAUDE_CODE_VERSION=2.1.322
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && npm install -g --no-fund --no-audit "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
    && npm cache clean --force \
    && groupadd -g 10001 appuser \
    && useradd -u 10001 -g appuser -M -s /usr/sbin/nologin appuser
ENV DISABLE_AUTOUPDATER=1

WORKDIR /app
COPY --from=builder /out/ /app/
# Layered config profiles (RFC-008) resolve config/<ENV>.yaml relative to the
# working directory, so the runtime image ships them.
COPY config /app/config
RUN chown -R appuser:appuser /app
USER appuser

ENV PORT=5040
# Default to production so the fail-closed auth guard applies; the dev stack
# overrides ENV explicitly.
ENV ENV=production
EXPOSE 5040

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -fsS "http://localhost:5040/api/v1/health/live" || exit 1

CMD ["/app/server"]
