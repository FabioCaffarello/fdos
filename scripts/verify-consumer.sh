#!/usr/bin/env bash
#
# Fitness function: the published contract module is actually consumable.
#
# ADR-0004 makes the open-core boundary depend on private repositories importing
# a published contract version rather than a local path. Everything up to now
# proved the *producing* half — the module builds, the schemas gate, `GOWORK=off`
# forces published resolution inside this repository.
#
# This proves the consuming half: a module created outside this repository, with
# no workspace, no `replace` and no filesystem path, resolves the contract
# through the Go proxy and compiles against it.
#
# NOT part of `make verify`, deliberately. It depends on a published tag having
# propagated to a third-party proxy, which is latency the per-commit gate must
# not inherit — the same mistake as the remote codegen plugin (ADR-0018). It
# runs at release and on demand.
#
# Enforcement ladder position: CI at release (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
GO="${GO:-go}"
MODULE="github.com/FabioCaffarello/fdos/libs/contracts"

version="${1:-}"
if [ -z "$version" ]; then
  version="$(git -C "$ROOT" tag --list 'libs/contracts/v*' --sort=-v:refname | head -1 | sed 's|.*/||')"
fi

printf 'Verifying the published contract module is consumable...\n'

if [ -z "$version" ]; then
  printf '  no libs/contracts/v* tag exists yet — nothing published to consume\n'
  printf 'SKIP: publish a version first (git tag libs/contracts/vX.Y.Z).\n'
  exit 0
fi

printf '  module     %s@%s\n' "$MODULE" "$version"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

cd "$WORK"

# A consumer that knows nothing about this repository: no go.work, no replace,
# no path. If this resolves, a private connector can do the same.
"$GO" mod init fdos-consumer-probe >/dev/null 2>&1

cat > main.go <<'EOF'
// Package main is a throwaway consumer proving the published FDOS contract
// module resolves and compiles from outside the repository that produces it.
package main

import (
	"fmt"

	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
	ledgerv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ledger/v1"
)

func main() {
	amount := &kernelv1.Money{
		Amount:   &kernelv1.Decimal{Value: "1234.5600"},
		Currency: "BRL",
	}

	fact := &ledgerv1.Fact{
		Kind:        ledgerv1.FactKind_FACT_KIND_OBSERVATION,
		Type:        "ledger.HoldingObserved",
		TypeVersion: 1,
	}

	fmt.Println(amount.GetCurrency(), fact.GetType())
}
EOF

failures=0

# GOWORK=off and an empty GOFLAGS: the probe must resolve through the proxy,
# and -mod=readonly would forbid it adding the requirement it is testing.
if ! GOWORK=off GOFLAGS= "$GO" get "${MODULE}@${version}" >/dev/null 2>&1; then
  printf '  resolve    FAILED\n' >&2
  GOWORK=off GOFLAGS= "$GO" get "${MODULE}@${version}" 2>&1 | sed 's/^/    /' >&2 || true
  failures=$((failures + 1))
else
  printf '  resolve    fetched from the proxy, no local path\n'
fi

# `go get module@version` pins the module but does not walk the import graph of
# the packages actually used, so transitive sums are missing. A real consumer
# hits exactly this, which is why the probe does what a real consumer does
# rather than what is convenient.
if [ "$failures" -eq 0 ]; then
  if ! GOWORK=off GOFLAGS= "$GO" mod tidy >/dev/null 2>&1; then
    printf '  sums       FAILED to resolve the transitive graph\n' >&2
    GOWORK=off GOFLAGS= "$GO" mod tidy 2>&1 | sed 's/^/    /' >&2 || true
    failures=$((failures + 1))
  else
    resolved="$(grep -E "^\s+${MODULE} v" go.mod 2>/dev/null | awk '{print $2}' || true)"
    if [ -n "$resolved" ] && [ "$resolved" != "$version" ]; then
      printf '  sums       FAILED: tidy moved the version to %s\n' "$resolved" >&2
      failures=$((failures + 1))
    else
      printf '  sums       transitive graph resolved, version held at %s\n' "$version"
    fi
  fi
fi

if [ "$failures" -eq 0 ]; then
  if ! GOWORK=off GOFLAGS= "$GO" build ./... >/dev/null 2>&1; then
    printf '  compile    FAILED\n' >&2
    GOWORK=off GOFLAGS= "$GO" build ./... 2>&1 | sed 's/^/    /' >&2 || true
    failures=$((failures + 1))
  else
    printf '  compile    consumer builds against the published contract\n'
  fi
fi

# The point of the exercise: prove no local path leaked into the resolution.
if [ "$failures" -eq 0 ]; then
  if grep -q '^replace' go.mod 2>/dev/null; then
    printf '  no-replace FAILED: the probe used a replace directive\n' >&2
    failures=$((failures + 1))
  else
    printf '  no-replace resolution used no replace directive\n'
  fi
fi

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: the published contract module is not consumable.\n' >&2
  printf 'The open-core boundary (ADR-0004) is not real until this passes.\n' >&2
  exit 1
fi

printf 'OK: %s@%s is consumable from outside this repository.\n' "$MODULE" "$version"
