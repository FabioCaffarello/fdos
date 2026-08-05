#!/usr/bin/env bash
#
# Fitness function: the decision log obeys the same law as the financial ledger.
#
# ADRs are append-only and immutable. A decision that turns out to be wrong is
# not edited — it is superseded by a new decision that says so. The history of
# how FDOS came to be shaped must be reconstructible years later, for exactly
# the same reason a financial report must be.
#
# Enforcement ladder position: static (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=lib/frontmatter.sh
source "${ROOT}/scripts/lib/frontmatter.sh"

ADR_DIR="${ROOT}/docs/adr"
REQUIRED_KEYS="id title status date deciders"
VALID_STATUSES="Proposed Accepted Rejected Superseded"

failures=0
seen_ids=""

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying architecture decision records...\n'

if [ ! -f "${ADR_DIR}/template.md" ]; then
  fail "docs/adr/template.md: missing — the ADR process has no template"
fi

checked=0
for adr in "${ADR_DIR}"/[0-9][0-9][0-9][0-9]-*.md; do
  [ -f "$adr" ] || continue
  name="$(basename "$adr")"
  checked=$((checked + 1))

  # shellcheck disable=SC2086
  fm_require_keys "$adr" "docs/adr/${name}" ${REQUIRED_KEYS} || failures=$((failures + 1))

  id="$(fm_value "$adr" id || true)"
  expected_id="ADR-${name%%-*}"
  if [ "$id" != "$expected_id" ]; then
    fail "docs/adr/${name}: declares id '${id}' but filename implies '${expected_id}'"
  fi

  case " ${seen_ids} " in
    *" ${id} "*) fail "docs/adr/${name}: duplicate id '${id}'" ;;
    *) seen_ids="${seen_ids} ${id}" ;;
  esac

  status="$(fm_value "$adr" status || true)"
  case " ${VALID_STATUSES} " in
    *" ${status} "*) ;;
    *) fail "docs/adr/${name}: status '${status}' is not one of: ${VALID_STATUSES}" ;;
  esac

  # A superseded decision must name its successor, or the chain of reasoning
  # breaks and the log stops being reconstructible.
  if [ "$status" = "Superseded" ]; then
    count="$(fm_list_count "$adr" superseded_by || true)"
    if [ "${count:-0}" -lt 1 ]; then
      fail "docs/adr/${name}: status is Superseded but 'superseded_by' names no successor"
    fi
  fi

  date_value="$(fm_value "$adr" date || true)"
  if ! printf '%s' "$date_value" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'; then
    fail "docs/adr/${name}: date '${date_value}' is not ISO-8601 (YYYY-MM-DD)"
  fi
done

if [ "$checked" -eq 0 ]; then
  fail "docs/adr/: no ADRs found — ADR-0000 must exist"
fi

# Second pass: supersession links must be real and bidirectional.
#
# Added when the log's first supersession (ADR-0001 -> ADR-0006) showed that
# requiring a successor to be *named* does not require it to *exist*, or to
# point back. A one-directional link makes the chain traversable forwards only,
# which defeats the purpose: a reader arriving at the successor cannot tell what
# it replaced.
adr_file_for() { # <ADR-NNNN>
  local num="${1#ADR-}"
  local match
  match="$(ls "${ADR_DIR}/${num}"-*.md 2>/dev/null | head -1 || true)"
  printf '%s' "$match"
}

for adr in "${ADR_DIR}"/[0-9][0-9][0-9][0-9]-*.md; do
  [ -f "$adr" ] || continue
  name="$(basename "$adr")"
  id="$(fm_value "$adr" id || true)"

  for successor in $(fm_list_items "$adr" superseded_by || true); do
    target="$(adr_file_for "$successor")"
    if [ -z "$target" ]; then
      fail "docs/adr/${name}: superseded_by names '${successor}', which does not exist"
      continue
    fi
    if ! fm_list_items "$target" supersedes | grep -qx -- "$id"; then
      fail "docs/adr/${name}: '${successor}' does not list '${id}' in its 'supersedes' — supersession must be bidirectional"
    fi
  done

  for predecessor in $(fm_list_items "$adr" supersedes || true); do
    target="$(adr_file_for "$predecessor")"
    if [ -z "$target" ]; then
      fail "docs/adr/${name}: supersedes names '${predecessor}', which does not exist"
      continue
    fi
    target_status="$(fm_value "$target" status || true)"
    if [ "$target_status" != "Superseded" ]; then
      fail "docs/adr/${name}: supersedes '${predecessor}', but that ADR's status is '${target_status}', not 'Superseded'"
    fi
    if ! fm_list_items "$target" superseded_by | grep -qx -- "$id"; then
      fail "docs/adr/${name}: '${predecessor}' does not list '${id}' in its 'superseded_by' — supersession must be bidirectional"
    fi
  done
done

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d ADR violation(s) across %d records.\n' "$failures" "$checked" >&2
  exit 1
fi

printf 'OK: %d ADRs valid.\n' "$checked"
