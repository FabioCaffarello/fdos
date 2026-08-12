#!/usr/bin/env bash
#
# Report which pinned GitHub Actions have moved on. Never update one.
#
#   make action-freshness
#
# ADR-0014 pinned every action to a commit SHA and accepted the cost in writing:
#
#   "Pinned actions do not receive security fixes automatically. This is the
#    real cost, and it is not small: an unpatched action stays unpatched until
#    someone re-resolves the SHA. The mitigation is the scheduled supply-chain
#    workflow plus deliberate review."
#
# The scheduled workflow exists and checks the freshness of nothing. This is the
# mitigation that was named and never built.
#
# **It reports and never applies.** Not a pull request, not a merge — an issue a
# person reads. Dependabot and Renovate would close this gap with less code and
# contradict ADR-0014 head-on, where an input that can change without a reviewed
# commit is the whole thing being defended against. Taking the reporting half and
# leaving the applying half to a human is the only version compatible with the
# decision that is already made.
#
# `gh` is required and is not in mise.toml, following the precedent ADR-0014
# recorded for syft and cosign: a tool that only ever talks to the forge is not
# developer toolchain.
#
# Enforcement ladder position: none. Reporting only, deliberately.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

if ! command -v gh >/dev/null 2>&1; then
  printf 'action-freshness: gh is not installed\n' >&2
  printf '  This reads the GitHub API and has no offline mode.\n' >&2
  exit 2
fi

printf 'Checking pinned actions against upstream...\n\n'

behind=0
checked=0
report=""

# `uses: owner/repo@sha # vX.Y.Z` across workflows and local composite actions.
# Local `./.github/actions/...` references have no upstream and are skipped.
refs="$(
  grep -rhoE 'uses:[[:space:]]+[A-Za-z0-9._-]+/[A-Za-z0-9._-]+@[0-9a-f]{40}[[:space:]]*#[[:space:]]*[^[:space:]]+' .github \
    | sed -E 's/^uses:[[:space:]]+//; s/[[:space:]]*#[[:space:]]*/ /' \
    | sort -u
)"

while IFS= read -r line; do
  [ -n "$line" ] || continue

  ref="${line%% *}"
  claimed="${line##* }"
  repo="${ref%@*}"
  pinned="${ref#*@}"
  checked=$((checked + 1))

  latest="$(gh api "repos/${repo}/releases/latest" --jq .tag_name 2>/dev/null || true)"
  if [ -z "$latest" ]; then
    # Not every action publishes releases; fall back to the newest tag.
    latest="$(gh api "repos/${repo}/tags?per_page=1" --jq '.[0].name' 2>/dev/null || true)"
  fi

  if [ -z "$latest" ]; then
    printf '  %-52s %s  (upstream unreadable)\n' "$repo" "$claimed"
    continue
  fi

  if [ "$latest" = "$claimed" ]; then
    printf '  %-52s %s  current\n' "$repo" "$claimed"
    continue
  fi

  latest_sha="$(gh api "repos/${repo}/git/ref/tags/${latest}" --jq .object.sha 2>/dev/null || true)"

  # A moved tag pointing at the SHA already pinned is not an upgrade. Reporting
  # it as one would train the reader to ignore this list.
  if [ -n "$latest_sha" ] && [ "$latest_sha" = "$pinned" ]; then
    printf '  %-52s %s  current (upstream retagged as %s)\n' "$repo" "$claimed" "$latest"
    continue
  fi

  behind=$((behind + 1))
  printf '  %-52s %s -> %s  BEHIND\n' "$repo" "$claimed" "$latest"
  report="${report}- \`${repo}\` — pinned \`${claimed}\`, latest \`${latest}\`"$'\n'
  if [ -n "$latest_sha" ]; then
    report="${report}  \`\`\`
  uses: ${repo}@${latest_sha} # ${latest}
  \`\`\`"$'\n'
  fi
done <<EOF
$refs
EOF

printf '\n%d action(s) checked, %d behind.\n' "$checked" "$behind"

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    printf '### Pinned action freshness\n\n'
    printf '%d checked, **%d behind**.\n\n' "$checked" "$behind"
    [ -n "$report" ] && printf '%s\n' "$report"
  } >> "$GITHUB_STEP_SUMMARY"
fi

# The issue is the deliverable: a person decides, and the decision is a reviewed
# commit. FRESHNESS_ISSUE names a long-lived issue so weekly reports accumulate
# in one place rather than creating one issue per week nobody closes.
if [ -n "${FRESHNESS_ISSUE:-}" ] && [ "$behind" -gt 0 ]; then
  gh issue comment "$FRESHNESS_ISSUE" --body "$(printf '**%s** — %d of %d pinned actions are behind.\n\n%s\nUpdating one is a reviewed commit, never an automatic merge (ADR-0014).' \
    "$(date -u +%Y-%m-%d)" "$behind" "$checked" "$report")"
  printf 'Posted to issue #%s.\n' "$FRESHNESS_ISSUE"
fi

exit 0
