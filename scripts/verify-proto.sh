#!/usr/bin/env bash
#
# Fitness function: the published contract surface is well-formed, cannot change
# silently, and cannot be made to carry model output.
#
# Constitution §11 requires contracts to be versioned, documented and tested. A
# contract that can change without anything failing is not a contract, so
# `buf breaking` is the load-bearing part of this script.
#
# Five checks:
#
#   lint      buf STANDARD — naming, package layout, file structure
#   format    schemas are canonically formatted
#   breaking  no wire- or JSON-incompatible change against main (§11)
#   pinned    the codegen plugin is pinned to an exact version (ADR-0014)
#   drift     committed generated code matches the schemas it came from
#
# Plus two FDOS-specific schema rules that no general linter knows about:
#
#   envelope  every ledger fact carries an Envelope (§6, §7)
#   boundary  no ledger message can reference ModelOutput (§2)
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

PROTO_DIR="libs/contracts/proto"
# Fact packages only. Payloads live in fdos/ledger/payload/v* and carry no
# envelope by construction: they sit inside Fact.payload, and the fact around
# them carries the envelope for both. The alternative was exempting messages by
# name, and an exemption by name is a rule waiting to be worked around.
LEDGER_FACT_DIRS="${PROTO_DIR}/fdos/ledger/v1"
GEN_DIR="libs/contracts/gen"

failures=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying the contract surface...\n'

# --- lint --------------------------------------------------------------------
if buf lint 2>&1; then
  printf '  lint       STANDARD clean\n'
else
  fail "lint: buf lint reported violations"
fi

# --- format ------------------------------------------------------------------
if [ -z "$(buf format -d 2>/dev/null)" ]; then
  printf '  format     canonical\n'
else
  fail "format: schemas are not canonically formatted — run \`buf format -w\`"
fi

# --- breaking ----------------------------------------------------------------
#
# Compared against main, not against the previous commit: a branch that breaks
# the contract in one commit and repairs it in the next has still shipped a
# break if only adjacent commits are compared.
if ! git rev-parse --verify --quiet main >/dev/null 2>&1; then
  printf '  breaking   skipped (no main branch to compare against)\n'
elif [ -z "$(git ls-tree -r --name-only main -- "$PROTO_DIR" 2>/dev/null)" ]; then
  # Bootstrap: main carries no schemas yet. Adding the first contract is not a
  # breaking change, and buf reports the empty side as a failure. Stated
  # explicitly rather than swallowed — this branch stops applying the moment
  # these schemas land on main, and it must not quietly hide a real break.
  printf '  breaking   skipped (main carries no schemas yet — first contract)\n'
elif buf breaking --against '.git#branch=main' 2>&1; then
  printf '  breaking   compatible with main\n'
else
  fail "breaking: the change is not backward compatible with main"
  fail "    a contract that can change silently is not a contract (§11)"
fi

# --- codegen plugin is pinned ------------------------------------------------
#
# Generated code is a build input. A floating plugin tag makes the output
# irreproducible with no commit here — the same argument that pins every
# GitHub Action to a commit SHA (ADR-0014).
if grep -qE '(@latest|@main|:latest|:main)' buf.gen.yaml; then
  fail "pinned: buf.gen.yaml uses a floating plugin version"
elif grep -qE '^\s*-\s*local:.*@v[0-9]+\.[0-9]+\.[0-9]+' buf.gen.yaml; then
  printf '  pinned     %s\n' "$(grep -oE '[a-z.]+/[^"]*@v[0-9]+\.[0-9]+\.[0-9]+' buf.gen.yaml | head -1)"
elif grep -qE '^\s*-\s*remote:.*:v[0-9]+\.[0-9]+\.[0-9]+\s*$' buf.gen.yaml; then
  fail "pinned: the codegen plugin is remote — every run then depends on BSR"
  fail "    use a local \`go run pkg@version\` plugin instead (ADR-0018)"
else
  fail "pinned: no exactly-pinned plugin found in buf.gen.yaml"
fi

# --- generated code matches the schemas --------------------------------------
#
# Regenerates into a scratch tree and diffs. The working tree is never touched:
# a check that fixes what it checks cannot report a failure.
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

if buf generate --output "$WORK" >/dev/null 2>&1; then
  if diff -r "${WORK}/${GEN_DIR}" "$GEN_DIR" >/dev/null 2>&1; then
    printf '  drift      generated code matches the schemas\n'
  else
    fail "drift: committed generated code differs from the schemas — run \`make proto-gen\`"
    diff -rq "${WORK}/${GEN_DIR}" "$GEN_DIR" 2>&1 | sed 's/^/    /' >&2 || true
  fi
else
  fail "drift: buf generate failed"
fi

# --- FDOS schema rule: every ledger fact carries an Envelope -----------------
#
# proto3 has no `required`, so the schema cannot make an envelope-less fact
# unrepresentable — that guarantee belongs to the Go kernel types at M6. This
# enforces it one rung lower: a ledger message with no Envelope field would
# make provenance and bitemporality optional in practice (§6, §7).
if [ -d "$LEDGER_FACT_DIRS" ]; then
  while IFS= read -r file; do
    [ -f "$file" ] || continue
    while IFS= read -r message; do
      [ -n "$message" ] || continue
      # The Envelope definition itself is not a fact.
      [ "$message" = "Envelope" ] && continue
      if ! awk -v m="$message" '
        $0 ~ "^message " m " \\{" { inside = 1; next }
        inside && /^}/ { exit found ? 0 : 1 }
        inside && /Envelope envelope/ { found = 1 }
        END { exit found ? 0 : 1 }
      ' "$file"; then
        fail "envelope: ${file}: message ${message} has no Envelope — every ledger fact carries one (§6, §7)"
      fi
    done < <(grep -oE '^message [A-Za-z0-9_]+' "$file" | sed 's/^message //' || true)
  done < <(find "$LEDGER_FACT_DIRS" -name '*.proto' | sort)
  printf '  envelope   every ledger fact carries an Envelope\n'
fi

# --- FDOS schema rule: the truth boundary ------------------------------------
#
# Constitution §2: model output must never become financial truth. On the wire
# that means no message the ledger accepts may reference ModelOutput. This makes
# the boundary a schema property rather than a principle in a document.
if grep -rn 'ModelOutput' "${PROTO_DIR}/fdos/ledger" >/dev/null 2>&1; then
  fail "boundary: a ledger message references ModelOutput"
  grep -rn 'ModelOutput' "${PROTO_DIR}/fdos/ledger" | sed 's/^/    /' >&2
  fail "    a model may render a trace; it may not produce one (Constitution §2)"
else
  printf '  boundary   no ledger message can carry model output\n'
fi

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d contract violation(s).\n' "$failures" >&2
  exit 1
fi

printf 'OK: contract surface valid.\n'
