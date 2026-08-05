#!/usr/bin/env bash
#
# Fitness function: every top-level directory must declare its architectural
# contract, and that declaration must agree with CODEOWNERS.
#
# This is the first enforcement mechanism in FDOS and it is deliberately the
# most boring one. Its purpose is structural: from M2 onwards the `allowed` and
# `forbidden` lists in these READMEs are the SOURCE of the import-boundary
# configuration, not a description of it. A README that lies will break the
# build. That is what "documentation is production code" has to mean if it is
# to mean anything.
#
# Enforcement ladder position: static (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=lib/frontmatter.sh
source "${ROOT}/scripts/lib/frontmatter.sh"

REQUIRED_KEYS="directory purpose owner allowed forbidden"

# VCS-internal trees carry no architectural contract. `.context/` is NOT
# excluded: as of M1 it declares its own contract like every other directory,
# which is what ADR-0006 promised. `.claude/` was excluded while it was a
# gitignored export; ADR-0017 versions it, so it declares a contract too —
# generated is not a reason to be unaccountable once it is committed.
EXCLUDED_DIRS=".git"

failures=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

# codeowners_for <dir> — the owner CODEOWNERS assigns to a top-level directory,
# falling back to the catch-all rule.
codeowners_for() {
  local dir="$1" owner
  owner="$(grep -E "^/${dir}/[[:space:]]" "${ROOT}/CODEOWNERS" 2>/dev/null | tail -1 | awk '{print $NF}' || true)"
  if [ -z "$owner" ]; then
    owner="$(grep -E '^\*[[:space:]]' "${ROOT}/CODEOWNERS" 2>/dev/null | tail -1 | awk '{print $NF}' || true)"
  fi
  printf '%s' "$owner"
}

printf 'Verifying directory contracts...\n'

if [ ! -f "${ROOT}/README.md" ]; then
  fail "README.md: missing at repository root"
fi

checked=0
for path in "${ROOT}"/*/ "${ROOT}"/.*/; do
  [ -d "$path" ] || continue
  dir="$(basename "$path")"

  case "$dir" in
    .|..) continue ;;
  esac
  case " ${EXCLUDED_DIRS} " in
    *" ${dir} "*) continue ;;
  esac

  checked=$((checked + 1))
  readme="${path}README.md"

  if [ ! -f "$readme" ]; then
    fail "${dir}/: no README.md — every directory must declare its contract"
    continue
  fi

  # shellcheck disable=SC2086
  fm_require_keys "$readme" "${dir}/README.md" ${REQUIRED_KEYS} || failures=$((failures + 1))

  declared="$(fm_value "$readme" directory || true)"
  if [ "$declared" != "$dir" ]; then
    fail "${dir}/README.md: front matter declares directory '${declared}', but lives in '${dir}/'"
  fi

  for list_key in allowed forbidden; do
    count="$(fm_list_count "$readme" "$list_key" || true)"
    if [ "${count:-0}" -lt 1 ]; then
      fail "${dir}/README.md: '${list_key}' is empty — a contract that permits or forbids nothing is not a contract"
    fi
  done

  declared_owner="$(fm_value "$readme" owner || true)"
  actual_owner="$(codeowners_for "$dir")"
  if [ -n "$declared_owner" ] && [ -n "$actual_owner" ] && [ "$declared_owner" != "$actual_owner" ]; then
    fail "${dir}/README.md: owner '${declared_owner}' disagrees with CODEOWNERS ('${actual_owner}')"
  fi
done

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d directory contract violation(s) across %d directories.\n' "$failures" "$checked" >&2
  exit 1
fi

printf 'OK: %d directory contracts valid.\n' "$checked"
