#!/usr/bin/env bash
#
# Devcontainer setup. Installs mise, then lets mise install the toolchain from
# mise.toml — the same pins `make toolchain-check` enforces and the same ones CI
# reads through scripts/tool-version.sh.
#
# Nothing here declares a version. If this script ever needs to know one, that
# is the signal it has become a second source of truth and should read
# mise.toml instead.

set -euo pipefail

printf '==> Installing mise\n'
if ! command -v mise >/dev/null 2>&1; then
  curl -fsSL https://mise.run | sh
  export PATH="${HOME}/.local/bin:${PATH}"
fi

# Activate mise for interactive shells.
for rc in "${HOME}/.bashrc" "${HOME}/.zshrc"; do
  [ -f "$rc" ] || continue
  grep -q 'mise activate' "$rc" || printf '\neval "$(mise activate %s)"\n' "$(basename "$rc" | sed 's/^\.//;s/rc$//')" >> "$rc"
done

printf '==> Installing pinned toolchain from mise.toml\n'
mise trust --yes >/dev/null 2>&1 || true
# Tool installs are best-effort: a transient registry failure should leave a
# usable container, and `make doctor` will say exactly what is missing.
mise install || printf 'WARNING: some tools failed to install; run `make doctor`\n' >&2

printf '==> Bootstrapping the repository\n'
eval "$(mise activate bash)" 2>/dev/null || true
make bootstrap || true

printf '\n'
make doctor || true
