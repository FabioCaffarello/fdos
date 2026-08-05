#!/usr/bin/env bash
#
# Fitness function: builds are byte-reproducible.
#
# Constitution §9 requires a report to be reproducible years later. That is
# impossible if the binary producing it is not reproducible first — two builds
# of the same source that differ mean the toolchain, the environment or an
# embedded path has leaked into the output.
#
# This builds every command twice and compares digests. `-trimpath` removes
# absolute paths; `-buildvcs=false` removes the commit stamp, which changes
# between builds of identical source and would otherwise mask real
# irreproducibility with noise.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

GO="${GO:-go}"
BUILD_FLAGS=(-trimpath -buildvcs=false)
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

digest() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    sha256sum "$1" | awk '{print $1}'
  fi
}

failures=0
checked=0

printf 'Verifying build reproducibility...\n'

while IFS= read -r module; do
  [ -n "$module" ] || continue

  # Commands only. A library has no linked output to compare.
  commands="$(cd "$module" && GOWORK=off "$GO" list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./... 2>/dev/null || true)"

  while IFS= read -r pkg; do
    [ -n "$pkg" ] || continue
    name="$(basename "$pkg")"
    checked=$((checked + 1))

    ( cd "$module" && GOWORK=off "$GO" build "${BUILD_FLAGS[@]}" -o "${WORK}/${name}.1" "$pkg" )
    ( cd "$module" && GOWORK=off "$GO" build "${BUILD_FLAGS[@]}" -o "${WORK}/${name}.2" "$pkg" )

    a="$(digest "${WORK}/${name}.1")"
    b="$(digest "${WORK}/${name}.2")"

    if [ "$a" != "$b" ]; then
      printf '  %s: NOT reproducible\n    build 1: %s\n    build 2: %s\n' \
        "$pkg" "$a" "$b" >&2
      failures=$((failures + 1))
    else
      printf '  %-52s %s\n' "$name" "${a:0:16}"
    fi
  done <<EOF
$commands
EOF
done < <("${ROOT}/scripts/list-modules.sh")

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d command(s) did not build reproducibly.\n' "$failures" >&2
  exit 1
fi

if [ "$checked" -eq 0 ]; then
  printf 'OK: no commands to build yet.\n'
else
  printf 'OK: %d command(s) build reproducibly.\n' "$checked"
fi
