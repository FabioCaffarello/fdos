#!/usr/bin/env bash
#
# Print the pinned version of a tool from mise.toml.
#
# This is the single parser for the toolchain pins. `make toolchain-check`, the
# vulnerability scan and every CI workflow read the pin through here, so there
# is exactly one place that knows the format and no opportunity for CI to
# install a version the repository did not ask for.
#
#   scripts/tool-version.sh go
#   scripts/tool-version.sh go:golang.org/x/vuln/cmd/govulncheck
#
# With no argument, lists every pinned tool as "name<TAB>version".

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
MISE_TOML="${ROOT}/mise.toml"

if [ ! -f "$MISE_TOML" ]; then
  printf 'tool-version: mise.toml not found\n' >&2
  exit 1
fi

list() {
  awk '
    /^\[/ { in_tools = ($0 ~ /^\[tools\]/); next }
    !in_tools { next }
    /^[[:space:]]*#/ { next }
    /=/ {
      key = $0
      sub(/[[:space:]]*=.*$/, "", key)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", key)
      gsub(/^"|"$/, "", key)

      val = $0
      sub(/^[^=]*=[[:space:]]*/, "", val)
      gsub(/"/, "", val)
      gsub(/[[:space:]]+$/, "", val)

      if (key != "") printf "%s\t%s\n", key, val
    }
  ' "$MISE_TOML"
}

if [ "$#" -eq 0 ]; then
  list
  exit 0
fi

want="$1"
version="$(list | awk -F'\t' -v w="$want" '$1 == w { print $2; exit }')"

if [ -z "$version" ]; then
  printf 'tool-version: %s is not pinned in mise.toml\n' "$want" >&2
  exit 1
fi

printf '%s\n' "$version"
