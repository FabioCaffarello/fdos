#!/usr/bin/env bash
#
# Fitness function: no compiled executable is tracked.
#
# A committed binary escapes §9 entirely. Nothing says which source produced it,
# which toolchain built it, or which platform it runs on; `make repro-check`
# cannot compare it against anything, and the checksum in the diff is the only
# record it was ever the right bytes.
#
# `examples/ingest/ingest` was tracked for months: 7.3 MB, `Mach-O 64-bit
# executable arm64` — one developer's local build, in the directory whose job is
# to show a third party what conformance looks like, and absent from the file
# table in `examples/README.md` that documents the kit's deliverables (#79).
#
# It was also live: an ordinary `go build ./...` in that directory overwrites it
# silently, which happened during the work that added `examples/` to the gate.
# So it was not inert history — it was a 7.3 MB diff waiting for whoever ran the
# most obvious command in the module.
#
# Detection is by magic bytes rather than by the executable bit or by extension.
# Every script here is executable and none of them is a compiled artifact, and a
# binary committed without a suffix is exactly the case a name-based check
# misses.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

failures=0

printf 'Verifying no compiled executable is tracked...\n'

# First four bytes of each known executable image format. `cafebabe` is Mach-O
# universal and also a Java class file; neither belongs in this repository, so
# the ambiguity costs nothing.
is_executable_image() {
  local magic
  magic="$(od -An -tx1 -N4 "$1" 2>/dev/null | tr -d ' \n')"
  case "$magic" in
    7f454c46) printf 'ELF' ;;                                  # Linux
    cffaedfe|cefaedfe|feedface|feedfacf) printf 'Mach-O' ;;     # macOS
    cafebabe) printf 'Mach-O universal or Java class' ;;
    4d5a*) printf 'PE' ;;                                      # Windows
    *) return 1 ;;
  esac
}

checked=0
while IFS= read -r path; do
  [ -n "$path" ] || continue
  [ -f "$path" ] || continue
  checked=$((checked + 1))

  if kind="$(is_executable_image "$path")"; then
    size="$(wc -c < "$path" | tr -d ' ')"
    printf '  %s: tracked %s image, %s bytes\n' "$path" "$kind" "$size" >&2
    failures=$((failures + 1))
  fi
done < <(git ls-files)

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d compiled executable(s) tracked, of %d files.\n' "$failures" "$checked" >&2
  printf 'Build output belongs in bin/ or dist/, which are ignored. If one of\n' >&2
  printf 'these is a deliberate fixture, it needs a reason nobody has written yet.\n' >&2
  exit 1
fi

printf 'OK: %d tracked files, none a compiled executable.\n' "$checked"
