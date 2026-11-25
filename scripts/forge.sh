#!/usr/bin/env bash
# Runs the forge build pinned in .forge/quality.json. Each pin is installed once, outside the repo (the architecture
# gate scans every file under it), in ${XDG_CACHE_HOME:-~/.cache}/hoply/forge-<hash of the pin>.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
recipe="$root/.forge/quality.json"

pin="$(python3 -c '
import json, sys
recipe = json.load(open(sys.argv[1]))
print("\n".join([recipe["generator_requirement"], *recipe.get("plugin_requirements", [])]))
' "$recipe")"
hash="$(printf '%s' "$pin" | shasum -a 256 | cut -c1-16)"
venv="${XDG_CACHE_HOME:-$HOME/.cache}/hoply/forge-$hash"

if [[ ! -f "$venv/.installed" ]]; then
  command -v uv >/dev/null || { echo "forge.sh: uv is required (https://docs.astral.sh/uv/)" >&2; exit 1; }
  rm -rf "$venv"
  uv venv --quiet --python 3.13 "$venv"
  requirements=()
  while IFS= read -r line; do [[ -n "$line" ]] && requirements+=("$line"); done <<< "$pin"
  uv pip install --quiet --python "$venv/bin/python" "${requirements[@]}"
  touch "$venv/.installed"
fi

exec "$venv/bin/forge" "$@"
