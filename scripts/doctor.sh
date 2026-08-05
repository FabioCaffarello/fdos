#!/usr/bin/env bash
#
# Diagnose a working copy and say what to do about it.
#
# `make toolchain-check` answers "is this correct" and fails. This answers "why
# does it not work on my machine" and never fails — it is a diagnostic, and a
# diagnostic that exits non-zero cannot be run when things are broken, which is
# the only time anyone wants it.
#
# Every finding names the fix. A doctor that reports symptoms without remedies
# just relocates the confusion.

set -uo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

problems=0

ok()   { printf '  \033[32m✓\033[0m %-30s %s\n' "$1" "${2:-}"; }
bad()  { printf '  \033[31m✗\033[0m %-30s %s\n' "$1" "$2"; problems=$((problems + 1)); }
hint() { printf '      → %s\n' "$1"; }

printf '\nFDOS — environment report\n\n'

# --- toolchain ---------------------------------------------------------------
printf 'Toolchain (pins from mise.toml)\n'
while IFS="$(printf '\t')" read -r tool pinned; do
  [ -n "$tool" ] || continue
  case "$tool" in
    go:*)
      ok "${tool##*/}" "$pinned (go run, no install needed)"
      continue
      ;;
  esac

  if ! command -v "$tool" >/dev/null 2>&1; then
    bad "$tool" "not installed (pinned ${pinned})"
    hint "mise install    # or install ${tool} ${pinned} by hand"
    continue
  fi

  case "$tool" in
    go)            actual="$(go version 2>/dev/null)" ;;
    lefthook)      actual="$(lefthook version 2>/dev/null)" ;;
    gitleaks)      actual="$(gitleaks version 2>/dev/null)" ;;
    *)             actual="$("$tool" --version 2>/dev/null)" ;;
  esac
  actual="$(printf '%s' "$actual" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"

  if [ "$actual" = "$pinned" ]; then
    ok "$tool" "$actual"
  else
    bad "$tool" "${actual:-unknown} installed, ${pinned} pinned"
    hint "mise install    # a wrong version is worse than no version"
  fi
done < <("${ROOT}/scripts/tool-version.sh" 2>/dev/null || true)

# --- repository state --------------------------------------------------------
printf '\nRepository\n'

if [ -f .git/hooks/pre-commit ] && grep -q lefthook .git/hooks/pre-commit 2>/dev/null; then
  ok "git hooks" "installed"
else
  bad "git hooks" "not installed"
  hint "make hooks"
fi

if [ -f go.work ]; then
  ok "go.work" "present (editor navigation; every make target overrides it)"
else
  bad "go.work" "missing"
  hint "go work init ./libs/analysis    # ADR-0004"
fi

modules="$(./scripts/list-modules.sh 2>/dev/null | wc -l | tr -d ' ')"
ok "modules" "$modules"

if git remote get-url origin >/dev/null 2>&1; then
  ok "remote" "$(git remote get-url origin)"
else
  bad "remote" "no origin configured"
  hint "git remote add origin git@github.com:FabioCaffarello/financial-data-operating-system.git"
fi

# --- environment that silently changes behaviour -----------------------------
printf '\nEnvironment\n'

if [ "${GOWORK:-}" = "off" ]; then
  ok "GOWORK" "off in this shell"
else
  ok "GOWORK" "${GOWORK:-unset} — make and CI set it per command (ADR-0004)"
fi

if [ -n "${GOFLAGS:-}" ] && [ "${GOFLAGS}" != "-mod=readonly" ]; then
  bad "GOFLAGS" "${GOFLAGS} — not -mod=readonly"
  hint "unset GOFLAGS; the Makefile exports the right value"
else
  ok "GOFLAGS" "${GOFLAGS:--mod=readonly (set by make)}"
fi

if [ -n "${GOPRIVATE:-}${GOPROXY:+}" ] && [ "${GOPROXY:-}" = "off" ]; then
  bad "GOPROXY" "off — module downloads will fail"
  hint "unset GOPROXY"
else
  ok "GOPROXY" "${GOPROXY:-default}"
fi

# --- summary -----------------------------------------------------------------
printf '\n'
if [ "$problems" -eq 0 ]; then
  printf 'No problems found. Run `make verify` for the full gate.\n\n'
else
  printf '%d problem(s) found. Fix the arrows above, then run `make verify`.\n\n' "$problems"
fi

# Always succeeds: a diagnostic that fails cannot be run when things are broken.
exit 0
