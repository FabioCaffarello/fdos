SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

GO ?= go
SCRIPTS_DIR := scripts

# Dependencies are never resolved implicitly. An implicit `go mod tidy` during a
# build is a silent, unreviewed change to the dependency graph (mise.toml states
# the same; this makes it true whether or not mise is installed).
export GOFLAGS := -mod=readonly

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

.PHONY: help bootstrap verify \
	toolchain-check contracts-check adr-check rfc-check constitution-check \
	fmt fmt-check vet lint test analyze repro-check tidy tidy-check build clean

help: ## Show available targets
	@printf 'FDOS — Financial Data Operating System\n\n'
	@printf 'Current milestone: M2 (Determinism Toolchain)\n\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@printf '\n'

bootstrap: ## Prepare a working copy for development
	@printf '==> Bootstrapping FDOS\n'
	@$(MAKE) --no-print-directory toolchain-check
	@printf '\nBootstrap complete. Run `make verify` to check the repository.\n'

verify: toolchain-check contracts-check adr-check rfc-check constitution-check \
        tidy-check fmt-check vet lint test analyze repro-check ## Run every enforcement mechanism available at this milestone
	@printf '\nAll checks passed.\n'

# ---------------------------------------------------------------------------
# Governance
# ---------------------------------------------------------------------------

toolchain-check: ## Assert the installed toolchain matches the pins in mise.toml
	@$(SCRIPTS_DIR)/toolchain-check.sh

contracts-check: ## Assert every directory declares a valid architectural contract
	@$(SCRIPTS_DIR)/verify-directory-contracts.sh

adr-check: ## Assert the decision log is well-formed and append-only
	@$(SCRIPTS_DIR)/verify-adr.sh

rfc-check: ## Assert the RFC set is well-formed and accepted RFCs produced ADRs
	@$(SCRIPTS_DIR)/verify-rfc.sh

constitution-check: ## Assert every principle appears in the §15 enforcement table
	@$(SCRIPTS_DIR)/verify-constitution-coverage.sh

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

test: ## Run all tests with the race detector
	$(call FOR_EACH_MODULE,$(GO) test -race ./...)

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
