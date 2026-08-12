SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

GO ?= go
SCRIPTS_DIR := scripts

# Dependencies are never resolved implicitly. An implicit `go mod tidy` during a
# build is a silent, unreviewed change to the dependency graph (mise.toml states
# the same; this makes it true whether or not mise is installed).
export GOFLAGS := -mod=readonly

# cgo is off, and that is a supply-chain decision rather than a preference
# (ADR-0035). A cgo dependency makes the build depend on the host C toolchain,
# which puts the byte-reproducibility `make repro-check` asserts at the mercy of
# a system compiler nobody pinned.
#
# The tree was measured as cgo-free before this was pinned, so this preserves a
# property rather than imposing one — and turns a future dependency that quietly
# needs cgo into a build failure instead of a silent loss of reproducibility.
export CGO_ENABLED := 0

# ADR-0004 makes each libs/* an independent module, so Go commands run per
# module rather than once at the root.
#
# GOWORK=off is the load-bearing half of ADR-0004: it forces module resolution
# through published versions instead of local workspace paths. Without it the
# open-core boundary silently stops being verified, with nothing to indicate it
# has stopped.
define FOR_EACH_MODULE
	@set -euo pipefail; \
	for module in $$($(SCRIPTS_DIR)/list-modules.sh); do \
		printf '>>> %s\n' "$$module"; \
		( cd "$$module" && GOWORK=off $(1) ); \
	done
endef

.PHONY: help bootstrap hooks doctor verify affected \
	toolchain-check contracts-check adr-check adr-immutability-check rfc-check constitution-check action-pinning-check \
	context-check agent-contract-check proto-check proto-gen proto-lint proto-breaking consumer-check \
	fmt fmt-check vet lint test analyze repro-check tidy tidy-check build clean \
	secrets-check secrets-check-staged vuln-check commit-msg-check commit-msg-check-file

help: ## Show available targets
	@printf 'FDOS — Financial Data Operating System\n\n'
	@printf 'New here?  make doctor   diagnose this working copy\n'
	@printf '           make verify   the full gate (exactly what CI runs)\n\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@printf '\n'

bootstrap: ## Prepare a working copy for development
	@printf '==> Bootstrapping FDOS\n'
	@$(MAKE) --no-print-directory toolchain-check
	@$(MAKE) --no-print-directory hooks
	@printf '\nBootstrap complete. Run `make verify` to check the repository.\n'

doctor: ## Diagnose this working copy and say what to fix
	@$(SCRIPTS_DIR)/doctor.sh

hooks: ## Install the git hooks (lefthook)
	@if command -v lefthook >/dev/null 2>&1; then \
		lefthook install >/dev/null && printf 'Git hooks installed.\n'; \
	else \
		printf 'lefthook not installed — hooks skipped. See mise.toml.\n'; \
	fi

verify: toolchain-check contracts-check adr-check adr-immutability-check rfc-check constitution-check \
        action-pinning-check context-check agent-contract-check proto-check \
        secrets-check tidy-check fmt-check vet lint test analyze vuln-check repro-check ## Run every enforcement mechanism available at this milestone
	@printf '\nAll checks passed.\n'

affected: ## Print the modules affected by the current change
	@$(SCRIPTS_DIR)/affected-modules.sh $(BASE)

# ---------------------------------------------------------------------------
# Governance
# ---------------------------------------------------------------------------

toolchain-check: ## Assert the installed toolchain matches the pins in mise.toml
	@$(SCRIPTS_DIR)/toolchain-check.sh

contracts-check: ## Assert every directory declares a valid architectural contract
	@$(SCRIPTS_DIR)/verify-directory-contracts.sh

adr-check: ## Assert the decision log is well-formed and append-only
	@$(SCRIPTS_DIR)/verify-adr.sh

adr-immutability-check: ## Assert no accepted ADR has been rewritten
	@$(SCRIPTS_DIR)/verify-adr-immutability.sh

rfc-check: ## Assert the RFC set is well-formed and accepted RFCs produced ADRs
	@$(SCRIPTS_DIR)/verify-rfc.sh

constitution-check: ## Assert every principle appears in the §15 enforcement table
	@$(SCRIPTS_DIR)/verify-constitution-coverage.sh

action-pinning-check: ## Assert every GitHub Action is pinned to a commit SHA
	@$(SCRIPTS_DIR)/verify-action-pinning.sh

context-check: ## Assert documentation describes the repository that exists
	@$(SCRIPTS_DIR)/verify-doc-references.sh

agent-contract-check: ## Assert agent playbooks declare a valid prompt contract
	@$(SCRIPTS_DIR)/verify-agent-contracts.sh

# ---------------------------------------------------------------------------
# Contracts
# ---------------------------------------------------------------------------

proto-check: ## Assert the contract surface is valid, compatible and unchanged
	@$(SCRIPTS_DIR)/verify-proto.sh

proto-gen: ## Regenerate Go from the proto schemas
	@buf generate && printf 'Generated. Review the diff before committing.\n'

proto-lint: ## Lint the proto schemas
	@buf lint && printf 'Schemas lint clean.\n'

proto-breaking: ## Check the contract surface for breaking changes against main
	@buf breaking --against '.git#branch=main'

# Deliberately NOT in `verify`: it depends on a published tag having propagated
# to a third-party proxy, which is latency the per-commit gate must not inherit
# (ADR-0018 made that mistake once with a remote codegen plugin). Runs at
# release and on demand.
consumer-check: ## Prove the published contract module is consumable from outside
	@$(SCRIPTS_DIR)/verify-consumer.sh $(VERSION)

# ---------------------------------------------------------------------------
# Security and supply chain
# ---------------------------------------------------------------------------

secrets-check: ## Scan the full history for committed secrets
	@$(SCRIPTS_DIR)/verify-secrets.sh history

secrets-check-staged: ## Scan staged changes for secrets (used by the pre-commit hook)
	@$(SCRIPTS_DIR)/verify-secrets.sh staged

# Deliberately NOT in `verify`, and the reason is measured rather than cautious.
# Nine of the last sixty commits on main violate the 72-character limit because
# GitHub's squash-merge appends ` (#NN)` to a subject that was compliant when it
# was written, and four more predate the convention. Wiring this into the gate
# would fail on main today, for commits nobody can now amend (Constitution §4).
#
# This target ranges over `origin/main..HEAD` — the author's own commits, before
# the forge rewrites them — which is the only range where a failure is both fair
# and actionable. Raising it to a gate is an enforcement-ladder change and owes
# an ADR (issue #109).
commit-msg-check: ## Assert this branch's commit messages follow the convention
	@$(SCRIPTS_DIR)/verify-commit-message.sh branch $(BASE)

commit-msg-check-file: ## Assert one message file follows the convention (used by the commit-msg hook)
	@$(SCRIPTS_DIR)/verify-commit-message.sh message $(MSG)

vuln-check: ## Assert no known vulnerability is reachable from FDOS code
	@$(SCRIPTS_DIR)/verify-vulns.sh

# ---------------------------------------------------------------------------
# Go
# ---------------------------------------------------------------------------

fmt: ## Format all Go code
	$(call FOR_EACH_MODULE,$(GO) fmt ./...)

fmt-check: ## Assert all Go code is formatted
	@$(SCRIPTS_DIR)/verify-gofmt.sh

vet: ## Run go vet across all modules
	$(call FOR_EACH_MODULE,$(GO) vet ./...)

lint: ## Run golangci-lint across all modules
	$(call FOR_EACH_MODULE,golangci-lint run ./...)

# The race detector is the one place cgo is enabled, and it has to be.
#
# `-race` requires cgo on linux/amd64 — where CI runs — and does not on
# darwin/arm64, where this was written. So the CGO_ENABLED=0 pin above passed
# locally and failed in CI with `-race requires cgo`, which is exactly the
# divergence ADR-0014 says a check must not have. A developer machine could not
# reproduce it at all.
#
# Enabling cgo here does not weaken ADR-0035. That pin protects the *build* —
# `repro-check`, releases, `vet`, `lint` all still run cgo-free, and a dependency
# that quietly needed cgo would still fail them. A test binary is not a released
# artifact and its reproducibility is not what anyone audits.
test: ## Run all tests with the race detector
	$(call FOR_EACH_MODULE,CGO_ENABLED=1 $(GO) test -race ./...)

analyze: ## Enforce domain purity and layer boundaries (FDOS analysers)
	@$(SCRIPTS_DIR)/run-analyzers.sh

repro-check: ## Assert every command builds byte-reproducibly
	@$(SCRIPTS_DIR)/verify-reproducible-build.sh

tidy: ## Tidy every module's dependencies
	$(call FOR_EACH_MODULE,$(GO) mod tidy)

tidy-check: ## Assert go.mod and go.sum are tidy
	@$(SCRIPTS_DIR)/verify-tidy.sh

build: ## Build all commands into bin/
	@mkdir -p bin
	$(call FOR_EACH_MODULE,$(GO) build -trimpath -o "$(CURDIR)/bin/" ./...)

clean: ## Remove build output
	@rm -rf bin dist
	@printf 'Cleaned.\n'
