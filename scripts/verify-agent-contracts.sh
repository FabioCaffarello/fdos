#!/usr/bin/env bash
#
# Fitness function: agent playbooks declare a prompt contract, and the rosters
# match what is on disk.
#
# A "prompt contract" here is not a template for phrasing. It is the subset of
# an agent's obligations that can be stated as data and checked: what it reads
# before acting, what is out of bounds, and what evidence it produces (ADR-0015).
#
# What this CANNOT check, stated plainly because the gap matters: whether an
# agent actually reads `must_read`, or honours `must_not`. The fields make the
# obligation explicit and reviewable; they do not enforce it. A green run here
# is not evidence of agent compliance.
#
# Enforcement ladder position: CI (see ADR-0005).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=lib/frontmatter.sh
source "${ROOT}/scripts/lib/frontmatter.sh"
cd "$ROOT"

AGENTS_DIR=".context/agents"
SKILLS_DIR=".context/skills"
REQUIRED_AGENT_KEYS="type name description agentType status must_read must_not evidence"

failures=0
agents=0
skills=0

fail() {
  printf '  %s\n' "$1" >&2
  failures=$((failures + 1))
}

printf 'Verifying agent prompt contracts...\n'

# --- no scaffold left unfilled anywhere under .context ------------------------
#
# Scans the filesystem, not the git index. A freshly generated scaffold is
# untracked, and untracked is precisely the state this check exists to catch —
# `git ls-files` would skip exactly the files that matter.
while IFS= read -r doc; do
  [ -f "$doc" ] || continue
  if [ "$(fm_value "$doc" status || true)" = "unfilled" ]; then
    fail "${doc}: still marked 'unfilled' — a scaffold that describes nothing is worse than an absent file"
  fi
done < <(find .context -name '*.md' -not -path '*/cache/*' -not -path '*/runtime/*' 2>/dev/null || true)

# --- agent playbooks ----------------------------------------------------------
for agent in "${AGENTS_DIR}"/*.md; do
  [ -f "$agent" ] || continue
  name="$(basename "$agent")"
  [ "$name" = "README.md" ] && continue
  agents=$((agents + 1))

  # shellcheck disable=SC2086
  fm_require_keys "$agent" "$agent" ${REQUIRED_AGENT_KEYS} || failures=$((failures + 1))

  for list_key in must_read must_not evidence; do
    if [ "$(fm_list_count "$agent" "$list_key" || true)" -lt 1 ]; then
      fail "${agent}: '${list_key}' is empty — a contract that requires nothing is not a contract"
    fi
  done

  # Every path the agent is told to read must exist, or the instruction sends it
  # somewhere that is not there.
  while IFS= read -r path; do
    [ -n "$path" ] || continue
    if [ ! -e "$path" ]; then
      fail "${agent}: must_read names '${path}', which does not exist"
    fi
  done < <(fm_list_items "$agent" must_read || true)
done

# --- rosters match disk, in both directions -----------------------------------
roster_agents="$(grep -oE '\]\(\./[a-z0-9-]+\.md\)' "${AGENTS_DIR}/README.md" 2>/dev/null \
  | sed -E 's|\]\(\./(.*)\.md\)|\1|' | sort -u || true)"
disk_agents="$(find "$AGENTS_DIR" -maxdepth 1 -name '*.md' ! -name 'README.md' -exec basename {} .md \; | sort -u)"

while IFS= read -r a; do
  [ -n "$a" ] || continue
  printf '%s\n' "$roster_agents" | grep -qx -- "$a" \
    || fail "${AGENTS_DIR}/README.md: '${a}' exists on disk but is not listed"
done <<EOF
$disk_agents
EOF

while IFS= read -r a; do
  [ -n "$a" ] || continue
  printf '%s\n' "$disk_agents" | grep -qx -- "$a" \
    || fail "${AGENTS_DIR}/README.md: lists '${a}', which does not exist"
done <<EOF
$roster_agents
EOF

roster_skills="$(grep -oE '\]\(\./[a-z0-9-]+/SKILL\.md\)' "${SKILLS_DIR}/README.md" 2>/dev/null \
  | sed -E 's|\]\(\./(.*)/SKILL\.md\)|\1|' | sort -u || true)"
disk_skills="$(find "$SKILLS_DIR" -maxdepth 1 -mindepth 1 -type d -exec basename {} \; | sort -u)"

while IFS= read -r s; do
  [ -n "$s" ] || continue
  skills=$((skills + 1))
  [ -f "${SKILLS_DIR}/${s}/SKILL.md" ] || fail "${SKILLS_DIR}/${s}: no SKILL.md"
  printf '%s\n' "$roster_skills" | grep -qx -- "$s" \
    || fail "${SKILLS_DIR}/README.md: '${s}' exists on disk but is not listed"
done <<EOF
$disk_skills
EOF

while IFS= read -r s; do
  [ -n "$s" ] || continue
  printf '%s\n' "$disk_skills" | grep -qx -- "$s" \
    || fail "${SKILLS_DIR}/README.md: lists '${s}', which does not exist"
done <<EOF
$roster_skills
EOF

if [ "$failures" -gt 0 ]; then
  printf '\nFAIL: %d agent contract violation(s).\n' "$failures" >&2
  exit 1
fi

printf 'OK: %d agent contracts valid, %d skills, rosters match disk.\n' "$agents" "$skills"
