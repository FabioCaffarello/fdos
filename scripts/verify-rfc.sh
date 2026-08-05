#!/usr/bin/env bash
#
# Fitness function: the RFC set is well-formed and its lifecycle is honest.
#
# RFCs are where FDOS decides things that cannot be decided by an ADR alone.
# Unlike the ADR log they are NOT immutable — a Draft is meant to change. What
# must not happen is an RFC drifting into permanent limbo, or claiming to be
# Accepted without the ADRs that record what it settled.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=lib/frontmatter.sh
source "${ROOT}/scripts/lib/frontmatter.sh"

RFC_DIR="${ROOT}/docs/rfc"
ADR_DIR="${ROOT}/docs/adr"
REQUIRED_KEYS="id title status date authors"
VALID_STATUSES="Draft Proposed Accepted Rejected Withdrawn Superseded"

failures=0
seen_ids=""

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying requests for comments...\n'

if [ ! -f "${RFC_DIR}/template.md" ]; then
  fail "docs/rfc/template.md: missing — the RFC process has no template"
fi

checked=0
for rfc in "${RFC_DIR}"/[0-9][0-9][0-9][0-9]-*.md; do
  [ -f "$rfc" ] || continue
  name="$(basename "$rfc")"
  checked=$((checked + 1))

  # shellcheck disable=SC2086
  fm_require_keys "$rfc" "docs/rfc/${name}" ${REQUIRED_KEYS} || failures=$((failures + 1))

  id="$(fm_value "$rfc" id || true)"
  expected_id="RFC-${name%%-*}"
  if [ "$id" != "$expected_id" ]; then
    fail "docs/rfc/${name}: declares id '${id}' but filename implies '${expected_id}'"
  fi

  case " ${seen_ids} " in
    *" ${id} "*) fail "docs/rfc/${name}: duplicate id '${id}'" ;;
    *) seen_ids="${seen_ids} ${id}" ;;
  esac

  status="$(fm_value "$rfc" status || true)"
  case " ${VALID_STATUSES} " in
    *" ${status} "*) ;;
    *) fail "docs/rfc/${name}: status '${status}' is not one of: ${VALID_STATUSES}" ;;
  esac

  date_value="$(fm_value "$rfc" date || true)"
  if ! printf '%s' "$date_value" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'; then
    fail "docs/rfc/${name}: date '${date_value}' is not ISO-8601 (YYYY-MM-DD)"
  fi

  if [ "$(fm_list_count "$rfc" authors || true)" -lt 1 ]; then
    fail "docs/rfc/${name}: 'authors' names nobody"
  fi

  # An RFC that claims to be Accepted must have produced at least one ADR.
  # Otherwise "accepted" records an intention, not a decision — and the whole
  # point of the RFC stage is that it terminates in recorded decisions.
  if [ "$status" = "Accepted" ]; then
    if ! grep -rqlF "$id" "$ADR_DIR" 2>/dev/null; then
      fail "docs/rfc/${name}: status is Accepted but no ADR references '${id}' — an accepted RFC must produce the ADRs recording what it settled"
    fi
  fi
done

if [ "$checked" -eq 0 ]; then
  fail "docs/rfc/: no RFCs found"
fi

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d RFC violation(s) across %d documents.\n' "$failures" "$checked" >&2
  exit 1
fi

printf 'OK: %d RFCs valid.\n' "$checked"
