#!/usr/bin/env bash
#
# Print the release chain a change implies, in the order it has to be performed.
#
# "Which modules did this change affect" and "which modules now need a tag" are
# the same question asked twice. This repository answered the first with
# `scripts/affected-modules.sh`, which no automation called, and the second by
# hand — thirty-six tags, and a version chain rediscovered by hand in M9 Track A,
# M10 and the M11 gate ([#65](https://github.com/FabioCaffarello/fdos/issues/65)).
#
#   make release-plan            # against origin/main
#   make release-plan BASE=main
#
# This plans; it does not publish. Choosing a version number is a human
# judgement about compatibility that no script can make, and pushing a tag is a
# publication. So the output names what must happen and in what order, and every
# version it prints for a *new* release is a placeholder the author replaces.
#
# The ordering is the part worth automating. A module must be tagged before a
# dependent can pin it, so the chain is a topological sort of the first-party
# dependency graph restricted to the affected set. Getting that order wrong is
# what costs a second release to correct.
#
# Enforcement ladder position: none. This is a planning aid; `make pin-check`
# and `make registry-check` are what actually hold the invariants.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODULE_PREFIX="github.com/FabioCaffarello/fdos/"
BASE="${1:-}"

newest_tag() {
  git tag --list "${1}/v*" 2>/dev/null \
    | sed "s|^${1}/v||" \
    | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -t. -k1,1n -k2,2n -k3,3n \
    | tail -1 || true
}

# Direct first-party dependencies of a module, as repository-relative paths.
first_party_deps() {
  grep -E "^[[:space:]]*${MODULE_PREFIX//./\\.}" "${1}/go.mod" 2>/dev/null \
    | grep -v '^module' \
    | grep -v '// indirect' \
    | awk -v p="$MODULE_PREFIX" '{ sub(p, "", $1); print $1 }' \
    | sort -u || true
}

affected="$(scripts/affected-modules.sh "$BASE" || true)"

if [ -z "$affected" ]; then
  printf 'Nothing affected. No release chain.\n'
  exit 0
fi

printf 'Release chain for the current change\n'
printf '====================================\n\n'

# --- who needs a tag ---------------------------------------------------------
#
# A module needs one when its own source differs from its newest tag. An
# affected module whose source is untouched does not: it is affected because a
# dependency moved, and what it needs is a pin bump, which is a source change,
# which then makes it need a tag. That cascade is the chain.

needs_tag=""
while IFS= read -r module; do
  [ -n "$module" ] || continue
  own="$(newest_tag "$module")"
  case "$module" in
    apps/*|examples/*)
      # Not published as a module for anyone to pin. It sits at the end of the
      # chain: it consumes tags and produces none.
      continue
      ;;
  esac
  if [ -z "$own" ] || [ -n "$(git diff --name-only "${module}/v${own}" -- "$module" 2>/dev/null)" ]; then
    needs_tag="${needs_tag}${module}"$'\n'
  fi
done <<EOF
$affected
EOF

if [ -z "${needs_tag//[[:space:]]/}" ]; then
  printf 'Affected, and none of it is published as a module:\n'
  printf '%s\n' "$affected" | sed 's/^/  /'
  printf '\nNo tags required.\n'
  exit 0
fi

# --- topological order -------------------------------------------------------
#
# Kahn's algorithm over the affected set. A cycle is impossible in a Go module
# graph — the proxy could not resolve one — so an unemptied worklist means the
# reachability logic is wrong rather than the repository is, and it says so.

remaining="$(printf '%s' "$needs_tag" | sed '/^$/d')"
ordered=""
guard=0

while [ -n "${remaining//[[:space:]]/}" ]; do
  guard=$((guard + 1))
  if [ "$guard" -gt 100 ]; then
    printf 'release-plan: dependency order did not converge — this is a bug in the script,\n' >&2
    printf '  not a cycle in the module graph, which Go could not resolve anyway.\n' >&2
    exit 1
  fi

  progressed=false
  next_remaining=""

  while IFS= read -r module; do
    [ -n "$module" ] || continue
    blocked=false
    while IFS= read -r dep; do
      [ -n "$dep" ] || continue
      if printf '%s\n' "$remaining" | grep -qx "$dep"; then
        blocked=true
        break
      fi
    done < <(first_party_deps "$module")

    if [ "$blocked" = true ]; then
      next_remaining="${next_remaining}${module}"$'\n'
    else
      ordered="${ordered}${module}"$'\n'
      progressed=true
    fi
  done <<INNER
$remaining
INNER

  if [ "$progressed" = false ]; then
    printf 'release-plan: no module became unblocked — reporting the remainder unordered:\n' >&2
    printf '%s\n' "$remaining" | sed 's/^/  /' >&2
    exit 1
  fi

  remaining="$(printf '%s' "$next_remaining" | sed '/^$/d')"
done

# --- the plan ----------------------------------------------------------------

step=0
printf 'Tag in this order. Each step is: bump pins, verify, merge, tag.\n\n'

while IFS= read -r module; do
  [ -n "$module" ] || continue
  step=$((step + 1))
  own="$(newest_tag "$module")"

  if [ -n "$own" ]; then
    printf '%d. %s  (currently v%s)\n' "$step" "$module" "$own"
  else
    printf '%d. %s  (never released)\n' "$step" "$module"
  fi

  # Pins this module must carry before it is tagged: any first-party dependency
  # earlier in this chain will have a new tag by the time we reach here.
  bumps=""
  while IFS= read -r dep; do
    [ -n "$dep" ] || continue
    if printf '%s\n' "$ordered" | grep -qx "$dep"; then
      bumps="${bumps}     bump ${dep} to its new tag (step $(printf '%s\n' "$ordered" | grep -nx "$dep" | cut -d: -f1))"$'\n'
    else
      dep_newest="$(newest_tag "$dep")"
      current="$(grep -E "^[[:space:]]*${MODULE_PREFIX//./\\.}${dep} " "${module}/go.mod" 2>/dev/null | awk '{ print $2 }' | head -1)"
      if [ -n "$dep_newest" ] && [ -n "$current" ] && [ "$current" != "v${dep_newest}" ]; then
        bumps="${bumps}     bump ${dep} ${current} -> v${dep_newest}"$'\n'
      fi
    fi
  done < <(first_party_deps "$module")

  if [ -n "$bumps" ]; then
    printf '%s' "$bumps"
  else
    printf '     no pin changes\n'
  fi
  printf '     git tag %s/vX.Y.Z && git push origin %s/vX.Y.Z\n\n' "$module" "$module"
done <<EOF
$ordered
EOF

# --- consumers that only take ------------------------------------------------

tail_consumers=""
while IFS= read -r module; do
  [ -n "$module" ] || continue
  case "$module" in
    apps/*|examples/*) tail_consumers="${tail_consumers}${module}"$'\n' ;;
  esac
done <<EOF
$affected
EOF

if [ -n "${tail_consumers//[[:space:]]/}" ]; then
  printf 'Then, last, the consumers that publish nothing:\n\n'
  printf '%s' "$tail_consumers" | sed '/^$/d' | sed 's/^/  /'
  printf '\n  They pin the tags above and are not tagged themselves.\n\n'
fi

# --- what the registry will have to say --------------------------------------
#
# Generated rather than remembered, because remembering is what let the table go
# stale for four milestones. It is printed rather than written: a script that
# edits a document containing prose will eventually eat a paragraph.

printf 'Registry rows for docs/ecosystem/contracts.md once the tags exist:\n\n'
while IFS= read -r module; do
  [ -n "$module" ] || continue
  printf '  | `%s` | `vX.Y.Z` | no | no |\n' "$module"
done <<EOF
$ordered
EOF

printf '\n`make registry-check` fails until the table matches the tags.\n'
