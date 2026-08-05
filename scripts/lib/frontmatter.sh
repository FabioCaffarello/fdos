#!/usr/bin/env bash
# Minimal YAML front-matter reader.
#
# Deliberately dependency-free: the directory-contract check is the first
# fitness function in the repository and runs from commit #1, before any
# language toolchain or package manager exists. It must never be the reason a
# clean clone fails to verify.
#
# Scope: top-level scalar keys and top-level list keys. That is the whole of
# the contract schema. If the schema ever needs more than this, it has stopped
# being a contract and become a configuration file.

set -euo pipefail

# fm_has_frontmatter <file>
# Succeeds if the file opens with a `---` delimiter and closes it.
fm_has_frontmatter() {
  awk '
    NR == 1 { if ($0 != "---") exit 1; next }
    /^---[[:space:]]*$/ { found = 1; exit 0 }
    END { if (!found) exit 1 }
  ' "$1"
}

# fm_keys <file>
# Prints every top-level key inside the front matter, one per line.
fm_keys() {
  awk '
    NR == 1 { next }
    /^---[[:space:]]*$/ { exit 0 }
    /^[A-Za-z_][A-Za-z0-9_]*:/ {
      key = $0
      sub(/:.*$/, "", key)
      print key
    }
  ' "$1"
}

# fm_value <file> <key>
# Prints the scalar value of a top-level key, with surrounding quotes stripped.
# Prints nothing when the key is absent or is a list.
fm_value() {
  awk -v want="$2" '
    NR == 1 { next }
    /^---[[:space:]]*$/ { exit 0 }
    /^[A-Za-z_][A-Za-z0-9_]*:/ {
      key = $0
      sub(/:.*$/, "", key)
      if (key != want) next
      val = $0
      sub(/^[^:]*:[[:space:]]*/, "", val)
      gsub(/^["'"'"']|["'"'"']$/, "", val)
      print val
      exit 0
    }
  ' "$1"
}

# fm_list_count <file> <key>
# Prints the number of `- ` items belonging to a top-level list key.
fm_list_count() {
  awk -v want="$2" '
    NR == 1 { next }
    # `exit` still runs END, so the count is printed there and only there.
    /^---[[:space:]]*$/ { exit 0 }
    /^[A-Za-z_][A-Za-z0-9_]*:/ {
      key = $0
      sub(/:.*$/, "", key)
      inlist = (key == want)
      next
    }
    inlist && /^[[:space:]]+-[[:space:]]+/ { n++ }
    END { print n + 0 }
  ' "$1"
}

# fm_list_items <file> <key>
# Prints each `- ` item of a top-level list key, one per line, unquoted.
fm_list_items() {
  awk -v want="$2" '
    NR == 1 { next }
    /^---[[:space:]]*$/ { exit 0 }
    /^[A-Za-z_][A-Za-z0-9_]*:/ {
      key = $0
      sub(/:.*$/, "", key)
      inlist = (key == want)
      next
    }
    inlist && /^[[:space:]]+-[[:space:]]+/ {
      item = $0
      sub(/^[[:space:]]+-[[:space:]]+/, "", item)
      gsub(/^["'"'"']|["'"'"']$/, "", item)
      gsub(/[[:space:]]+$/, "", item)
      if (item != "") print item
    }
  ' "$1"
}

# fm_require_keys <file> <label> <key>...
# Reports every missing required key. Returns 1 if any are missing.
fm_require_keys() {
  local file="$1" label="$2"
  shift 2
  local present missing=0 key

  if ! fm_has_frontmatter "$file"; then
    printf '  %s: missing or unterminated YAML front matter\n' "$label" >&2
    return 1
  fi

  present="$(fm_keys "$file")"
  for key in "$@"; do
    if ! printf '%s\n' "$present" | grep -qx -- "$key"; then
      printf '  %s: missing required key `%s`\n' "$label" "$key" >&2
      missing=1
    fi
  done
  return "$missing"
}
