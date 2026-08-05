#!/usr/bin/env bash
#
# Fitness function: the installed toolchain matches the pins in mise.toml.
#
# mise.toml is the source of truth, but mise itself is NOT a prerequisite. This
# script reads the pins through scripts/tool-version.sh and validates whatever
# is on PATH, so the pin is enforced identically for a developer using mise, a
# developer installing by hand, and CI.
#
# Tools are graded by the milestone that makes them load-bearing. A tool that is
# not yet required is reported when absent but does not fail the build; when it
# IS present, its version is enforced — a wrong version is always an error,
# because a wrong version is worse than no version.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"

# Tools that must be present for the current milestone (M3).
REQUIRED_TOOLS="go golangci-lint gitleaks"

failures=0
warnings=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

warn() {
  printf '  %s\n' "$1"
  warnings=$((warnings + 1))
}

# installed_version <tool> — first semantic version reported by the tool itself.
installed_version() {
  local tool="$1" out=""
  case "$tool" in
    go)            out="$(go version 2>/dev/null || true)" ;;
    golangci-lint) out="$(golangci-lint --version 2>/dev/null || true)" ;;
    buf)           out="$(buf --version 2>/dev/null || true)" ;;
    lefthook)      out="$(lefthook version 2>/dev/null || true)" ;;
    gitleaks)      out="$(gitleaks version 2>/dev/null || true)" ;;
    *)             out="$("$tool" --version 2>/dev/null || true)" ;;
  esac
  printf '%s' "$out" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true
}

is_required() {
  case " ${REQUIRED_TOOLS} " in
    *" $1 "*) return 0 ;;
    *) return 1 ;;
  esac
}

printf 'Verifying toolchain against mise.toml pins...\n'

while IFS="$(printf '\t')" read -r tool pinned; do
  [ -n "$tool" ] || continue

  # `go:` entries pin a Go module run with `go run`, not a PATH binary. They are
  # built by the pinned toolchain on demand, so there is nothing to check here.
  case "$tool" in
    go:*)
      printf '  %-16s %s (go run, built by the pinned toolchain)\n' "${tool##*/}" "$pinned"
      continue
      ;;
  esac

  if ! command -v "$tool" >/dev/null 2>&1; then
    if is_required "$tool"; then
      fail "${tool}: not installed (pinned ${pinned}) — required by the current milestone"
    else
      warn "${tool}: not installed (pinned ${pinned}) — not yet required; install before the milestone that needs it"
    fi
    continue
  fi

  actual="$(installed_version "$tool")"
  if [ -z "$actual" ]; then
    warn "${tool}: installed, but its version could not be parsed (pinned ${pinned})"
    continue
  fi

  if [ "$actual" != "$pinned" ]; then
    fail "${tool}: version ${actual} does not match the pin ${pinned} — update the tool, or amend mise.toml with an ADR"
    continue
  fi

  printf '  %-16s %s\n' "$tool" "$actual"
done < <("${ROOT}/scripts/tool-version.sh")

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d toolchain violation(s).\n' "$failures" >&2
  exit 1
fi

if [ "$warnings" -gt 0 ]; then
  printf 'OK: pinned tools valid (%d not yet installed, not yet required).\n' "$warnings"
else
  printf 'OK: toolchain matches all pins.\n'
fi
