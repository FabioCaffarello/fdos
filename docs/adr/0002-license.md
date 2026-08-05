---
id: ADR-0002
title: The FDOS public core is licensed under Apache-2.0
status: Accepted
date: 2026-08-04
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0002 — The FDOS public core is licensed under Apache-2.0

## Context

FDOS follows an Open Core architecture (Constitution §13). The public core
contains the engineering platform, canonical models, ledger, SDKs, APIs and
testing infrastructure. Private repositories contain authenticated providers and
institution-specific connectors.

The licence of the public core determines who can adopt it, who can contribute,
and what legal exposure adopters carry. In a financial domain, calculation
methods are patentable subject matter, which makes patent treatment a material
concern rather than a formality.

## Decision

The FDOS public core is licensed under the Apache License, Version 2.0.

Private repositories are not covered by this licence and are licensed
separately. The `NOTICE` file states this boundary explicitly.

## Consequences

### Positive

- Explicit patent grant (§3) and retaliation clause. For software implementing
  financial calculation methods this is the decisive property, and it is exactly
  what MIT and BSD lack.
- Low adoption friction: Apache-2.0 is pre-approved at most institutions,
  removing a legal review that would otherwise block evaluation.
- Compatible with the Open Core model: the permissive core does not reach into
  separately licensed private repositories.

### Negative

- No protection against a competitor building a commercial product on the core.
  This is an accepted trade: the commercial position rests on the private
  connectors and operational surface, not on the core being closed.
- Apache-2.0 cannot be revoked for code already released. If the commercial
  strategy later requires a source-available licence, only new code can carry
  it, and the fork risk is real.
- §4(b) requires modified files to carry change notices — an obligation on
  downstream forks that occasionally surprises contributors.

### Enforcement

Rung 5 (documentation). `LICENSE` and `NOTICE` are present and CODEOWNERS-
protected. A licence-header check across source files is deferred to M3, when
source files exist.

## Alternatives considered

**BUSL-1.1.** Rejected. It would defend against commercial competition, but at
the cost of external adoption and contribution — and adoption is the point of
having a public core at all. BUSL is also unusual for domain libraries and would
itself trigger the legal review Apache-2.0 avoids.

**MIT.** Rejected. Simpler and more permissive, but no patent grant. For a
system whose value is concentrated in financial calculation methods, that
absence is a real exposure for adopters, not a theoretical one.

**AGPL-3.0.** Rejected. The network-use clause would force disclosure on
institutions running FDOS internally, which is precisely the deployment model
FDOS targets. It would eliminate the intended audience.
