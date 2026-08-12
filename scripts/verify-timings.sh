#!/usr/bin/env bash
#
# Reports what the gate costs, per check.
#
# This enforces nothing. It is here because nothing in this repository could say
# where `make verify` spends its time, and a plan to make the gate faster or
# larger without that number is a guess. RFC-0018 §Motivation had to be produced
# by a throwaway script written for the occasion, which was the finding.
#
#   make verify-timings
#
# The target list is passed in by the Makefile from $(VERIFY_TARGETS) — the same
# variable `verify` expands. There is deliberately no second list here: a copy of
# what the gate contains is the copy that drifts, and B-008 is what that costs.
#
# It runs the checks, so a failure fails this too. A timing report that passes
# while the gate is red would be a stopwatch on a broken clock.
#
# Enforcement ladder position: none. Reporting only.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

if [ "$#" -eq 0 ]; then
  printf 'verify-timings: no targets given — invoke through `make verify-timings`\n' >&2
  exit 2
fi

# Bash's own `time`, because `date +%s%N` has no %N on macOS and the scripts here
# run on bash 3.2. TIMEFORMAT='%3R' gives wall-clock milliseconds on both.
TIMEFORMAT='%3R'

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

results="${work}/results"
: > "$results"

failures=0

printf 'Running the gate with a stopwatch (%d checks)...\n\n' "$#"

for target in "$@"; do
  log="${work}/${target}.log"
  timing="${work}/${target}.time"

  { time { if ${MAKE:-make} "$target" >"$log" 2>&1; then rc=0; else rc=$?; fi; }; } 2>"$timing"

  elapsed="$(cat "$timing")"
  if [ "$rc" -eq 0 ]; then
    state=ok
  else
    state=FAIL
    failures=$((failures + 1))
  fi

  printf '%s %s %s\n' "$elapsed" "$target" "$state" >> "$results"
  printf '  %-28s %8ss  %s\n' "$target" "$elapsed" "$state"

  # A failing check is the interesting one. Its output is worth more than the
  # timing, so it is not swallowed to keep the table tidy.
  if [ "$rc" -ne 0 ]; then
    printf '\n--- %s ---\n' "$target" >&2
    cat "$log" >&2
    printf -- '--- end %s ---\n\n' "$target" >&2
  fi
done

total="$(awk '{ sum += $1 } END { printf "%.3f", sum }' "$results")"

printf '\nSlowest first:\n\n'
sort -rn "$results" | awk '{ printf "  %-28s %8ss  %s\n", $2, $1, $3 }'
printf '\n  %-28s %8ss\n' "TOTAL" "$total"

# On a runner, the same table goes to the job summary. `verify.yml` calls this
# through make and holds no logic of its own (ADR-0014).
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    printf '### Gate timings\n\n'
    printf '| Check | Seconds | Result |\n'
    printf '|---|---:|---|\n'
    sort -rn "$results" | awk '{ printf "| `%s` | %s | %s |\n", $2, $1, $3 }'
    printf '| **TOTAL** | **%s** | |\n' "$total"
  } >> "$GITHUB_STEP_SUMMARY"
fi

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d of %d check(s) failed.\n' "$failures" "$#" >&2
  exit 1
fi

printf '\nAll checks passed.\n'
