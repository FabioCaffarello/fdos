#!/usr/bin/env bash
#
# Records what a CI run actually ran on, so a slow run can be told from a
# degrading one.
#
# Measured over twenty completed runs, the gate ranges from 87s to 279s — a 3.2x
# spread, bimodal, with sixteen runs under 135s and three over 220s. Nothing
# recorded which mode a run was in. The plausible cause is `setup-go` restoring
# the module and build caches or not, and *plausible* was as far as the evidence
# could go. This is what turns that inference into a fact, one run at a time.
#
#   make ci-summary
#
# Locally it prints to stdout and is a no-op worth nothing. On a runner it
# appends to the job summary, which is where the attribution has to live: the
# environment is gone by the time anyone reads the duration.
#
# Enforcement ladder position: none. Reporting only.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

cache_state="${TOOLCHAIN_CACHE_HIT:-unknown}"
case "$cache_state" in
  true)  cache_label="hit" ;;
  false) cache_label="miss" ;;
  *)     cache_label="unknown (not running under the toolchain action)" ;;
esac

go_version="$(scripts/tool-version.sh go 2>/dev/null || printf 'unresolved')"
go_actual="$(go version 2>/dev/null | awk '{print $3}' || printf 'absent')"

report() {
  printf 'Build cache:   %s\n' "$cache_label"
  printf 'Runner:        %s %s\n' "${RUNNER_OS:-$(uname -s)}" "${RUNNER_ARCH:-$(uname -m)}"
  printf 'Go pinned:     %s\n' "$go_version"
  printf 'Go present:    %s\n' "$go_actual"
  printf 'Commit:        %s\n' "$(git rev-parse --short HEAD)"
}

report

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    printf '### Run environment\n\n'
    printf '| | |\n|---|---|\n'
    printf '| Build cache | **%s** |\n' "$cache_label"
    printf '| Runner | %s %s |\n' "${RUNNER_OS:-unknown}" "${RUNNER_ARCH:-unknown}"
    printf '| Go pinned | `%s` |\n' "$go_version"
    printf '| Go present | `%s` |\n' "$go_actual"
    printf '| Commit | `%s` |\n' "$(git rev-parse --short HEAD)"
  } >> "$GITHUB_STEP_SUMMARY"
fi
