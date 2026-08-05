#!/usr/bin/env bash
#
# Fitness function: no secret has ever been committed.
#
# `.gitignore` covers *.pem, *.key and .env — but .gitignore is defence in
# depth, not enforcement. It prevents an accident; it detects nothing, and it is
# useless against a secret pasted into a Go file or a Markdown example.
#
# This scans the full history rather than the working tree. A secret removed in
# a later commit is still a leaked secret: the object remains reachable and
# anyone who cloned before the removal already has it.
#
# Private repositories hold real institution credentials. They are separate
# repositories precisely so that material never shares a trust boundary with the
# public core (Constitution §13) — this check is what keeps that boundary from
# eroding one convenient paste at a time.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

MODE="${1:-history}"

printf 'Scanning for committed secrets (%s)...\n' "$MODE"

case "$MODE" in
  history)
    # `git` mode scans commits. Exit code 1 means findings.
    gitleaks git --no-banner --redact --config .gitleaks.toml .
    ;;
  staged)
    # Used by the pre-commit hook: fast, and catches the paste before it lands.
    gitleaks git --no-banner --redact --staged --config .gitleaks.toml .
    ;;
  *)
    printf 'usage: %s [history|staged]\n' "$0" >&2
    exit 2
    ;;
esac

printf 'OK: no secrets found.\n'
