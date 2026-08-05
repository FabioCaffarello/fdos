---
directory: deploy
purpose: Deployment topology and runtime environment definitions.
owner: "@FabioCaffarello"
allowed:
  - Container image definitions
  - Local development stacks (compose files)
  - Kubernetes manifests and configuration overlays
  - Environment templates containing no secret values
forbidden:
  - Secrets, credentials, tokens or private keys of any kind
  - Business logic or financial calculations
  - Configuration that changes behaviour a test cannot reproduce
  - Environment-specific values baked into images
---

# deploy

Deployment topology for FDOS: how the applications in `apps/` are packaged, run
and connected.

## Status: empty by design

`deploy/` contains nothing at M0. There are no applications to deploy until
`apps/` is populated, which follows M2.

## Infrastructure is replaceable

Constitution §10 requires that the domain never depend on infrastructure.
`deploy/` is where that promise is either kept or quietly broken.

The test is simple and should be applied to every addition: **could this
component be replaced without changing anything in `libs/`?** If replacing the
message broker requires touching a domain module, the boundary has already
failed and the fix belongs in `libs/`, not here.

## Configuration must not change behaviour

Configuration selects *which* infrastructure is used. It must never select which
business rules apply. A deployment flag that changes a calculation makes that
calculation unreproducible from the ledger alone, which violates Constitution §9
directly — the report can no longer be regenerated years later without also
knowing the deployment state at the time.

## Secrets

No secret value is ever committed, in any form, including examples that look
like real values. `.env.example` files carry placeholders only. From M3,
`gitleaks` enforces this in CI; until then it is convention, and convention is
rung 6.
