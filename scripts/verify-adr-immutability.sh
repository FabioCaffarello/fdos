#!/usr/bin/env bash
#
# Fitness function: accepted ADRs are not rewritten.
#
# ADR-0000 makes the decision log append-only and immutable, for the same reason
# the ledger is: a reversal that leaves no trace reads as though the reversed
# choice was never made. Until now that was enforced by review alone — rung 6,
# and recorded as such in Constitution §15.
#
# This compares every ADR against its content in the commit that introduced it.
# The only permitted differences are the supersession metadata and added lines
# (the banner pointing at the successor). Any other removal or rewrite of an
# existing line is tampering with the record.
#
# Requires full history: CI must checkout with fetch-depth: 0.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

ADR_DIR="docs/adr"

failures=0
checked=0
skipped=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

# Lines whose modification is part of the supersession protocol rather than a
# rewrite of the decision.
is_permitted_removal() {
  case "$1" in
    status:*|superseded_by:*|supersedes:*) return 0 ;;
    "  - ADR-"*) return 0 ;;
    "") return 0 ;;
  esac
  return 1
}

printf 'Verifying ADR immutability...\n'

for adr in "${ADR_DIR}"/[0-9][0-9][0-9][0-9]-*.md; do
  [ -f "$adr" ] || continue
  name="$(basename "$adr")"

  # The commit that introduced the file. --follow survives renames.
  first="$(git log --follow --diff-filter=A --format=%H -- "$adr" 2>/dev/null | tail -1 || true)"
  if [ -z "$first" ]; then
    skipped=$((skipped + 1))
    continue
  fi

  checked=$((checked + 1))

  original="$(git show "${first}:${adr}" 2>/dev/null || true)"
  if [ -z "$original" ]; then
    skipped=$((skipped + 1))
    continue
  fi

  # Lines present in the original and absent now: the only kind of change that
  # can destroy the record. Added lines are always fine.
  removed="$(diff <(printf '%s\n' "$original") "$adr" | sed -n 's/^< //p' || true)"

  while IFS= read -r line; do
    [ -n "$line" ] || continue
    if ! is_permitted_removal "$line"; then
      fail "${adr}: line removed or rewritten since ${first:0:8} — accepted ADRs are superseded, never edited (ADR-0000)"
      fail "    removed: ${line}"
      break
    fi
  done <<EOF
$removed
EOF
done

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: the decision log has been rewritten.\n' >&2
  printf 'To reverse a decision, write a new ADR that supersedes it.\n' >&2
  exit 1
fi

if [ "$skipped" -gt 0 ]; then
  printf 'OK: %d ADRs unmodified since introduction (%d not yet committed).\n' "$checked" "$skipped"
else
  printf 'OK: %d ADRs unmodified since introduction.\n' "$checked"
fi
