SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

GO ?= go
SCRIPTS_DIR := scripts

# Milestone M0 delivers repository, governance and tooling only. There is no Go
# code yet: the layer structure (domain / app / adapters) is an output of the
# M1.5 canonical-architecture RFCs, and creating modules before that RFC lands
# would pre-judge it. `go.work`, `go.mod`, `fmt`, `lint`, `test` and `build`
# arrive in M2.

.PHONY: help bootstrap verify toolchain-check contracts-check adr-check constitution-check clean

help: ## Show available targets
	@printf 'FDOS — Financial Data Operating System\n\n'
	@printf 'Current milestone: M0 (Repository Genesis)\n\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@printf '\n'

bootstrap: ## Prepare a working copy for development
	@printf '==> Bootstrapping FDOS\n'
	@$(MAKE) --no-print-directory toolchain-check
	@printf '\nBootstrap complete. Run `make verify` to check the repository.\n'

verify: toolchain-check contracts-check adr-check constitution-check ## Run every enforcement mechanism available at this milestone
	@printf '\nAll checks passed.\n'

toolchain-check: ## Assert the installed toolchain matches the pins in mise.toml
	@$(SCRIPTS_DIR)/toolchain-check.sh

contracts-check: ## Assert every directory declares a valid architectural contract
	@$(SCRIPTS_DIR)/verify-directory-contracts.sh

adr-check: ## Assert the decision log is well-formed and append-only
	@$(SCRIPTS_DIR)/verify-adr.sh

constitution-check: ## Assert every principle appears in the §15 enforcement table
	@$(SCRIPTS_DIR)/verify-constitution-coverage.sh

clean: ## Remove build output
	@rm -rf bin dist
	@printf 'Cleaned.\n'
