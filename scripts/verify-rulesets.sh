#!/usr/bin/env bash
#
# Compare the repository's live protection settings against what is committed.
#
#   make ruleset-check
#
# `docs/branch-protection.md` has stated the gap since it was written:
#
#   "Nothing checks that the live rulesets match this document. They are
#    repository state, not files: someone can change them in the UI with no
#    commit here and nothing would notice."
#
# # Why this runs locally and deliberately not in CI
#
# Reading rulesets needs an admin-scoped token. ADR-0014 declined to put one in
# CI — "it needs an admin token, which is its own risk" — and ADR-0020 recorded
# the consequence as an open gap. That objection is about CI, not about checking:
# run from the maintainer's own authenticated CLI it needs no new credential and
# grants nothing to a workflow. It is the same argument `branch-protection.md`
# used to apply the rulesets by hand rather than from a workflow.
#
# So this is rung 6 from CI's perspective and rung 3 from a maintainer's, and
# saying that plainly is better than a check in CI that needs a token nobody
# wanted to create.
#
# It is invoked by `make doctor`, because a check nobody runs is not a check.
#
# Enforcement ladder position: local (rung 3 when run, 6 in CI — see ADR-0048).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

COMMITTED=".github/rulesets"
REPO="${REPO:-FabioCaffarello/fdos}"
failures=0

printf 'Verifying live protection against %s...\n' "$COMMITTED"

if ! command -v gh >/dev/null 2>&1; then
  printf '  gh is not installed — live settings could not be read\n'
  printf '  This check needs the forge. Nothing was verified.\n'
  exit 2
fi

if [ ! -d "$COMMITTED" ]; then
  printf '\nFAIL: %s is missing.\n' "$COMMITTED" >&2
  exit 1
fi

# Normalise identically on both sides, so an ordering difference from the API is
# not reported as drift. Only the fields that decide behaviour are compared:
# ids, timestamps and URLs change without anything changing.
normalise_ruleset() {
  python3 -c '
import json,sys
d=json.load(sys.stdin)
out={"name":d["name"],"target":d["target"],"enforcement":d["enforcement"],
     "conditions":d.get("conditions",{}),
     "rules":sorted([{k:v for k,v in r.items() if k in ("type","parameters")} for r in d["rules"]], key=lambda r:r["type"])}
print(json.dumps(out,indent=2,sort_keys=True))
'
}

normalise_environment() {
  python3 -c '
import json,sys
d=json.load(sys.stdin)
out={"name":d["name"],"deployment_branch_policy":d.get("deployment_branch_policy"),
     "protection_rules":sorted([r.get("type") for r in d.get("protection_rules",[])])}
print(json.dumps(out,indent=2,sort_keys=True))
'
}

live_ids="$(gh api "repos/${REPO}/rulesets" --jq '.[].id' 2>/dev/null || true)"
if [ -z "$live_ids" ]; then
  printf '  the rulesets endpoint returned nothing\n'
  printf '  Either none are configured, or this token lacks admin scope. Both are\n'
  printf '  worth knowing and neither is verified — refusing to report success.\n'
  exit 2
fi

seen=""
while IFS= read -r id; do
  [ -n "$id" ] || continue
  live="$(gh api "repos/${REPO}/rulesets/${id}" | normalise_ruleset)"
  name="$(printf '%s' "$live" | python3 -c 'import json,sys; print(json.load(sys.stdin)["name"])')"
  seen="${seen}${name} "

  file="${COMMITTED}/${name}.json"
  if [ ! -f "$file" ]; then
    printf '  %s: live ruleset with no committed definition\n' "$name" >&2
    failures=$((failures + 1))
    continue
  fi

  if diff -u "$file" <(printf '%s\n' "$live") > /tmp/ruleset-diff.$$ 2>&1; then
    printf '  %-22s matches\n' "$name"
  else
    printf '  %s: live settings differ from %s\n' "$name" "$file" >&2
    sed 's/^/      /' /tmp/ruleset-diff.$$ >&2
    failures=$((failures + 1))
  fi
  rm -f /tmp/ruleset-diff.$$
done <<EOF
$live_ids
EOF

for file in "${COMMITTED}"/*.json; do
  [ -f "$file" ] || continue
  name="$(basename "$file" .json)"
  case "$name" in environment-*) continue ;; esac
  case " ${seen} " in
    *" ${name} "*) ;;
    *)
      printf '  %s: committed, and not applied to the repository\n' "$name" >&2
      failures=$((failures + 1))
      ;;
  esac
done

# The release environment is where `contents: write` lives (ADR-0046), so it is
# worth the same treatment as a ruleset.
env_file="${COMMITTED}/environment-release.json"
if [ -f "$env_file" ]; then
  live_env="$(gh api "repos/${REPO}/environments/release" 2>/dev/null | normalise_environment || true)"
  if [ -z "$live_env" ]; then
    printf '  release environment: not readable or not configured\n' >&2
    failures=$((failures + 1))
  elif diff -u "$env_file" <(printf '%s\n' "$live_env") > /tmp/env-diff.$$ 2>&1; then
    printf '  %-22s matches\n' "release environment"
  else
    printf '  release environment: live settings differ from %s\n' "$env_file" >&2
    sed 's/^/      /' /tmp/env-diff.$$ >&2
    failures=$((failures + 1))
  fi
  rm -f /tmp/env-diff.$$
fi

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d protection setting(s) drifted.\n' "$failures" >&2
  printf 'The live settings are what actually gate merges and releases. If the\n' >&2
  printf 'change was deliberate, commit the new JSON and say why in\n' >&2
  printf 'docs/branch-protection.md; if it was not, restore it.\n' >&2
  exit 1
fi

printf 'OK: live protection matches what is committed.\n'
