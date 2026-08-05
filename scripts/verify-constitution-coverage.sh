#!/usr/bin/env bash
#
# Fitness function: the enforcement table cannot go stale.
#
# ADR-0005 requires every Constitution principle to declare the ladder rung it
# currently sits at, recorded in the §15 "Current position" table. That table is
# the repository's honest self-assessment of how much of its own architecture is
# actually enforced.
#
# Nothing prevented a principle from being added, renamed or removed without the
# table following — which would quietly turn the one artifact measuring
# architectural erosion into a source of false confidence.
#
# This check asserts the table lists every principle, by number and by name.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
CONSTITUTION="${ROOT}/docs/constitution.md"

# The section holding the table is itself a numbered section but is not a
# principle — it is the mechanism. It is excluded by title.
LADDER_SECTION="The Enforcement Ladder"

failures=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying Constitution enforcement coverage...\n'

if [ ! -f "$CONSTITUTION" ]; then
  printf '  docs/constitution.md: missing\n' >&2
  exit 1
fi

# Principles: "## N. Title"
principles="$(awk -v skip="$LADDER_SECTION" '
  /^## [0-9]+\. / {
    line = $0
    sub(/^## /, "", line)
    num = line; sub(/\..*$/, "", num)
    title = line; sub(/^[0-9]+\.[[:space:]]*/, "", title)
    if (title == skip) next
    print num "\t" title
  }
' "$CONSTITUTION")"

# Table rows: "| N | Principle | mechanism | rung | target |"
rows="$(awk '
  /^### Current position/ { intable = 1; next }
  !intable { next }
  /^\|[[:space:]]*[0-9]+[[:space:]]*\|/ {
    n = split($0, cell, "|")
    num = cell[2]; gsub(/^[[:space:]]+|[[:space:]]+$/, "", num)
    title = cell[3]; gsub(/^[[:space:]]+|[[:space:]]+$/, "", title)
    print num "\t" title
  }
' "$CONSTITUTION")"

if [ -z "$principles" ]; then
  fail "docs/constitution.md: no numbered principles found"
fi

if [ -z "$rows" ]; then
  fail "docs/constitution.md: §15 'Current position' table not found or empty"
fi

principle_count=0
while IFS="$(printf '\t')" read -r num title; do
  [ -n "$num" ] || continue
  principle_count=$((principle_count + 1))
  row_title="$(printf '%s\n' "$rows" | awk -F'\t' -v n="$num" '$1 == n { print $2; exit }')"
  if [ -z "$row_title" ]; then
    fail "principle ${num} '${title}' is missing from the §15 enforcement table"
  elif [ "$row_title" != "$title" ]; then
    fail "principle ${num}: heading says '${title}', §15 table says '${row_title}'"
  fi
done <<EOF
$principles
EOF

# The reverse direction: a table row for a principle that no longer exists.
while IFS="$(printf '\t')" read -r num title; do
  [ -n "$num" ] || continue
  if ! printf '%s\n' "$principles" | awk -F'\t' -v n="$num" '$1 == n { found = 1 } END { exit !found }'; then
    fail "§15 table lists principle ${num} '${title}', which has no corresponding section"
  fi
done <<EOF
$rows
EOF

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d enforcement-table violation(s).\n' "$failures" >&2
  exit 1
fi

printf 'OK: all %d principles present in the §15 enforcement table.\n' "$principle_count"
