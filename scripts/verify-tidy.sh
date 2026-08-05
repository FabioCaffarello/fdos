#!/usr/bin/env bash
#
# Fitness function: go.mod and go.sum are tidy in every module.
#
# An untidy module is a dependency graph nobody reviewed. Missing sums make a
# build non-hermetic; stale requirements make the build list depend on when it
# was last resolved rather than on what the source declares — which breaks
# reproducibility (Constitution §9) before any FDOS code runs.
#
# Runs `go mod tidy` against a copy and diffs. The working tree is never
# modified: a check that fixes what it checks cannot report a failure.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
GO="${GO:-go}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

failures=0
checked=0

printf 'Verifying module tidiness...\n'

while IFS= read -r module; do
  [ -n "$module" ] || continue
  checked=$((checked + 1))

  scratch="${WORK}/${checked}"
  mkdir -p "$scratch"
  # Copy the module so `go mod tidy` cannot touch the working tree.
  ( cd "${ROOT}/${module}" && tar cf - . ) | ( cd "$scratch" && tar xf - )

  ( cd "$scratch" && GOWORK=off GOFLAGS= "$GO" mod tidy >/dev/null 2>&1 ) || {
    printf '  %s: `go mod tidy` failed\n' "$module" >&2
    failures=$((failures + 1))
    continue
  }

  for f in go.mod go.sum; do
    if [ -f "${ROOT}/${module}/${f}" ] || [ -f "${scratch}/${f}" ]; then
      if ! diff -q "${ROOT}/${module}/${f}" "${scratch}/${f}" >/dev/null 2>&1; then
        printf '  %s/%s is not tidy\n' "$module" "$f" >&2
        failures=$((failures + 1))
      fi
    fi
  done
done < <("${ROOT}/scripts/list-modules.sh")

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d untidy file(s). Run `make tidy` and review the diff.\n' "$failures" >&2
  exit 1
fi

printf 'OK: %d module(s) tidy.\n' "$checked"
