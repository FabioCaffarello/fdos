#!/usr/bin/env bash
#
# Run the gate as it will run *after* the tag exists, without creating one.
#
#   make release-simulate MODULE=libs/ledger-wire VERSION=v0.5.0
#
# # Why this exists
#
# `release.yml` re-runs `make verify` on the tagged commit, so the gate's answer
# has to be the same before and after the tag. It was not: `pin-check` read the
# tag namespace, and the first release cut through the dispatched path failed on
# the tag it was publishing (#125). That tag is permanent and its release can
# never complete.
#
# The fix made `pin-check` a function of the commit. This is what proves it,
# per release, before anything is published — it creates the tag locally, runs
# the whole gate, and deletes the tag whether the gate passed or not.
#
# It is deliberately local. `release-rehearse.yml` checks out the *tag's* tree,
# so it cannot rehearse a path the tag does not yet contain — which is why no
# published tag could be rehearsed against at all (#124). This runs the current
# tree's gate against a hypothetical tag, which is the combination that matters.
#
# Enforcement ladder position: none. It is the dry run for the gate, as
# `release-tag` without PUBLISH is the dry run for the publication.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODULE="${MODULE:-${1:-}}"
VERSION="${VERSION:-${2:-}}"

if [ -z "$MODULE" ] || [ -z "$VERSION" ]; then
  printf 'usage: make release-simulate MODULE=<path> VERSION=vX.Y.Z\n' >&2
  exit 2
fi

TAG="${MODULE}/${VERSION}"

if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null 2>&1; then
  printf 'release-simulate: %s already exists; there is nothing to simulate\n' "$TAG" >&2
  exit 1
fi

# Deleted on every exit path. A simulation tag left behind would be pushed by
# the next `git push --tags`, which is how a rehearsal becomes a release.
cleanup() {
  git tag -d "$TAG" >/dev/null 2>&1 || true
}
trap cleanup EXIT

printf 'Simulating %s: the gate as it will run once the tag exists.\n\n' "$TAG"

git tag "$TAG"

if make verify; then
  printf '\nThe gate is green with %s present.\n' "$TAG"
  printf 'The tag has been deleted; nothing was published.\n'
  exit 0
fi

printf '\nFAIL: the gate is red with %s present.\n' "$TAG" >&2
printf 'Publishing it would create a permanent tag whose release cannot complete,\n' >&2
printf 'because release.yml re-verifies the tagged commit and checks out its tree.\n' >&2
printf 'That is #125, and this is the check that exists so it does not recur.\n' >&2
exit 1
