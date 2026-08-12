#!/usr/bin/env bash
#
# Run the Go checks over the modules a change affects, for fast failure.
#
# ADR-0014 rejected pruning the gate by affectedness — "under-reporting
# affectedness ships a broken module, and the failure is silent" — and said where
# the speed belongs instead:
#
#   "Speed belongs in a separate job."
#
# This is that job. It is **not** a gate and must never become one. `make verify`
# runs the full set regardless, and the only required status check stays
# `verify`.
#
# It cannot fail alone. Every command here is one `make verify` also runs, over a
# subset of the modules `make verify` covers, so a red preflight implies a red
# gate. That matters: a check that can fail on its own while the gate is green is
# a check people learn to ignore, and this repository has already disabled one
# for exactly that reason.
#
#   make affected-preflight
#
# Enforcement ladder position: none. Advisory by construction.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

GO="${GO:-go}"
BASE="${1:-}"

affected="$(scripts/affected-modules.sh "$BASE" || true)"

if [ -z "$affected" ]; then
  printf 'Nothing affected. The gate still runs everything.\n'
  exit 0
fi

count="$(printf '%s\n' "$affected" | grep -c .)"
total="$(scripts/list-modules.sh | grep -c .)"
printf 'Preflight over %d of %d module(s):\n\n' "$count" "$total"

failures=0

while IFS= read -r module; do
  [ -n "$module" ] || continue
  printf '>>> %s\n' "$module"

  # GOWORK=off, GOFLAGS and the cgo settings mirror the Makefile exactly. This
  # is a faster subset of the gate, not a different one — the moment it runs
  # something the gate does not, a green preflight stops predicting anything.
  if ! ( cd "$module" && GOWORK=off "$GO" vet ./... ); then
    failures=$((failures + 1))
    continue
  fi
  if ! ( cd "$module" && GOWORK=off golangci-lint run ./... ); then
    failures=$((failures + 1))
    continue
  fi
  if ! ( cd "$module" && GOWORK=off CGO_ENABLED=1 "$GO" test -race ./... ); then
    failures=$((failures + 1))
  fi
done <<EOF
$affected
EOF

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d affected module(s) did not pass.\n' "$failures" >&2
  printf 'The gate will fail on this too — this only says so sooner.\n' >&2
  exit 1
fi

printf '\nOK: %d affected module(s) pass vet, lint and test.\n' "$count"
printf 'This is not the gate. `make verify` runs the full set.\n'
