#!/usr/bin/env bash
#
# Reports how the gate has been behaving: duration percentiles and failure rate
# over recent runs of `verify`.
#
# One number is not a measurement. RFC-0018's first draft quoted a single 103s
# run as though it were the gate's cost; it was the second-fastest of twelve, and
# the distribution turned out to be bimodal with a 279s tail. This exists so the
# next such claim is a distribution rather than whichever run someone looked at.
#
#   make ci-stats            # last 30 runs
#   make ci-stats LIMIT=100
#
# `gh` is required and is NOT in mise.toml, deliberately: this reads the forge's
# API and cannot run without one, so it is CI-and-maintainer only. That follows
# the precedent ADR-0014 recorded for syft and cosign — tools that run only in CI
# are not pinned as developer toolchain. It fails loudly when gh is absent rather
# than reporting an empty distribution, which would read as a healthy pipeline.
#
# Enforcement ladder position: none. Reporting only.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

LIMIT="${1:-30}"
WORKFLOW="${WORKFLOW:-verify.yml}"

if ! command -v gh >/dev/null 2>&1; then
  printf 'ci-run-stats: gh is not installed\n' >&2
  printf '  This reads the GitHub API and has no offline mode. Install gh, or\n' >&2
  printf '  read the scheduled run of this script in the Actions job summary.\n' >&2
  exit 2
fi

runs="$(mktemp)"
trap 'rm -f "$runs"' EXIT

# created_at to updated_at is the whole run, including queue time — which is what
# a person waiting on the gate actually experiences, and is not the same as the
# `make verify` step's duration.
gh run list \
  --workflow "$WORKFLOW" \
  --limit "$LIMIT" \
  --json conclusion,createdAt,updatedAt,event \
  --jq '.[] | select(.conclusion != "" and .conclusion != null)
        | [.conclusion, .createdAt, .updatedAt, .event] | @tsv' > "$runs"

if [ ! -s "$runs" ]; then
  printf 'ci-run-stats: no completed runs of %s found\n' "$WORKFLOW" >&2
  exit 1
fi

# Durations in seconds, successes only: a failed run stops early and its duration
# describes the failure rather than the gate.
report="$(
  awk -F'\t' '
    # Days since 1970-01-01 from a civil date, by arithmetic.
    #
    # `mktime` would be one line and is a gawk extension: macOS ships the one
    # true awk, which does not have it, and these scripts run on both. Found by
    # running it — "calling undefined function mktime" — rather than by reading
    # a compatibility table.
    function days_from_civil(y, m, d,   era, yoe, doy, doe) {
      y += 0; m += 0; d += 0
      if (m <= 2) y -= 1
      era = int((y >= 0 ? y : y - 399) / 400)
      yoe = y - era * 400
      doy = int((153 * (m + (m > 2 ? -3 : 9)) + 2) / 5) + d - 1
      doe = yoe * 365 + int(yoe / 4) - int(yoe / 100) + doy
      return era * 146097 + doe - 719468
    }

    function to_epoch(ts,   y, mo, d, h, mi, s) {
      # ISO-8601 Z, e.g. 2026-08-12T20:16:06Z.
      y  = substr(ts, 1, 4);  mo = substr(ts, 6, 2);  d = substr(ts, 9, 2)
      h  = substr(ts, 12, 2); mi = substr(ts, 15, 2); s = substr(ts, 18, 2)
      return days_from_civil(y, mo, d) * 86400 + (h + 0) * 3600 + (mi + 0) * 60 + (s + 0)
    }
    {
      total++
      if ($1 != "success") { failed++; next }
      d[++n] = to_epoch($3) - to_epoch($2)
    }
    END {
      if (n == 0) { print "no successful runs"; exit }
      # Insertion sort: n is tens, and asort() is a gawk extension the runners
      # may not have.
      for (i = 2; i <= n; i++) {
        v = d[i]; j = i - 1
        while (j > 0 && d[j] > v) { d[j+1] = d[j]; j-- }
        d[j+1] = v
      }
      p50 = d[int((n + 1) * 0.50 + 0.5)]
      p95 = d[int((n + 1) * 0.95 + 0.5)]
      if (p95 == "") p95 = d[n]
      printf "runs=%d successful=%d failed=%d fail_rate=%.1f%%\n", total, n, failed+0, (failed+0) * 100.0 / total
      printf "min=%ds p50=%ds p95=%ds max=%ds spread=%.1fx\n", d[1], p50, p95, d[n], d[n] / (d[1] > 0 ? d[1] : 1)
      printf "durations="
      for (i = 1; i <= n; i++) printf "%s%d", (i > 1 ? " " : ""), d[i]
      printf "\n"
    }
  ' "$runs"
)"

printf 'Gate statistics — %s, last %s runs\n\n' "$WORKFLOW" "$LIMIT"
printf '%s\n' "$report"

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    printf '### Gate statistics — last %s runs of `%s`\n\n' "$LIMIT" "$WORKFLOW"
    printf '```\n%s\n```\n\n' "$report"
    printf 'Durations are wall-clock from queue to completion, successes only: a\n'
    printf 'failed run stops early and describes the failure rather than the gate.\n'
  } >> "$GITHUB_STEP_SUMMARY"
fi

# A tracking issue turns weekly snapshots into a trend somebody can read in one
# place. Without one this reports into a job summary that expires with the run.
if [ -n "${STATS_ISSUE:-}" ]; then
  gh issue comment "$STATS_ISSUE" --body "$(printf '**%s** — last %s runs of `%s`\n\n```\n%s\n```' \
    "$(date -u +%Y-%m-%d)" "$LIMIT" "$WORKFLOW" "$report")"
  printf '\nPosted to issue #%s.\n' "$STATS_ISSUE"
fi
