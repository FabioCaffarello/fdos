#!/usr/bin/env bash
#
# Print every Go module in the repository, one relative path per line.
#
# ADR-0004 makes each libs/* an independent module, so almost every Go command
# has to be run per module rather than once at the root. This is the single
# place that knows how to find them.
#
# `examples` is in the search path, and was not until ADR-0044. Nothing in
# `make verify` touched it, so `examples/ingest` stopped compiling under a
# kernel change and stayed broken unreported while an enforcement row claimed
# the kit ran in CI. A conformance kit outside the gate proves conformance to
# whatever it last compiled against.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

find libs apps examples -name go.mod -not -path "*/testdata/*" 2>/dev/null \
  | sed 's|/go.mod$||' \
  | sort
