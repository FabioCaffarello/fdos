#!/usr/bin/env bash
#
# Fitness function: no known vulnerability is reachable from FDOS code.
#
# govulncheck is run via `go run` at the version pinned in mise.toml, rather
# than as an installed binary. This is deliberate: govulncheck embeds the
# go/packages loader, so a binary built with go1.25 cannot parse go1.26 source
# and fails with a toolchain error that reads exactly like a scan result. Running
# it through the project's own toolchain removes that entire failure mode.
#
# govulncheck reports only vulnerabilities that are actually *reachable* from
# this code, not everything present in the dependency graph. That distinction
# matters: a scanner that reports unreachable findings trains people to ignore
# it, and an ignored scanner enforces nothing.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
GO="${GO:-go}"
MODULE_PIN="go:golang.org/x/vuln/cmd/govulncheck"

version="$("${ROOT}/scripts/tool-version.sh" "$MODULE_PIN")"
tool="golang.org/x/vuln/cmd/govulncheck@${version}"

failures=0
checked=0

printf 'Verifying dependencies against the Go vulnerability database (%s)...\n' "$version"

while IFS= read -r module; do
  [ -n "$module" ] || continue
  checked=$((checked + 1))
  printf '>>> %s\n' "$module"

  # GOFLAGS is cleared: `go run` of a tool must be allowed to resolve its own
  # module graph, which -mod=readonly would forbid.
  if ! ( cd "${ROOT}/${module}" && GOWORK=off GOFLAGS= "$GO" run "$tool" ./... ); then
    failures=$((failures + 1))
  fi
done < <("${ROOT}/scripts/list-modules.sh")

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d module(s) have reachable vulnerabilities.\n' "$failures" >&2
  exit 1
fi

printf 'OK: %d module(s) free of reachable vulnerabilities.\n' "$checked"
