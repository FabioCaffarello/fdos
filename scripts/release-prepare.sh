#!/usr/bin/env bash
#
# Prepare the commit a release tag will point at.
#
#   make release-prepare MODULE=libs/kernel VERSION=v0.10.0
#
# It does one mechanical thing: set the module's row in the contract registry to
# the version about to be released. That row belongs in the tagged commit, so
# that whoever checks out the tag reads a table describing it — and it is what
# `make release-tag` refuses to proceed without.
#
# It does **not** choose the version, bump pins, commit, or open a pull request
# from a script. Pins are already held current by `make pin-check` R3 for any
# module with unreleased changes, so by the time a release is being prepared they
# are correct or the gate is red. Adding a second mechanism to bump them would be
# a second opinion about the same invariant.
#
# The result is a working-tree edit for a person to review, commit and push
# through the normal pull-request path (ADR-0020). Nothing here merges anything.
#
# Enforcement ladder position: none. A preparation aid.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODULE="${MODULE:-${1:-}}"
VERSION="${VERSION:-${2:-}}"
REGISTRY="docs/ecosystem/contracts.md"

die() {
  printf 'release-prepare: %s\n' "$1" >&2
  exit 1
}

if [ -z "$MODULE" ] || [ -z "$VERSION" ]; then
  printf 'usage: make release-prepare MODULE=<path> VERSION=vX.Y.Z\n' >&2
  exit 2
fi

printf '%s' "$VERSION" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$' \
  || die "version '${VERSION}' is not vX.Y.Z"

scripts/list-modules.sh | grep -qx "$MODULE" \
  || die "'${MODULE}' is not a module in this repository"

current_row="$(grep -nE "^\| \`${MODULE}\` \| \`v[0-9]+\.[0-9]+\.[0-9]+\` \|" "$REGISTRY" || true)"
if [ -z "$current_row" ]; then
  die "${MODULE} has no row in ${REGISTRY}; add one rather than letting this guess the columns"
fi

line_no="${current_row%%:*}"
old_version="$(printf '%s' "$current_row" | grep -oE '`v[0-9]+\.[0-9]+\.[0-9]+`' | tail -1 | tr -d '`')"

if [ "$old_version" = "$VERSION" ]; then
  printf 'The registry already declares %s %s. Nothing to prepare.\n' "$MODULE" "$VERSION"
  exit 0
fi

# In place, on one line, matched by module rather than by version — so a table
# that has grown a column still works, and a second module at the same version
# is not touched.
tmp="$(mktemp)"
awk -v n="$line_no" -v old="$old_version" -v new="$VERSION" '
  NR == n { gsub("`" old "`", "`" new "`") }
  { print }
' "$REGISTRY" > "$tmp"
mv "$tmp" "$REGISTRY"

printf 'Registry updated: %s %s -> %s\n\n' "$MODULE" "$old_version" "$VERSION"
printf 'Next, and each step is a person:\n\n'
printf '  1. describe what the version carries, under the table — the row says\n'
printf '     which version, not what it means\n'
printf '  2. make release-simulate MODULE=%s VERSION=%s\n' "$MODULE" "$VERSION"
printf '     the gate as it will run once the tag exists — #125 is what happens\n'
printf '     when that answer differs from the one before the tag\n'
printf '  3. commit, open a pull request, merge it\n'
printf '  4. make release-tag MODULE=%s VERSION=%s\n' "$MODULE" "$VERSION"
printf '\n`make registry-check` accepts a row above the newest tag only while the\n'
printf 'module has unreleased changes, which is exactly the window this opens.\n'
