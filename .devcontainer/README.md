---
directory: .devcontainer
purpose: Reproducible development container, built from the same toolchain pins as everything else.
owner: "@FabioCaffarello"
allowed:
  - Container definition and its post-create script
  - Editor customisations that apply only inside the container
forbidden:
  - Tool versions declared here instead of read from mise.toml
  - Build steps that produce a released artifact
  - Secrets, credentials or tokens
  - Anything CI depends on
---

# .devcontainer

A ready development environment: clone, open in a container, and
`make verify` passes without installing anything by hand.

## No version is declared here

The container installs `mise`, and `mise` installs the toolchain from
`mise.toml`. That file is the single source of truth (ADR-0014) and this
directory reads it like `make toolchain-check` and CI do.

Devcontainer *features* for Go were deliberately not used: a feature declares
its own version, which would make this a second place tool versions live — and
the two would drift.

If a script here ever needs to know a version, that is the signal it has become
a second source of truth. Read `mise.toml` instead.

## Not a build input

**CI does not use this container.** Released artifacts are built by
`release.yml` on a GitHub runner, from source, with every action pinned by
commit SHA.

That distinction is why the image here is pinned by tag rather than by digest,
and why `curl | sh` for the `mise` installer is tolerable in this one place:
nothing produced in this container is ever attested to or released. Applying the
same rule everywhere would be consistent but would confuse a convenience with a
supply chain.

If that ever stops being true — if a release is ever built here — this
directory becomes a build input and must be digest-pinned like every other one.

## `GOWORK` is not set to `off`

Every `make` target and every CI job sets `GOWORK=off`, because module
resolution must go through published versions (ADR-0004).

The editor is the deliberate exception: `go.work` exists precisely so that
cross-module navigation works while editing. Setting `GOWORK=off` in the
container environment would break that for no gain, since every command that
matters overrides it anyway.

## Verifying

```sh
make doctor    # what is installed, what is missing, and what to do about it
make verify    # the full gate
```

A container where `make verify` does not pass is a bug in this directory.
