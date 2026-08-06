#!/usr/bin/env bash
#
# Fitness function: documentation describes the repository that exists.
#
# `.context/` is derived from `docs/`, and until now nothing checked the
# derivation. A playbook naming a `make` target that was renamed, or citing an
# ADR that was never written, is worse than no playbook: an agent acts on it
# confidently and nothing reports that it was wrong.
#
# Checks every Markdown file for references that can be resolved mechanically:
# relative links, `make` targets, script paths, and ADR/RFC identifiers. It
# cannot tell whether a paragraph is merely wrong — only whether the things it
# names still exist. That is the honest scope (ADR-0015).
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

failures=0
files=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying documentation references...\n'

make_targets="$(awk '/^[a-z][a-z0-9-]*:/ { t = $0; sub(/:.*$/, "", t); print t }' Makefile | sort -u)"

while IFS= read -r doc; do
  [ -f "$doc" ] || continue
  dir="$(dirname "$doc")"
  files=$((files + 1))

  # The decision log is a historical record, not live documentation, and
  # ADR-0000 forbids editing an accepted entry. Two consequences:
  #
  #   - An ADR legitimately names things that do not exist: a rejected
  #     alternative is recorded precisely so the reasoning survives.
  #   - If a `make` target or script is renamed years from now, every ADR
  #     naming it becomes "stale" and none of them may be corrected.
  #
  # So ADRs and RFCs are exempt from the mutable-name checks. Links and
  # ADR/RFC identifiers are still verified: those name stable things, and a
  # broken one is a real defect rather than the passage of time.
  historical=false
  case "$doc" in
    docs/adr/*|docs/rfc/*) historical=true ;;
  esac

  # --- relative links resolve -------------------------------------------------
  while IFS= read -r link; do
    [ -n "$link" ] || continue
    case "$link" in
      http*|mailto:*|"#"*) continue ;;
    esac
    target="${link%%#*}"
    [ -n "$target" ] || continue
    if [ ! -e "${dir}/${target}" ]; then
      fail "${doc}: link to '${target}' does not resolve"
    fi
  done < <(grep -oE '\]\([^)]+\)' "$doc" 2>/dev/null | sed -E 's/^\]\((.*)\)$/\1/' || true)

  # --- `make <target>` names a real target ------------------------------------
  if [ "$historical" = false ]; then
    while IFS= read -r target; do
      [ -n "$target" ] || continue
      if ! printf '%s\n' "$make_targets" | grep -qx -- "$target"; then
        fail "${doc}: references \`make ${target}\`, which is not a Makefile target"
      fi
    done < <(
      {
        grep -oE '`make [a-z][a-z0-9-]*' "$doc" 2>/dev/null | sed 's/^`make //'
        grep -oE '^make [a-z][a-z0-9-]*' "$doc" 2>/dev/null | sed 's/^make //'
      } | sort -u || true
    )

    # --- scripts/*.sh exists --------------------------------------------------
    while IFS= read -r script; do
      [ -n "$script" ] || continue
      if [ ! -f "$script" ]; then
        fail "${doc}: references '${script}', which does not exist"
      fi
    done < <(grep -oE 'scripts/[a-zA-Z0-9_/-]+\.sh' "$doc" 2>/dev/null | sort -u || true)
  fi

  # --- ADR / RFC identifiers resolve ------------------------------------------
  #
  # A decision in another repository is cited qualified — `fdos-connectors:ADR-0026`
  # — because each repository keeps its own sequence and a bare `ADR-0026` in a
  # document spanning both is ambiguous. Those are skipped: resolving them here
  # would demand that this repository contain the other one's decision log.
  #
  # `fdos:ADR-0014` is qualified *and* local, so it is still checked. Dropping
  # every qualified reference would let a typo in this repository's own name
  # silence the check.
  for kind in ADR RFC; do
    case "$kind" in
      ADR) subdir="adr" ;;
      RFC) subdir="rfc" ;;
    esac

    while IFS= read -r ref; do
      [ -n "$ref" ] || continue

      case "$ref" in
        *:*)
          # Qualified. Only this repository's own prefix is resolvable here.
          [ "${ref%%:*}" = "fdos" ] || continue
          ;;
      esac

      id="${ref##*:}"
      num="${id#${kind}-}"
      if ! ls "docs/${subdir}/${num}"-*.md >/dev/null 2>&1; then
        fail "${doc}: cites ${id}, which does not exist"
      fi
    done < <(grep -oE "[A-Za-z0-9._-]*:?${kind}-[0-9]{4}" "$doc" 2>/dev/null | sort -u || true)
  done

done < <(
  # Filesystem, not the git index: a new document is untracked, and a stale
  # reference in one is exactly as wrong as in a committed file.
  find . -name '*.md' \
    -not -path './.git/*' \
    -not -path '*/testdata/*' \
    -not -path '*/node_modules/*' \
    -not -path './.context/cache/*' \
    -not -path './.context/runtime/*' \
    -not -path './.claude/*' \
    2>/dev/null | sed 's|^\./||' | sort || true
)

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d stale reference(s) across %d documents.\n' "$failures" "$files" >&2
  exit 1
fi

printf 'OK: %d documents, every reference resolves.\n' "$files"
