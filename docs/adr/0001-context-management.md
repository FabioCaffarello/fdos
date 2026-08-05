---
id: ADR-0001
title: FDOS adopts .dotcontext as the canonical AI knowledge directory
status: Accepted
date: 2026-08-04
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0001 — FDOS adopts `.dotcontext` as the canonical AI knowledge directory

## Context

FDOS is developed with substantial AI assistance, and the quality of that
assistance is bounded by the quality of the structured knowledge available to
it. That knowledge — engineering constitution, architecture, playbooks, agent
definitions, skills — needs a canonical home.

The tooling default is `.context`. Sibling repositories in this workspace use
`.context`. Choosing otherwise means diverging from a convention that already
exists.

The counter-argument is about what the directory *is*. `.context` reads as
generic configuration and will accrete generic configuration. The directory does
not hold application settings; it holds structured engineering knowledge, and
that distinction is the reason it is worth maintaining at all.

## Decision

FDOS adopts `.dotcontext/` as the canonical AI knowledge directory.

The directory represents structured engineering knowledge rather than
application configuration. The name is part of the project's identity and
signals that distinction to every reader, human or agent.

The dotcontext MCP server is configured to write to `.dotcontext` via its
`outputDir` parameter, so there is exactly one such directory and no divergence
between what the tooling produces and what the repository declares.

Tracking follows the tool's own classification:

| Path | Classification | Tracked |
|------|----------------|---------|
| `.dotcontext/docs/**` | versioned | yes |
| `.dotcontext/agents/**` | versioned | yes |
| `.dotcontext/skills/**` | versioned | yes |
| `.dotcontext/config/**` | versioned | yes |
| `.dotcontext/plans/**` | local | no |
| `.dotcontext/cache/**` | runtime | no |
| `.dotcontext/runtime/**` | runtime | no |

## Consequences

### Positive

- The name states what the directory is, and resists accreting unrelated
  configuration.
- Knowledge intended for agents is unambiguously separated from settings
  intended for tools.

### Negative

- Divergence from the tooling default and from sibling repositories. Every new
  contributor and every generic tool integration must be told.
- The `outputDir` parameter must be passed on every scaffolding operation. If it
  is ever omitted, a stray `.context/` appears. A guard for this belongs in
  `make verify` and is deliberately deferred to M1, when `.dotcontext` is first
  populated.

### Enforcement

Rung 5 (documentation) today. Climbing to rung 3 in M1: `make verify` will fail
if a `.context/` directory exists.

## Alternatives considered

**`.context/`.** Rejected on identity and semantic grounds above. The practical
objection to `.dotcontext` — that the tooling would produce a second directory —
does not hold, because `outputDir` is a first-class parameter.

**Both, with `.context` as a symlink.** Rejected: two names for one thing is the
worst outcome. It guarantees that half the tooling and half the documentation
will reference the name FDOS did not choose.
