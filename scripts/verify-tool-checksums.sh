#!/usr/bin/env bash
#
# Fitness function: every build input downloaded by URL is identified by digest.
#
# ADR-0014 pinned every GitHub Action to a commit SHA, on the grounds that a
# mutable reference is "an unreviewed third party with write access to the
# build". It then recorded, openly, that two inputs escaped that rule:
#
#   "The gitleaks install is pinned by version, not by checksum. Every other
#    build input is digest-pinned; this one is not, and stating it is better
#    than implying coverage."
#
# A version is not a digest. A GitHub release asset can be deleted and
# re-uploaded under the same tag, so the artifact behind a version URL can change
# with no commit here — the same mutation SHA-pinning removes for actions, on the
# inputs that install the tools every later attestation depends on.
#
# Three rules, because a digest can fail to be a pin in three ways:
#
#   1. an artifact is downloaded with no digest recorded
#   2. a digest is recorded but never compared — decoration, not enforcement
#   3. a digest is recorded for a version that is no longer pinned — stale, and
#      it passes rule 1 while describing an artifact nobody installs
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

CHECKSUMS="tool-checksums.txt"
failures=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying toolchain artifact digests...\n'

if [ ! -f "$CHECKSUMS" ]; then
  printf '\nFAIL: %s is missing.\n' "$CHECKSUMS" >&2
  exit 1
fi

# Tool names from mise.toml, longest first so `golangci-lint` is attributed
# before `go` when both appear in one URL. Backend keys such as
# `go:golang.org/x/vuln/...` are not PATH tools and are never downloaded here.
tools="$(
  scripts/tool-version.sh \
    | cut -f1 \
    | grep -v ':' \
    | awk '{ print length($0), $0 }' \
    | sort -rn \
    | cut -d' ' -f2-
)"

# --- rule 1 and 2: every downloaded artifact is attributed and compared -------

downloads="$(grep -rlE 'releases/download/' .github 2>/dev/null || true)"

if [ -z "$downloads" ]; then
  printf '  no URL downloads found under .github — nothing to pin\n'
fi

attributed=""
while IFS= read -r file; do
  [ -n "$file" ] || continue

  while IFS= read -r url_line; do
    [ -n "$url_line" ] || continue

    tool=""
    while IFS= read -r candidate; do
      [ -n "$candidate" ] || continue
      case "$url_line" in
        *"$candidate"*) tool="$candidate"; break ;;
      esac
    done <<EOF
$tools
EOF

    if [ -z "$tool" ]; then
      fail "${file}: downloads a release artifact this check cannot attribute to a pinned tool"
      fail "  ${url_line}"
      continue
    fi

    case " ${attributed} " in
      *" ${tool} "*) continue ;;
      *) attributed="${attributed}${tool} " ;;
    esac

    version="$(scripts/tool-version.sh "$tool")"
    tool_ok=true

    # Rule 1 — a digest exists for the pinned version.
    if ! awk -v t="$tool" -v v="$version" '
          /^[[:space:]]*#/ { next }
          NF == 0 { next }
          $1 == t && $2 == v { found = 1; exit }
          END { exit !found }
        ' "$CHECKSUMS"; then
      fail "${tool} ${version}: downloaded by ${file} with no digest in ${CHECKSUMS}"
      tool_ok=false
    fi

    # Rule 2 — the digest is actually consulted where the artifact is fetched.
    # A recorded digest nobody compares is decoration, and it reads in review
    # exactly like a pin.
    if ! grep -q 'tool-checksum\.sh' "$file"; then
      fail "${file}: downloads ${tool} but never resolves a digest through scripts/tool-checksum.sh"
      tool_ok=false
    fi

    # Only when both held. A check that reports a tool as pinned in the same
    # breath as failing it is a check nobody can read.
    if [ "$tool_ok" = true ]; then
      printf '  %-14s %-10s pinned by digest, verified in %s\n' "$tool" "$version" "$file"
    fi
  done < <(grep -E 'releases/download/' "$file" || true)
done <<EOF
$downloads
EOF

# --- rule 3: no entry describes a version that is not pinned -----------------

while IFS= read -r line; do
  [ -n "$line" ] || continue

  tool="$(printf '%s' "$line" | awk '{ print $1 }')"
  version="$(printf '%s' "$line" | awk '{ print $2 }')"
  platform="$(printf '%s' "$line" | awk '{ print $3 }')"
  digest="$(printf '%s' "$line" | awk '{ print $4 }')"

  if ! printf '%s' "$digest" | grep -qE '^[0-9a-f]{64}$'; then
    fail "${tool} ${version} ${platform}: '${digest}' is not a 64-character SHA-256 digest"
    continue
  fi

  pinned="$(scripts/tool-version.sh "$tool" 2>/dev/null || true)"
  if [ -z "$pinned" ]; then
    fail "${tool}: has a digest but is not pinned in mise.toml"
  elif [ "$pinned" != "$version" ]; then
    fail "${tool}: digest recorded for ${version}, but mise.toml pins ${pinned}"
  fi
done < <(grep -vE '^[[:space:]]*(#|$)' "$CHECKSUMS")

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d digest violation(s).\n' "$failures" >&2
  exit 1
fi

printf 'OK: every URL-downloaded build input is pinned by digest.\n'
