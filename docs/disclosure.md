# Disclosure register

What this public repository has revealed about the private side of the
ecosystem, when, and what of it is permanent.

Kept because a redaction without a record reads as though nothing was ever
disclosed. Anyone assessing what is public needs the list, not the absence of
one — and the tags and history are public regardless of what the current tree
says.

## The rule

`fdos` is public. The acquisition side is private and commercial. Nothing
private leaks in: no provider name, no credential shape, no session mechanic,
no evasion technique, no reference to a private module path — not in code,
tests, decisions, fixtures, or commit messages. Provider identity enters `fdos`
only as opaque provenance metadata (`fdos.kernel.v1.SourceRef`).

The private repository's *own name* is not covered: the responsibility matrix
has to say who owns what, and a boundary with an anonymous owner cannot be
checked.

## D-001 — Provider names in the Tier-0 boundary matrix

**Disclosed:** two financial institutions and one central bank, named as
examples of what the private side integrates, plus a naming convention for
private plugin repositories.

| Where | Since | State |
|---|---|---|
| `docs/ecosystem/boundary.md` — matrix row and the D2 text | `ecosystem/v0.1.0`, 2026-08-05 | **redacted** at `v0.3.0` |
| tags `ecosystem/v0.1.0`, `ecosystem/v0.2.0` | 2026-08-05 | **permanent** — the `release-tags` ruleset refuses deletion |
| git history | 2026-08-05 | **permanent** |
| the vendored copy downstream | 2026-08-05 | carries the old text until the consumer re-syncs to `v0.3.0` |

**How it happened.** The text came from the engineering brief's own Tier-0
block, which instructed verbatim reproduction and explicitly forbade paraphrase.
Tier 0 exists so a vendored rule cannot be improved by whoever is editing; the
same property meant a defect inside it could not be fixed on the way in. It was
published, listed as a defect, and amended by the Tier-0 procedure — which is
the procedure working slowly rather than failing.

**Severity, assessed rather than assumed.** No credential, no session mechanic,
no captcha or OTP handling, no evasion or anti-detect detail, no provider
markup or endpoint. What was revealed is *commercial* — which institutions the
paid side targets — and structural. A competitor learns that a Brazilian
financial platform integrates the national exchange and central bank, which is
close to the minimum inference available about any such platform.

That is an argument about proportion, not a reason it was fine.

## D-002 — Private module and package identifiers

**Disclosed:** module and proto-package names belonging to the private
repository.

**Redacted from current documents** — `docs/blocked.md`,
`docs/ecosystem/roadmap.md`, `docs/ecosystem/boundary.md`.

**Retained in historical documents** — `fdos:RFC-0008` and `fdos:ADR-0026`,
where those identifiers are the *subject* of the decision. Removing them would
leave a decision whose reasoning cannot be read, and an accepted decision is
superseded rather than rewritten (`fdos:ADR-0000`). Both carry a banner pointing
here.

That line is a judgement: identifiers are redacted where they are *incidental*
and retained where they are *load-bearing*. Stated so the next person applies
the same one rather than inventing a stricter or looser one.

## What no mechanism can check

This register is **rung 6, and structurally cannot climb.**

A check for "no provider is named here" needs a list of provider names to match
against. That list, committed to a public repository, *is* the disclosure — in a
more complete and more machine-readable form than the leak it was written to
prevent. The mechanism would cost more than the defect.

What can be checked mechanically is the narrower, uninteresting half: that no
private module path appears in the current tree. That would not have caught
D-001.

So this stays a review obligation, and the honest statement is that **the next
disclosure will be found by a person or not at all.**

## Method note

Facts recorded in this repository about the private side — in `B-001`, in the
D5 evidence, and in the consumer milestone table in `roadmap.md` — were verified
by reading that repository's committed manifests through the GitHub API, at a
time when that was the sanctioned coordination channel. It no longer is.

Those facts stand; the method is closed. Anything asserted here about the
private side from now on comes from an artifact it published to this repository
— an issue, a pull request, or a contract — or it is not asserted.
