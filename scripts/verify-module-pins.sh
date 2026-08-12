#!/usr/bin/env bash
#
# Fitness function: a first-party pin names a version that exists, and a module
# under active development pins its siblings at what they actually released.
#
# ADR-0004 makes every libs/* an independent module with its own tag, so a change
# spanning two modules is really two or three coordinated releases. `go.work`
# covers for the incompatibility locally right up until `make verify` runs, which
# is the end of a slice rather than the start — so the chain is discovered at the
# moment it fails rather than the moment it is designed. That happened three
# milestones running (M9 Track A, M10, the M11 gate), each time found by hand.
#
# Three blocking rules and one report, and the split is measured rather than
# cautious.
#
#   R1  a first-party pin names a tag that exists                      fails
#   R2  a pin is not newer than the newest tag                         fails
#   R3  a module with unreleased changes pins siblings at their newest  fails
#   R4  an unreleased-clean module pinning behind                      reports
#
# **Why R4 does not fail.** Measured across the tree: thirteen pins in five
# modules were behind their dependency's newest tag, and none of them was a
# defect — a module legitimately stays on the version it was released against
# until someone bumps it. Making that blocking would turn `main` red the instant
# any module is tagged, and would make the release sequence ADR-0041 documents
# ("this release is deliberately published into a tree that does not build as a
# workspace… the next release closes it") impossible to perform.
#
# **Why R3 does fail.** It fires only on modules with changes not yet released —
# the ones somebody is working on. Tagging a dependency does not redden the gate;
# editing a module whose pins are stale does. That is the case #65 recorded three
# times, and it is the one where a stale pin is about to cost a rediscovered
# release chain.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODULE_PREFIX="github.com/FabioCaffarello/fdos/"
failures=0
behind_report=""

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

# Newest vX.Y.Z tag for a module path, or empty. Pre-release and metadata
# suffixes are deliberately not ranked: this repository releases X.Y.Z, and a
# comparison that silently mis-orders `v0.1.0-rc.1` is worse than one that
# declines to consider it.
#
# `|| true` is load-bearing: with `set -euo pipefail`, the grep in this pipeline
# returns 1 for a module that has never been tagged, and the exit status of a
# command substitution is the exit status of the assignment — so an untagged
# module killed the script with no output at all. Found by running it against
# `libs/analysis`, which has no tags.
newest_tag() {
  git tag --list "${1}/v*" 2>/dev/null \
    | sed "s|^${1}/v||" \
    | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -t. -k1,1n -k2,2n -k3,3n \
    | tail -1 || true
}

# 0 when a == b, 1 when a > b, 2 when a < b.
compare_versions() {
  if [ "$1" = "$2" ]; then return 0; fi
  higher="$(printf '%s\n%s\n' "$1" "$2" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)"
  if [ "$higher" = "$1" ]; then return 1; fi
  return 2
}

printf 'Verifying first-party module pins...\n'

if [ -z "$(git tag --list 'libs/*/v*' | head -1)" ]; then
  printf '\nFAIL: no release tags are present.\n' >&2
  printf 'This check compares pins against published tags and cannot run on a\n' >&2
  printf 'shallow clone. Fetch tags (actions/checkout with fetch-depth: 0).\n' >&2
  exit 1
fi

modules="$(scripts/list-modules.sh)"

while IFS= read -r module; do
  [ -n "$module" ] || continue
  [ -f "${module}/go.mod" ] || continue

  own_tag="$(newest_tag "$module")"

  # A module with no tag has never been released, so everything in it is
  # unreleased by definition. `apps/*` and `examples/*` are that case and stay
  # that case; nothing imports them, and R3 holding there costs nothing.
  if [ -z "$own_tag" ]; then
    unreleased=true
    state="never released"
  elif [ -n "$(git diff --name-only "${module}/v${own_tag}" -- "$module" 2>/dev/null)" ]; then
    unreleased=true
    state="changed since v${own_tag}"
  else
    unreleased=false
    state="released at v${own_tag}"
  fi

  while IFS= read -r requirement; do
    [ -n "$requirement" ] || continue

    dep_path="$(printf '%s' "$requirement" | awk '{ print $1 }')"
    dep_version="$(printf '%s' "$requirement" | awk '{ print $2 }')"
    indirect="$(printf '%s' "$requirement" | awk '{ print $3 }')"
    dep="${dep_path#${MODULE_PREFIX}}"
    pinned="${dep_version#v}"

    # R1 — the pin names a tag that exists. A pseudo-version fails here, which
    # is the point: it means the dependency was resolved off a branch rather
    # than off a release, and the M11 gate found exactly that.
    if ! git rev-parse -q --verify "refs/tags/${dep}/${dep_version}^{}" >/dev/null 2>&1 \
       && ! git rev-parse -q --verify "refs/tags/${dep}/${dep_version}" >/dev/null 2>&1; then
      fail "${module}: pins ${dep} ${dep_version}, which is not a published tag"
      continue
    fi

    # An indirect requirement's version is chosen by minimal version selection
    # across the whole graph, not by whoever wrote this go.mod. Demanding it be
    # the newest tag would demand a resolution Go may not produce, so R2, R3 and
    # R4 stop at direct requirements. R1 above does not: an indirect pin naming a
    # tag that does not exist is still a build resolved off nothing.
    if [ "$indirect" = "//" ]; then
      continue
    fi

    dep_newest="$(newest_tag "$dep")"
    [ -n "$dep_newest" ] || continue

    cmp=0
    compare_versions "$pinned" "$dep_newest" || cmp=$?

    if [ "$cmp" -eq 0 ]; then
      continue # current
    elif [ "$cmp" -eq 1 ]; then
      # R2 — pinned above the newest tag. Reachable when a tag is deleted, which
      # the release-tags ruleset now forbids, and when a pin is hand-edited.
      fail "${module}: pins ${dep} v${pinned}, above its newest tag v${dep_newest}"
      continue
    fi

    if [ "$unreleased" = true ]; then
      # R3
      fail "${module} (${state}): pins ${dep} v${pinned}, but v${dep_newest} is published"
      fail "  a module being changed must pin what its siblings actually released,"
      fail "  or the release chain is discovered when the gate fails instead of when it is designed"
    else
      # R4
      behind_report="${behind_report}  ${module} (${state}): ${dep} v${pinned} -> v${dep_newest}"$'\n'
    fi
  done < <(
    grep -E "^[[:space:]]*${MODULE_PREFIX//./\\.}" "${module}/go.mod" 2>/dev/null \
      | grep -v '^module' \
      | awk 'NF >= 2 { print $1, $2, $3 }'
  )
done <<EOF
$modules
EOF

if [ -n "$behind_report" ]; then
  printf '\nBehind, and not a failure — these are the release chain, not a defect:\n'
  printf '%s' "$behind_report"
  printf 'Each is a released module still pinning what it was released against.\n'
fi

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d pin violation(s).\n' "$failures" >&2
  exit 1
fi

printf '\nOK: every first-party pin names a published version.\n'
