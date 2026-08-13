#!/usr/bin/env bash
#
# Create a release tag, after checking everything that makes it safe to.
#
#   make release-tag MODULE=libs/kernel VERSION=v0.10.0            # dry run
#   make release-tag MODULE=libs/kernel VERSION=v0.10.0 PUBLISH=1  # for real
#
# **A dry run is the default and publishing requires saying so.** The dangerous
# path should cost a word; the safe path should cost nothing.
#
# # Why a tag deserves this much checking
#
# A release tag here is immutable by ruleset (ADR-0043) — it cannot be moved or
# deleted. So a tag pushed onto a commit that cannot release is permanent
# garbage, and that is not hypothetical: B-008 is fourteen tags whose release
# failed, still in the namespace, describing releases that were never published.
#
# The tag is also what `release.yml` signs and attests against. Tagging a commit
# whose gate is red produces a signed statement about code that does not pass.
#
# So, in order:
#
#   1. the version is well formed and above the module's newest tag
#   2. the tag does not already exist
#   3. the module actually has unreleased changes
#   4. the registry already declares this version
#   5. the working tree is clean and matches the remote
#   6. the gate is green *for this exact commit*
#
# Rule 6 is the one B-008 would have wanted. Rules 1 to 5 are the ones a person
# gets wrong at the end of a long day.
#
# Enforcement ladder position: none directly — it is a publication tool. What it
# refuses to do is the point.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODULE="${MODULE:-${1:-}}"
VERSION="${VERSION:-${2:-}}"
PUBLISH="${PUBLISH:-}"
REMOTE="${REMOTE:-origin}"

die() {
  printf 'release-tag: %s\n' "$1" >&2
  exit 1
}

if [ -z "$MODULE" ] || [ -z "$VERSION" ]; then
  printf 'usage: make release-tag MODULE=<path> VERSION=vX.Y.Z [PUBLISH=1]\n' >&2
  exit 2
fi

newest_tag() {
  git tag --list "${1}/v*" 2>/dev/null \
    | sed "s|^${1}/v||" \
    | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -t. -k1,1n -k2,2n -k3,3n \
    | tail -1 || true
}

printf 'Release: %s %s\n\n' "$MODULE" "$VERSION"

# --- 1. the version is well formed and moves forward -------------------------

printf '%s' "$VERSION" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$' \
  || die "version '${VERSION}' is not vX.Y.Z"

scripts/list-modules.sh | grep -qx "$MODULE" \
  || die "'${MODULE}' is not a module in this repository"

# `examples/*` is a demonstration kit, not a deliverable. `apps/*` is: ADR-0039
# is accepted and says a tag matching `apps/<name>/vX.Y.Z` produces the same
# evidence a library does. This refused both, because it was written while
# ADR-0039 was still Proposed and Phase 5 accepted it without coming back here —
# so the one job allowed to write a tag refused to write the tag the decision
# authorises.
case "$MODULE" in
  examples/*)
    die "'${MODULE}' is a demonstration kit, not a deliverable"
    ;;
esac

newest="$(newest_tag "$MODULE")"
if [ -n "$newest" ]; then
  higher="$(printf '%s\n%s\n' "${VERSION#v}" "$newest" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)"
  [ "$higher" = "${VERSION#v}" ] && [ "${VERSION#v}" != "$newest" ] \
    || die "${VERSION} does not move forward from v${newest}"
  printf '  ok  moves forward from v%s\n' "$newest"
else
  printf '  ok  first release of this module\n'
fi

# --- 2. the tag does not exist ----------------------------------------------

TAG="${MODULE}/${VERSION}"
if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null 2>&1; then
  die "${TAG} already exists — release tags are immutable and are never reused"
fi
if git ls-remote --exit-code --tags "$REMOTE" "refs/tags/${TAG}" >/dev/null 2>&1; then
  die "${TAG} already exists on ${REMOTE}"
fi
printf '  ok  %s is free\n' "$TAG"

# --- 3. there is something to release ---------------------------------------

if [ -n "$newest" ] && [ -z "$(git diff --name-only "${MODULE}/v${newest}" -- "$MODULE")" ]; then
  die "${MODULE} is identical to v${newest} — a release with no change is a version nobody can explain"
fi
printf '  ok  %s has unreleased changes\n' "$MODULE"

# --- 4. the registry already says so ----------------------------------------
#
# The registry update belongs in the commit being tagged, so that whoever checks
# out the tag reads a table describing it. If it is missing, the release is being
# made from a tree that does not describe itself.

# Only libraries. The registry describes what a consumer may import, and an
# application is not importable — `release-artifacts` says the same thing when it
# skips the module zip for anything outside `libs/`.
case "$MODULE" in
  libs/*)
    if ! grep -qE "^\| \`${MODULE}\` \| \`${VERSION}\` \|" docs/ecosystem/contracts.md; then
      die "docs/ecosystem/contracts.md does not list ${MODULE} at ${VERSION} — update it, merge it, then tag the merged commit"
    fi
    printf '  ok  the registry declares %s\n' "$VERSION"
    ;;
  *)
    printf '  --  not a library; the registry describes what may be imported\n'
    ;;
esac

# --- 5. the tree is clean and matches the remote -----------------------------
#
# "A tag captures the tree at a commit, not your working directory" —
# CONTRIBUTING.md, written after this cost rework.

[ -z "$(git status --porcelain)" ] \
  || die "the working tree is dirty; a tag captures the commit, not your directory"

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
[ "$BRANCH" = "main" ] || die "on '${BRANCH}'; releases are tagged from main"

git fetch "$REMOTE" main --quiet
SHA="$(git rev-parse HEAD)"
[ "$SHA" = "$(git rev-parse "${REMOTE}/main")" ] \
  || die "HEAD is not ${REMOTE}/main — pull first, then tag what everyone else has"
printf '  ok  clean tree at %s on main\n' "${SHA:0:8}"

# --- 6. the gate is green for this commit ------------------------------------

if command -v gh >/dev/null 2>&1; then
  conclusion="$(
    gh api "repos/{owner}/{repo}/commits/${SHA}/check-runs" \
      --jq '[.check_runs[] | select(.name == "verify")] | first | .conclusion' 2>/dev/null || true
  )"
  case "$conclusion" in
    success) printf '  ok  verify is green for %s\n' "${SHA:0:8}" ;;
    "") die "no 'verify' check run found for ${SHA} — has CI started?" ;;
    null) die "verify is still running for ${SHA} — wait for it, then dispatch again" ;;
    *) die "verify concluded '${conclusion}' for ${SHA}; tagging it would sign a red commit" ;;
  esac
else
  # Refusing outright would make the tool unusable offline for a maintainer who
  # has just run the gate by hand. Saying nothing would let it look checked.
  printf '  --  gh is absent; the gate could not be confirmed for %s\n' "${SHA:0:8}"
  [ -n "$PUBLISH" ] && die "refusing to publish without confirming the gate; install gh or tag by hand knowingly"
fi

# --- publish -----------------------------------------------------------------

printf '\n'
if [ -z "$PUBLISH" ]; then
  printf 'Dry run. Everything above passed. To publish:\n\n'
  printf '  make release-tag MODULE=%s VERSION=%s PUBLISH=1\n\n' "$MODULE" "$VERSION"
  printf 'That creates %s and pushes it, which starts release.yml.\n' "$TAG"
  exit 0
fi

git tag -a "$TAG" -m "${MODULE} ${VERSION}"
git push "$REMOTE" "refs/tags/${TAG}"

printf 'Published %s at %s.\n' "$TAG" "${SHA:0:8}"
printf 'release.yml is now running: it re-verifies, builds, attests and signs.\n'
printf 'It has failed silently before (B-008) — check that it finished.\n'
