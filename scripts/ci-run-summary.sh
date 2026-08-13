#!/usr/bin/env bash
#
# Records what a CI run ran on, so a slow run can be told from a degrading one.
#
#   make ci-summary
#
# # What this got wrong the first time, and what replaced it
#
# It recorded `setup-go`'s `cache-hit` boolean and treated it as the explanation
# for the gate's bimodal duration. It explains nothing: runs reporting a hit were
# as slow as runs reporting a miss, while a tree whose dependencies had been
# stable ran the same checks 4.2x faster on the same runner minutes apart —
# `test` 12.9s against 81.3s, `vet` 2.3s against 32.4s, and `vuln-check`, the one
# check that does not compile the repository, unchanged at 18s (#139).
#
# A boolean cannot distinguish "a cache was restored" from "a cache useful for
# the dependency set now being compiled was restored". Two things can:
#
#   the go.sum fingerprint   `setup-go` keys its cache on `**/go.sum`, so two
#                            runs can only share a cache if this matches. It is
#                            computed from the tree, so it is comparable across
#                            runs, machines and time.
#
#   the restored cache size  measured immediately after the restore and before
#                            anything compiles. A cold start is small; a warm one
#                            is hundreds of megabytes.
#
# `cache-hit` is still recorded, and is no longer described as the reason for
# anything.
#
# Enforcement ladder position: none. Reporting only.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

cache_state="${TOOLCHAIN_CACHE_HIT:-unknown}"
case "$cache_state" in
  true)  cache_label="reported hit" ;;
  false) cache_label="reported miss" ;;
  *)     cache_label="unknown (not running under the toolchain action)" ;;
esac

# The cache key's input, not the key itself: `setup-go` hashes these files, and
# what matters here is whether two runs saw the same set. Sorted so the digest
# does not depend on `find` order.
gosum_digest="$(
  git ls-files '*go.sum' \
    | sort \
    | xargs cat 2>/dev/null \
    | { shasum -a 256 2>/dev/null || sha256sum; } \
    | cut -c1-12
)"
gosum_count="$(git ls-files '*go.sum' | grep -c . || true)"

restored_kb="${TOOLCHAIN_CACHE_BYTES:-}"
if [ -n "$restored_kb" ] && [ "$restored_kb" -gt 0 ] 2>/dev/null; then
  restored="$(( restored_kb / 1024 )) MB"
else
  restored="${restored_kb:-unknown}"
  [ "$restored" = "0" ] && restored="0 MB — nothing was restored"
fi

go_version="$(scripts/tool-version.sh go 2>/dev/null || printf 'unresolved')"
go_actual="$(go version 2>/dev/null | awk '{print $3}' || printf 'absent')"

printf 'Build cache restored:  %s\n' "$restored"
printf 'go.sum fingerprint:    %s  (%s files)\n' "$gosum_digest" "$gosum_count"
printf 'setup-go said:         %s\n' "$cache_label"
printf 'Runner:                %s %s\n' "${RUNNER_OS:-$(uname -s)}" "${RUNNER_ARCH:-$(uname -m)}"
printf 'Go pinned / present:   %s / %s\n' "$go_version" "$go_actual"
printf 'Commit:                %s\n' "$(git rev-parse --short HEAD)"

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    printf '### Run environment\n\n'
    printf '| | |\n|---|---|\n'
    printf '| Build cache restored | **%s** |\n' "$restored"
    printf '| `go.sum` fingerprint | `%s` (%s files) |\n' "$gosum_digest" "$gosum_count"
    printf '| `setup-go` reported | %s |\n' "$cache_label"
    printf '| Runner | %s %s |\n' "${RUNNER_OS:-unknown}" "${RUNNER_ARCH:-unknown}"
    printf '| Go pinned / present | `%s` / `%s` |\n' "$go_version" "$go_actual"
    printf '| Commit | `%s` |\n' "$(git rev-parse --short HEAD)"
    printf '\nA run can only reuse another run'"'"'s build cache if the `go.sum`\n'
    printf 'fingerprint matches. `setup-go` reporting a hit does not imply the\n'
    printf 'restored cache is useful for what is being compiled (#139).\n'
  } >> "$GITHUB_STEP_SUMMARY"
fi
