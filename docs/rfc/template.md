---
id: RFC-NNNN
title: Short statement of what is being proposed
status: Draft
date: YYYY-MM-DD
authors:
  - "@handle"
---

# RFC-NNNN — Short statement of what is being proposed

> An RFC is for decisions that require *design exploration* before they can be
> decided. If the decision is already clear, write an ADR instead. If this RFC
> is accepted, it is followed by one or more ADRs recording what was decided.

## Summary

One paragraph. What is being proposed and why it cannot be settled with an ADR
alone.

## Motivation

What breaks, or what becomes impossible, if this is not resolved? Which
Constitution principle is at stake?

State plainly whether this is retrofittable. Some decisions — reference-data
versioning, provenance shape, bitemporal scope — are permanently lost if
deferred past the first event schema. Say so if that applies.

## Design

The proposal in detail. Models, boundaries, types, schemas, failure modes.

Include what this does *not* cover. A design whose scope is unstated will be
assumed to cover everything.

## Enforcement

Which rung of the enforcement ladder (docs/constitution.md §15) this design can
be held at, and by what mechanism. A design that can only be enforced by human
discipline should say so explicitly and justify why nothing higher is feasible.

## Alternatives

At least two, each with the specific reason it is not being proposed. If no
genuine alternative exists, the problem is probably narrower than an RFC.

## Prior art

How comparable systems solved this. Financial infrastructure is old; most of
these problems have been solved badly at least once and well at least once.

## Open questions

What this RFC deliberately leaves unresolved, and who resolves it.

## Consequences

What becomes easier. What becomes harder. What becomes impossible.
