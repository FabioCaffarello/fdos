#!/usr/bin/env bash
#
# Print every Go module in the repository, one relative path per line.
#
# ADR-0004 makes each libs/* an independent module, so almost every Go command
# has to be run per module rather than once at the root. This is the single
# place that knows how to find them.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

find libs apps -name go.mod -not -path "*/testdata/*" 2>/dev/null \
  | sed 's|/go.mod$||' \
  | sort
