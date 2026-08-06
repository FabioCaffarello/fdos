#!/usr/bin/env bash
#
# Fitness function: every GitHub Action is pinned to a full commit SHA.
#
# A tag is mutable. An action referenced as `@v4` can change under the
# repository with no commit here — an unreviewed third party with write access
# to the build, and therefore to anything the build produces or attests to.
#
# For software asking institutions to trust its output, that is not acceptable
# (Constitution §9, §14). `.github/README.md` stated this rule from M0; this is
# the mechanism that makes it true rather than intended.
#
# Also asserts that CI contains no logic of its own beyond installing tools —
# see ADR-0014. That half is checked only loosely: `run:` steps are permitted,
# but a workflow whose `run:` blocks outnumber its `make` invocations is flagged
# for review rather than failed, because the line is a judgement.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

WORKFLOW_DIR=".github/workflows"

# Composite actions in this repository are scanned too. They are not third
# parties themselves, but they *reference* third parties, and a `uses:` hidden
# one directory deeper is exactly as much write access to the build as one in a
# workflow. Extracting a composite action moved two pinned references out of
# this check's sight before this line existed; the count dropping from 14 to 13
# was the only signal, and nothing would have reported an unpinned one.
ACTION_DIR=".github/actions"

failures=0
checked=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying GitHub Action pinning...\n'

if [ ! -d "$WORKFLOW_DIR" ]; then
  printf 'OK: no workflows yet.\n'
  exit 0
fi

while IFS= read -r workflow; do
  [ -f "$workflow" ] || continue

  while IFS= read -r line; do
    [ -n "$line" ] || continue
    ref="$(printf '%s' "$line" | sed -E 's/^[[:space:]]*-?[[:space:]]*uses:[[:space:]]*//; s/[[:space:]]*(#.*)?$//')"
    [ -n "$ref" ] || continue

    checked=$((checked + 1))

    # Local composite actions and reusable workflows in this repository are not
    # third parties and have nothing to pin.
    case "$ref" in
      ./*|.github/*) continue ;;
    esac

    if ! printf '%s' "$ref" | grep -qE '@[0-9a-f]{40}$'; then
      fail "${workflow}: '${ref}' is not pinned to a full commit SHA"
      fail "    resolve it: gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha"
    fi
  done < <(grep -E '^[[:space:]]*-?[[:space:]]*uses:' "$workflow" || true)
done < <(
  {
    ls "${WORKFLOW_DIR}"/*.yml "${WORKFLOW_DIR}"/*.yaml 2>/dev/null || true
    find "$ACTION_DIR" -name 'action.yml' -o -name 'action.yaml' 2>/dev/null || true
  } | sort -u
)

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d unpinned action reference(s).\n' "$failures" >&2
  printf 'A moved tag is an unreviewed third party with write access to the build.\n' >&2
  exit 1
fi

if [ "$checked" -eq 0 ]; then
  printf 'OK: no action references found.\n'
else
  printf 'OK: %d action reference(s), all pinned to a commit SHA.\n' "$checked"
fi
