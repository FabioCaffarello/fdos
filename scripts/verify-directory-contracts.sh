#!/usr/bin/env bash
#
# Fitness function: every top-level directory and every module under libs/ must
# declare its architectural contract, and that declaration must agree with
# CODEOWNERS.
#
# The libs/ half was missing until M7. The check walked `${ROOT}/*/` only, so it
# reported "10 directory contracts valid" while libs/kernel and libs/ledger —
# the two modules holding the domain core — had no README at all. Four of the
# six modules had one by convention, which is exactly how a gap like this stays
# invisible: the files that exist look like evidence of a rule.
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
EXCLUDED_DIRS=".git .claude"

failures=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

# codeowners_for <path> — the owner CODEOWNERS assigns to a directory path,
# most specific rule first, falling back to the catch-all.
#
# Walks parent prefixes so `libs/kernel` is answered by `/libs/kernel/` if such
# a rule exists and by `/libs/` if it does not. Without the walk, every nested
# module would resolve to the catch-all and a future per-module owner would be
# silently ignored — the check would keep passing while enforcing nothing.
codeowners_for() {
  local path="$1" owner=""
  while [ -n "$path" ] && [ "$path" != "." ]; do
    owner="$(grep -E "^/${path}/[[:space:]]" "${ROOT}/CODEOWNERS" 2>/dev/null | tail -1 | awk '{print $NF}' || true)"
    [ -n "$owner" ] && { printf '%s' "$owner"; return; }
    [ "$path" = "${path%/*}" ] && break
    path="${path%/*}"
  done
  grep -E '^\*[[:space:]]' "${ROOT}/CODEOWNERS" 2>/dev/null | tail -1 | awk '{print $NF}' || true
}

printf 'Verifying directory contracts...\n'

if [ ! -f "${ROOT}/README.md" ]; then
  fail "README.md: missing at repository root"
fi

checked=0

# check_contract <absolute-dir-with-trailing-slash> <repo-relative-path>
#
# The `directory:` field names the directory's own basename, not its path —
# `kernel`, not `libs/kernel`. Messages use the full path, because "kernel/" is
# ambiguous the moment the check descends.
check_contract() {
  local path="$1" rel="$2"
  local dir readme declared declared_owner actual_owner count list_key

  dir="$(basename "$rel")"
  checked=$((checked + 1))
  readme="${path}README.md"

  if [ ! -f "$readme" ]; then
    fail "${rel}/: no README.md — every directory must declare its contract"
    return
  fi

  # shellcheck disable=SC2086
  fm_require_keys "$readme" "${rel}/README.md" ${REQUIRED_KEYS} || failures=$((failures + 1))

  declared="$(fm_value "$readme" directory || true)"
  if [ "$declared" != "$dir" ]; then
    fail "${rel}/README.md: front matter declares directory '${declared}', but lives in '${rel}/'"
  fi

  for list_key in allowed forbidden; do
    count="$(fm_list_count "$readme" "$list_key" || true)"
    if [ "${count:-0}" -lt 1 ]; then
      fail "${rel}/README.md: '${list_key}' is empty — a contract that permits or forbids nothing is not a contract"
    fi
  done

  declared_owner="$(fm_value "$readme" owner || true)"
  actual_owner="$(codeowners_for "$rel")"
  if [ -n "$declared_owner" ] && [ -n "$actual_owner" ] && [ "$declared_owner" != "$actual_owner" ]; then
    fail "${rel}/README.md: owner '${declared_owner}' disagrees with CODEOWNERS ('${actual_owner}')"
  fi
}

for path in "${ROOT}"/*/ "${ROOT}"/.*/; do
  [ -d "$path" ] || continue
  dir="$(basename "$path")"

  case "$dir" in
    .|..) continue ;;
  esac
  case " ${EXCLUDED_DIRS} " in
    *" ${dir} "*) continue ;;
  esac

  check_contract "$path" "$dir"
done

# Every module under libs/ declares its own contract. ADR-0004 makes each one an
# independently published Go module, so each is a boundary somebody outside this
# repository can depend on — a boundary with no stated contract is one nobody
# can be held to.
#
# One level only. Layers below a module are packages, not boundaries (ADR-0013),
# and requiring a README per package would produce contracts nobody reads to
# satisfy a check nobody believes.
for path in "${ROOT}"/libs/*/; do
  [ -d "$path" ] || continue
  check_contract "$path" "libs/$(basename "$path")"
done

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d directory contract violation(s) across %d directories.\n' "$failures" "$checked" >&2
  exit 1
fi

printf 'OK: %d directory contracts valid.\n' "$checked"
