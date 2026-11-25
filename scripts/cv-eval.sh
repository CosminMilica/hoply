#!/usr/bin/env bash
# Runs the CV writer's evaluation (services/cv/eval/README.md) on this machine, with the Claude Code CLI version the
# cv image pins, installed once in ${XDG_CACHE_HOME:-~/.cache}/hoply/claude-code-<version>, and the local token.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="$(sed -n 's/^ARG CLAUDE_CODE_VERSION=//p' "$root/services/cv/Dockerfile")"
cli="${XDG_CACHE_HOME:-$HOME/.cache}/hoply/claude-code-$version"

if [[ ! -x "$cli/node_modules/.bin/claude" ]]; then
  command -v npm >/dev/null || { echo "cv-eval.sh: npm is required" >&2; exit 1; }
  npm install --silent --no-fund --no-audit --prefix "$cli" "@anthropic-ai/claude-code@$version"
fi

export CLAUDE_BIN="$cli/node_modules/.bin/claude"
export CLAUDE_TOKEN_FILE="${CLAUDE_TOKEN_FILE:-$root/deploy/compose/secrets/claude_oauth_token}"
export PROFILE_SERVICE_URL="${PROFILE_SERVICE_URL:-http://localhost:5010}"
export JOBS_SERVICE_URL="${JOBS_SERVICE_URL:-http://localhost:5020}"
cd "$root/services/cv"
GOTOOLCHAIN=local exec go run ./cmd/eval "$@"
