#!/usr/bin/env bash
#
# Fitness function: the tree is internally consistent, not only individually
# resolvable.
#
# `make verify` runs every module with GOWORK=off, so siblings resolve from the
# proxy at *published* versions. That is deliberate and load-bearing — it is
# what proves a module resolves standalone for a consumer with no workspace
# (ADR-0004) — and it means that while `libs/kernel` is being edited, the five
# modules importing it compile against the previous release from the module
# cache. A signature change is compile-checked only inside its own module, and
# its blast radius surfaces one tag at a time.
#
# ADR-0041 shipped into exactly that blind spot and said so in writing:
#
#   "it breaks every implementation of app.Store, including any out of tree,
#    and no mechanism here can see it."
#
# This is the other half. It compiles every module against its siblings' *source*
# and fails when the tree does not hold together. It does not replace the
# GOWORK=off runs; both properties matter and they are different properties.
#
# Two things worth knowing about how it is written:
#
#   - It runs `go vet`, not `go build`. Test files are where interface
#     assertions and doubles live, and `go build` does not compile them. The
#     defect this check was written against — apps/submitd's failingStore, which
#     stopped implementing app.Store when ADR-0041 added a method — is invisible
#     to `go build` and caught by `go vet`.
#
#   - It sets GOWORK to an explicit path. `verify.yml` exports GOWORK=off for
#     the whole workflow and mise.toml sets it for developers, so inheriting the
#     environment would make this check pass while testing nothing. It then
#     proves the workspace is live before trusting the result.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

GO="${GO:-go}"
WORK_FILE="${ROOT}/go.work"
failures=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying the workspace...\n'

if [ ! -f "$WORK_FILE" ]; then
  printf '\nFAIL: go.work is missing.\n' >&2
  exit 1
fi

modules="$(scripts/list-modules.sh)"

# `use` entries, normalised to the same relative form list-modules.sh prints.
used="$(
  awk '
    /^use[[:space:]]*\(/ { in_block = 1; next }
    in_block && /^\)/    { in_block = 0; next }
    in_block {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "")
      if ($0 == "" || $0 ~ /^\/\//) next
      sub(/^\.\//, "")
      print
    }
    /^use[[:space:]]+[^(]/ {
      line = $2
      sub(/^\.\//, "", line)
      print line
    }
  ' "$WORK_FILE" | sort -u
)"

# --- membership, both directions --------------------------------------------
#
# A module absent from go.work is one the workspace build silently skips, which
# is how libs/ledger-sqlite came to be the only module never compiled against
# local siblings. A `use` entry with no module is a path that will resolve to
# nothing the day someone relies on it.

while IFS= read -r module; do
  [ -n "$module" ] || continue
  if ! printf '%s\n' "$used" | grep -qx "$module"; then
    fail "${module}: is a module and is not in go.work — the workspace build cannot see it"
  fi
done <<EOF
$modules
EOF

while IFS= read -r entry; do
  [ -n "$entry" ] || continue
  if ! printf '%s\n' "$modules" | grep -qx "$entry"; then
    fail "go.work: uses '${entry}', which is not a module in this repository"
  fi
done <<EOF
$used
EOF

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d workspace membership violation(s).\n' "$failures" >&2
  exit 1
fi

printf '  membership: %d module(s), all present in go.work\n' "$(printf '%s\n' "$modules" | grep -c .)"

# --- the workspace is actually in effect ------------------------------------
#
# Checking the check. Under GOWORK=off every command below still succeeds and
# proves nothing, and GOWORK=off is the ambient setting in CI and in a mise
# shell. So: resolve one first-party dependency and require the answer to be
# inside this repository rather than in the module cache.

probe_module=""
probe_pkg=""
while IFS= read -r module; do
  [ -n "$module" ] || continue
  dep="$(
    grep -E '^[[:space:]]*github\.com/FabioCaffarello/fdos/' "${module}/go.mod" 2>/dev/null \
      | grep -v '// indirect' \
      | awk '{ print $1 }' \
      | head -1
  )"
  if [ -n "$dep" ]; then
    probe_module="$module"
    probe_pkg="$dep"
    break
  fi
done <<EOF
$modules
EOF

if [ -z "$probe_module" ]; then
  fail "no module declares a first-party dependency — this check cannot prove the workspace is live"
else
  where="$(cd "$probe_module" && GOWORK="$WORK_FILE" "$GO" list -m -f '{{.Dir}}' "$probe_pkg" 2>/dev/null || true)"
  case "$where" in
    "${ROOT}"/*)
      printf '  workspace live: %s resolves %s from the tree\n' "$probe_module" "${probe_pkg##*/}"
      ;;
    *)
      fail "the workspace is not in effect: ${probe_module} resolves ${probe_pkg} to '${where:-nothing}'"
      fail "  every result below would be vacuous, so this is a failure rather than a warning"
      ;;
  esac
fi

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d workspace violation(s).\n' "$failures" >&2
  exit 1
fi

# --- the tree holds together -------------------------------------------------

while IFS= read -r module; do
  [ -n "$module" ] || continue
  out="$(cd "$module" && GOWORK="$WORK_FILE" "$GO" vet ./... 2>&1)" || {
    fail "${module}: does not compile against its siblings' source"
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    continue
  }
  printf '  %s\n' "$module"
done <<EOF
$modules
EOF

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d module(s) inconsistent with the tree.\n' "$failures" >&2
  printf 'The per-module GOWORK=off runs cannot see this: they compile each module\n' >&2
  printf 'against the last published version of its siblings.\n' >&2
  exit 1
fi

printf 'OK: the tree builds as a workspace.\n'
