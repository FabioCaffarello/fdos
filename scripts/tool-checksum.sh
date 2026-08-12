#!/usr/bin/env bash
#
# Print the SHA-256 digest of a toolchain artifact from tool-checksums.txt.
#
# The digest counterpart of `tool-version.sh`, and it exists for the same reason:
# one parser, so the composite action cannot install an artifact the repository
# did not describe. CI resolves the version through one script and the digest
# through this one; neither declares a value of its own.
#
#   scripts/tool-checksum.sh gitleaks linux_x64
#
# The version is not a parameter. It is read from mise.toml, so asking for a
# digest is asking for *the pinned version's* digest — there is no way to
# verify one version's artifact against another version's entry.
#
# Fails when there is no entry. Printing nothing would let the caller compare
# against an empty string, which every artifact fails, or succeed against `-`,
# which every artifact passes depending on the checker. Neither is a pin.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
CHECKSUMS="${ROOT}/tool-checksums.txt"

if [ "$#" -ne 2 ]; then
  printf 'usage: %s <tool> <platform>\n' "$0" >&2
  exit 2
fi

tool="$1"
platform="$2"

if [ ! -f "$CHECKSUMS" ]; then
  printf 'tool-checksum: tool-checksums.txt not found\n' >&2
  exit 1
fi

version="$("${ROOT}/scripts/tool-version.sh" "$tool")"

digest="$(
  awk -v t="$tool" -v v="$version" -v p="$platform" '
    /^[[:space:]]*#/ { next }
    NF == 0 { next }
    $1 == t && $2 == v && $3 == p { print $4; exit }
  ' "$CHECKSUMS"
)"

if [ -z "$digest" ]; then
  printf 'tool-checksum: no digest for %s %s (%s)\n' "$tool" "$version" "$platform" >&2
  printf '  tool-checksums.txt has no entry for the version pinned in mise.toml.\n' >&2
  printf '  Record it before installing:\n' >&2
  printf '    curl -fsSL -o /tmp/a "<url>" && shasum -a 256 /tmp/a\n' >&2
  exit 1
fi

printf '%s\n' "$digest"
