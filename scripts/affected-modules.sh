#!/usr/bin/env bash
#
# Print the modules affected by a change, one per line.
#
# ADR-0004 chose Make over Nx, accepting the loss of affected-graph builds. This
# is the compensation: `go list -deps` intersected with the git diff recovers
# most of the benefit with no added tooling.
#
#   scripts/affected-modules.sh [base-ref]     # default: origin/main, else main
#
# A change to shared infrastructure — the Makefile, any script, a workflow, the
# toolchain pins — affects every module. That is deliberately conservative:
# under-reporting means a broken module ships, while over-reporting only costs
# time. When a change cannot be classified, everything is affected.
#
# Correctness is never gated on this. `make verify` always runs the full set;
# this exists for fast local feedback and for pull-request triage.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

GO="${GO:-go}"

base="${1:-}"
if [ -z "$base" ]; then
  if git rev-parse --verify --quiet origin/main >/dev/null 2>&1; then
    base="origin/main"
  elif git rev-parse --verify --quiet main >/dev/null 2>&1; then
    base="main"
  else
    base="HEAD"
  fi
fi

all_modules="$(./scripts/list-modules.sh)"

# Uncommitted work counts: the point is feedback before pushing.
changed="$(
  {
    if [ "$base" = "HEAD" ]; then
      git diff --name-only HEAD 2>/dev/null || true
    else
      git diff --name-only "${base}...HEAD" 2>/dev/null || true
      git diff --name-only HEAD 2>/dev/null || true
    fi
    git ls-files --others --exclude-standard 2>/dev/null || true
  } | sort -u
)"

[ -n "$changed" ] || exit 0

# Anything shared invalidates the whole set.
if printf '%s\n' "$changed" | grep -qE '^(Makefile|mise\.toml|go\.work|\.golangci\.yaml|lefthook\.yml|\.gitleaks\.toml|scripts/|\.github/)'; then
  printf '%s\n' "$all_modules"
  exit 0
fi

directly_changed=""
while IFS= read -r module; do
  [ -n "$module" ] || continue
  if printf '%s\n' "$changed" | grep -q "^${module}/"; then
    directly_changed="${directly_changed}${module}"$'\n'
  fi
done <<EOF
$all_modules
EOF

[ -n "${directly_changed// /}" ] || exit 0

# A module that depends on a changed module is affected too. Resolution runs
# with GOWORK=off so the dependency graph is the published one (ADR-0004).
affected="$directly_changed"
while IFS= read -r module; do
  [ -n "$module" ] || continue
  printf '%s\n' "$directly_changed" | grep -qx "$module" && continue

  deps="$(cd "$module" && GOWORK=off "$GO" list -deps -f '{{if .Module}}{{.Module.Path}}{{end}}' ./... 2>/dev/null | sort -u || true)"
  [ -n "$deps" ] || continue

  while IFS= read -r changed_module; do
    [ -n "$changed_module" ] || continue
    changed_path="$(cd "$changed_module" && GOWORK=off "$GO" list -m 2>/dev/null || true)"
    [ -n "$changed_path" ] || continue
    if printf '%s\n' "$deps" | grep -qx "$changed_path"; then
      affected="${affected}${module}"$'\n'
      break
    fi
  done <<INNER
$directly_changed
INNER
done <<EOF
$all_modules
EOF

printf '%s' "$affected" | sed '/^$/d' | sort -u
