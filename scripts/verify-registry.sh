#!/usr/bin/env bash
#
# Fitness function: the contract registry describes the tags that exist.
#
# ADR-0024 calls `docs/ecosystem/contracts.md` "part of the interface" rather
# than documentation about it. It went stale for four milestones anyway — listing
# `libs/kernel` at `v0.5.0` against a pinned `v0.7.0`, and omitting
# `libs/ledger-sqlite` although it was published, tagged and imported — because
# nothing compared it to anything. The document says so itself: "the mechanism is
# rung 6 and this is what rung 6 costs."
#
# Four rules, over the parts of that document a machine can decide:
#
#   G1  a released, unchanged module's row names its newest tag
#   G2  every published module has a row, or is documented in its own section
#   G3  the governance corpus row names the newest ecosystem tag
#   G4  every published libs/contracts tag has a version-history row
#
# G1 applies only to modules whose source matches their newest tag, and that is
# a correction rather than a nicety. Stated as "every row names the newest tag"
# (ADR-0045) it made the release ritual impossible: the registry update lands in
# the commit that gets tagged, so during that pull request the row names a
# version no tag has yet. The rule as written failed there, and would then have
# forced the row to be updated *after* tagging — reddening `main` on every
# release, which is the exact property ADR-0044 refused for `pin-check` R4.
#
# So the split is the same one, for the same reason: check what is settled, and
# let a release in flight declare where it is going. A module with unreleased
# changes may name a version above its newest tag; one without may not.
#
# Deliberately *not* checked: the per-package `version` column in the Published
# table. That column records the version in which a package last changed, which
# is a historical fact no tag can confirm — comparing it to the newest tag would
# make the check demand a wrong answer. Stating the boundary is better than
# quietly checking three of four columns and implying the fourth.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

REGISTRY="docs/ecosystem/contracts.md"
failures=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

# The row to paste. Not a violation of its own — counting the fix as a second
# failure would report four problems where there are three, and a check that
# cannot count is a check nobody trusts about anything else either.
note() {
  printf '  %s\n' "$1" >&2
}

newest_tag() {
  git tag --list "${1}/v*" 2>/dev/null \
    | sed "s|^${1}/v||" \
    | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -t. -k1,1n -k2,2n -k3,3n \
    | tail -1 || true
}

printf 'Verifying the contract registry...\n'

if [ ! -f "$REGISTRY" ]; then
  printf '\nFAIL: %s is missing.\n' "$REGISTRY" >&2
  exit 1
fi

if [ -z "$(git tag --list 'libs/*/v*' | head -1)" ]; then
  printf '\nFAIL: no release tags are present.\n' >&2
  printf 'This check compares the registry against published tags and cannot run\n' >&2
  printf 'on a shallow clone. Fetch tags (actions/checkout with fetch-depth: 0).\n' >&2
  exit 1
fi

# --- G1: every module row names the newest tag -------------------------------
#
# Rows in the "Published, and not offered" table have the shape
#   | `libs/x` | `v1.2.3` | no | no |

listed=""
while IFS= read -r row; do
  [ -n "$row" ] || continue
  module="$(printf '%s' "$row" | awk -F'|' '{ print $2 }' | tr -d ' `')"
  version="$(printf '%s' "$row" | awk -F'|' '{ print $3 }' | tr -d ' `')"
  [ -n "$module" ] || continue

  listed="${listed}${module} "
  newest="$(newest_tag "$module")"

  # A row for a module with no tag at all is a first release in flight: the row
  # has to be in the commit that gets tagged, and there is no earlier tag to
  # compare it against. Blocking it made a module's *first* release impossible —
  # `registry-check` refused the row and `release-tag` refused to tag without it.
  #
  # Reported rather than silent, because a row for a module nobody ever tags is
  # a claim about a version that does not exist.
  if [ -z "$newest" ]; then
    printf '  %-20s %-9s declares a first release; no tag exists yet\n' "$module" "$version"
    continue
  fi

  [ "$version" = "v${newest}" ] && continue

  # Unreleased changes mean a release is in flight, and the row is allowed to
  # name where it is going — but only forwards. A row *behind* the newest tag is
  # the stale case this check exists for, in either state.
  #
  # The diff is tag-to-working-tree, not tag-to-HEAD. In CI they are the same;
  # locally they are not, and comparing against HEAD would tell an author
  # editing a module that it has no unreleased changes.
  if [ -n "$(git diff --name-only "${module}/v${newest}" -- "$module" 2>/dev/null)" ]; then
    higher="$(printf '%s\n%s\n' "${version#v}" "$newest" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)"
    if [ "$higher" = "${version#v}" ]; then
      printf '  %-20s %-9s declares the release in flight (tagged v%s)\n' "$module" "$version" "$newest"
      continue
    fi
    fail "${module}: registry says ${version}, below its newest tag v${newest}"
    note "  | \`${module}\` | \`v${newest}\` | no | no |"
    continue
  fi

  fail "${module}: registry says ${version}, newest tag is v${newest}"
  note "  | \`${module}\` | \`v${newest}\` | no | no |"
done < <(grep -E '^\| `libs/[a-z-]+` \| `v[0-9]+\.[0-9]+\.[0-9]+` \|' "$REGISTRY" || true)

# --- G2: every published module is described somewhere -----------------------
#
# `libs/contracts` is the exception and it is a real one: it has its own
# "Published" section with a package table and a version history, so a row in
# the not-offered table would be a second, competing statement of its version.
# The exception is named here rather than assumed, and G4 is what keeps it
# honest.

for module in $(scripts/list-modules.sh | grep '^libs/'); do
  newest="$(newest_tag "$module")"
  [ -n "$newest" ] || continue
  [ "$module" = "libs/contracts" ] && continue

  case " ${listed} " in
    *" ${module} "*) ;;
    *)
      fail "${module}: published at v${newest} and absent from the registry"
      note "  | \`${module}\` | \`v${newest}\` | no | no |"
      ;;
  esac
done

# --- G3: the governance corpus row ------------------------------------------

corpus_newest="$(git tag --list 'ecosystem/v*' | sed 's|^ecosystem/||' | sort -t. -k1,1n -k2,2n -k3,3n | tail -1 || true)"
if [ -n "$corpus_newest" ]; then
  if ! grep -q "ecosystem/${corpus_newest}\`" "$REGISTRY"; then
    listed_corpus="$(grep -oE 'ecosystem/v[0-9]+\.[0-9]+\.[0-9]+' "$REGISTRY" | head -1 || true)"
    fail "the governance corpus: registry says ${listed_corpus:-nothing}, newest tag is ecosystem/${corpus_newest}"
  fi
fi

# --- G4: every published contracts version has a history row -----------------
#
# The one that would have caught v0.6.0, which was tagged, pinned by two modules
# and described nowhere.

for version in $(git tag --list 'libs/contracts/v*' | sed 's|^libs/contracts/||' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$'); do
  if ! grep -qE "^\| \`${version}\` \|" "$REGISTRY"; then
    fail "libs/contracts ${version}: published, with no row in the version history"
  fi
done

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d registry violation(s).\n' "$failures" >&2
  printf 'ADR-0024 makes this table part of the interface. A consumer reads it to\n' >&2
  printf 'learn what exists; a wrong row is a wrong answer, not untidiness.\n' >&2
  exit 1
fi

printf 'OK: the registry matches the published tags.\n'
