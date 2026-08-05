#!/usr/bin/env bash
#
# Fitness function: all Go source is canonically formatted.
#
# Formatting is not a style preference here. Reproducibility (Constitution §9)
# starts with the source being identical for everyone who checks it out, and a
# diff full of formatting noise hides the change that actually matters from
# review.
#
# Analyser fixtures under testdata/ are excluded: they are deliberately
# malformed Go in places, which is what makes them fixtures.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

printf 'Verifying Go formatting...\n'

unformatted="$(
  find libs apps -name '*.go' -not -path '*/testdata/*' 2>/dev/null -print0 \
    | xargs -0 gofmt -l 2>/dev/null || true
)"

if [ -n "$unformatted" ]; then
  printf '  the following files are not gofmt-clean:\n' >&2
  printf '    %s\n' $unformatted >&2
  printf '\nFAIL: run `make fmt`.\n' >&2
  exit 1
fi

count="$(find libs apps -name '*.go' -not -path '*/testdata/*' 2>/dev/null | wc -l | tr -d ' ')"
printf 'OK: %s Go files formatted.\n' "$count"
