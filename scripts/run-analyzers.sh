#!/usr/bin/env bash
#
# Fitness function: domain purity and layer boundaries.
#
# Runs the FDOS analysers (libs/analysis) over every module. These are the
# mechanisms that turn the Constitution into build errors:
#
#   nofloat    §2, §9  — binary floating point is not associative, so a fold in
#                        a different order yields a different total
#   nondet     §2, §9  — clocks, randomness, environment, map iteration order
#   impurity   §3, §10 — I/O, concurrency, serialisation in the domain
#   layering   §3, §11 — layer inversion and cross-context coupling (ADR-0013)
#
# The analysers are themselves Go code with their own tests, exercised against
# both violating and compliant fixtures. See libs/analysis/README.md.
#
# Enforcement ladder position: static analysis, rung 2 (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
GO="${GO:-go}"
ANALYSIS_MODULE="libs/analysis"

printf 'Running FDOS determinism analysers...\n'

if [ ! -d "${ROOT}/${ANALYSIS_MODULE}" ]; then
  printf '  %s: missing — nothing to run\n' "$ANALYSIS_MODULE" >&2
  exit 1
fi

BIN="$(mktemp -d)/fdoslint"
trap 'rm -rf "$(dirname "$BIN")"' EXIT

( cd "${ROOT}/${ANALYSIS_MODULE}" && GOWORK=off "$GO" build -trimpath -o "$BIN" ./cmd/fdoslint )

failures=0
checked=0

while IFS= read -r module; do
  [ -n "$module" ] || continue
  checked=$((checked + 1))

  # The analysers apply to first-party packages. Their own fixtures under
  # testdata/ are deliberately violating code and are excluded by `./...`.
  if ! ( cd "${ROOT}/${module}" && GOWORK=off "$BIN" ./... ); then
    failures=$((failures + 1))
  fi
done < <("${ROOT}/scripts/list-modules.sh")

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d module(s) violate the domain purity or layering rules.\n' "$failures" >&2
  exit 1
fi

printf 'OK: %d module(s) clean.\n' "$checked"
