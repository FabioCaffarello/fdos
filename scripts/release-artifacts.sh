#!/usr/bin/env bash
#
# Assemble the release artifacts for one module into dist/.
#
#   make release-artifacts MODULE=libs/analysis
#   make release-artifacts MODULE=libs/kernel VERSION=v0.9.0
#
# Two kinds of artifact, and which ones appear is decided by what the module
# *is* rather than by the shape of its tag:
#
#   binaries    every `main` package the module contains, cross-compiled
#   module zip  for a `libs/*` module, the zip the proxy serves — which is the
#               artifact anyone actually consumes
#
# # Why this exists rather than the steps that were in release.yml
#
# The workflow hardcoded `libs/analysis` and `fdoslint`, and triggered on
# `libs/*/v*`. So **every library tag published fdoslint binaries**: the release
# for `libs/kernel/v0.9.0` carries `fdoslint_linux_amd64`, an SBOM named
# `fdoslint.spdx.json`, and a signed `SHA256SUMS` describing none of the kernel.
# Verified against the published releases for `libs/kernel/v0.9.0` and
# `libs/ledger-sqlite/v0.3.0` — both of them.
#
# An adopter who verifies the signature on a kernel release is verifying a
# linter. The signature is real and the association is nonsense, which is a worse
# failure than an unsigned release: it looks like evidence.
#
# Enforcement ladder position: none. It builds what a release publishes; the
# gate is what says the source was good.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

GO="${GO:-go}"
# A tag is `<module>/<version>`, so the workflow passes TAG and nothing has to
# split it in YAML. MODULE and VERSION still work for a local run.
TAG="${TAG:-}"
MODULE="${MODULE:-${1:-}}"
VERSION="${VERSION:-${2:-}}"

if [ -n "$TAG" ]; then
  MODULE="${TAG%/*}"
  VERSION="${TAG##*/}"
fi

# Under Actions, publish the module so later steps can name it without splitting
# the tag in a YAML expression. GitHub expressions have no string split, and the
# workarounds are unreadable; this is the same shape as `tool-version.sh`
# resolving a pin into an output.
if [ -n "${GITHUB_OUTPUT:-}" ] && [ -n "$MODULE" ]; then
  printf 'module=%s\n' "$MODULE" >> "$GITHUB_OUTPUT"
fi
DIST="${DIST:-${ROOT}/dist}"

die() {
  printf 'release-artifacts: %s\n' "$1" >&2
  exit 1
}

[ -n "$MODULE" ] || { printf 'usage: make release-artifacts MODULE=<path> [VERSION=vX.Y.Z]\n' >&2; exit 2; }
scripts/list-modules.sh | grep -qx "$MODULE" || die "'${MODULE}' is not a module in this repository"

rm -rf "$DIST"
mkdir -p "$DIST"

printf 'Assembling release artifacts for %s\n\n' "$MODULE"

# --- binaries, if the module has any -----------------------------------------
#
# Driven by what is in the module, not by the tag. A library with no `main`
# package produces no binaries and says so, instead of publishing somebody
# else's.

commands="$(
  cd "$MODULE" && GOWORK=off "$GO" list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./... 2>/dev/null || true
)"

if [ -z "${commands//[[:space:]]/}" ]; then
  printf '  no main package — this module publishes no binary\n'
else
  module_path="$(cd "$MODULE" && GOWORK=off "$GO" list -m)"
  while IFS= read -r pkg; do
    [ -n "$pkg" ] || continue
    name="${pkg##*/}"
    rel="./${pkg#"${module_path}"}"
    rel="${rel#./}"
    [ -n "$rel" ] && rel="./${rel}" || rel="."

    for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
      os="${target%/*}"
      arch="${target#*/}"
      ( cd "$MODULE" && GOWORK=off GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
          "$GO" build -trimpath -buildvcs=false \
          -o "${DIST}/${name}_${os}_${arch}" "$rel" )
    done
    printf '  built %s for four platforms\n' "$name"
  done <<EOF
$commands
EOF
fi

# --- the module zip, for a library -------------------------------------------
#
# This is the artifact an external build actually consumes. `fdoslint` was
# attested and `libs/contracts` — the module another repository pins — was not,
# which is the inversion ADR-0014 left open in its notes.
#
# It comes from the proxy rather than being built here on purpose: attesting a
# zip this workflow made would attest something nobody downloads. The bytes
# worth signing are the bytes `go mod download` returns.

case "$MODULE" in
  libs/*)
    [ -n "$VERSION" ] || die "a libs/* release needs VERSION to fetch the published module zip"
    module_path="$(cd "$MODULE" && GOWORK=off "$GO" list -m)"
    cache="$(mktemp -d)"
    if GOMODCACHE="$cache" GOWORK=off GOFLAGS= "$GO" mod download "${module_path}@${VERSION}" 2>/dev/null; then
      zip="$(find "$cache" -name "${VERSION}.zip" | head -1)"
      if [ -n "$zip" ]; then
        name="$(printf '%s' "$module_path" | tr '/' '_')_${VERSION}.zip"
        cp "$zip" "${DIST}/${name}"
        printf '  fetched the published module zip as %s\n' "$name"
      else
        die "the proxy served ${module_path}@${VERSION} but no zip was found in the cache"
      fi
    else
      die "${module_path}@${VERSION} is not resolvable yet — the proxy has not seen the tag"
    fi
    # The module cache is written read-only on purpose, so a plain rm fails.
    chmod -R u+w "$cache" 2>/dev/null || true
    rm -rf "$cache"
    ;;
  *)
    printf '  not a libs/* module — nothing is published for anyone to import\n'
    ;;
esac

# --- one manifest over everything -------------------------------------------

if [ -z "$(ls -A "$DIST")" ]; then
  die "nothing to release for ${MODULE}"
fi

( cd "$DIST" && shasum -a 256 * > SHA256SUMS 2>/dev/null || sha256sum * > SHA256SUMS )

printf '\nSHA256SUMS covers exactly what this release carries:\n\n'
sed 's/^/  /' "${DIST}/SHA256SUMS"
