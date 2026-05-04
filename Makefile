.DEFAULT_GOAL := help

SHELL := /usr/bin/env bash

# =============================================================================
# One-shot setup
# =============================================================================

.PHONY: setup
setup: setup-tools setup-go setup-py setup-web ## install all toolchains + per-app deps; run this first
	@printf "\n\033[32m✓\033[0m Toolchains and per-app deps installed.\n"
	@printf "  Next: \033[36mmake up\033[0m (docker infra) → \033[36mmake register-schemas\033[0m → \033[36mmake test\033[0m\n"

.PHONY: setup-tools
setup-tools: ## install go/python/node toolchains via mise (skipped if mise missing)
	@if command -v mise >/dev/null 2>&1; then \
		printf "\033[36m→\033[0m \033[1mmise install\033[0m (go / python / node from .tool-versions)\n"; \
		mise install || printf "\033[33m!\033[0m mise install reported errors — continuing with whatever's on PATH\n"; \
	else \
		printf "\033[33m!\033[0m mise not found — install from https://mise.jdx.dev (recommended).\n"; \
		printf "  Falling back to whatever go/python3/npm are on your PATH.\n"; \
	fi

.PHONY: setup-go
setup-go: ## download Go module deps for api-svc + indexer
	@printf "\033[36m→\033[0m apps/api-svc:    go mod download\n"
	@cd apps/api-svc && go mod download
	@printf "\033[36m→\033[0m apps/indexer:    go mod download\n"
	@cd apps/indexer && go mod download

.PHONY: setup-py
setup-py: ## create athlete-svc venv and install deps (uses mise python or system python3)
	@printf "\033[36m→\033[0m apps/athlete-svc: python venv + pip install\n"
	@cd apps/athlete-svc && \
		( [ -d .venv ] || python3 -m venv .venv ) && \
		.venv/bin/pip install -q --upgrade pip && \
		.venv/bin/pip install -q -e ".[dev]"

.PHONY: setup-web
setup-web: ## install web npm deps
	@printf "\033[36m→\033[0m apps/web:        npm install\n"
	@cd apps/web && npm install --silent

# =============================================================================
# Infrastructure
# =============================================================================

.PHONY: up
up: ## start docker infra (postgres, kafka, schema registry, opensearch)
	@docker compose up -d --wait
	@printf "\033[32m✓\033[0m Infra up. Run \033[36mmake register-schemas\033[0m next.\n"

.PHONY: down
down: ## stop docker infra (volumes preserved)
	@docker compose down

.PHONY: nuke
nuke: ## stop docker infra and DELETE volumes
	@docker compose down -v

.PHONY: ps
ps: ## show docker service status
	@docker compose ps

.PHONY: stop
stop: ## kill any app processes holding the per-app ports (8000-8003)
	@for p in 8000 8001 8002 8003; do \
		fuser -k $$p/tcp 2>/dev/null || true; \
	done
	@printf "\033[32m✓\033[0m app ports cleared (run \033[36mmake down\033[0m to stop docker infra too)\n"

# =============================================================================
# Schemas
# =============================================================================

.PHONY: register-schemas
register-schemas: ## POST every schemas/*.avsc to Schema Registry, set compat policy
	@bash scripts/register-schemas.sh

.PHONY: check-compat
check-compat: ## check whether current schemas/*.avsc are compatible with what's registered
	@bash scripts/check-compat.sh

# =============================================================================
# Seed data
# =============================================================================

.PHONY: seed
seed: ## POST a handful of camps via api-svc (api-svc + indexer should be running)
	@bash scripts/seed-camps.sh

# =============================================================================
# Per-app passthroughs
# =============================================================================

.PHONY: gen
gen: ## run code-gen across every app (oapi-codegen, ent, types)
	@$(MAKE) -C apps/api-svc gen
	@$(MAKE) -C apps/indexer gen 2>/dev/null || true
	@$(MAKE) -C apps/web gen 2>/dev/null || true

.PHONY: gen/api-svc gen/athlete-svc gen/indexer gen/web
gen/api-svc:     ## run code-gen for api-svc only (sqlc + oapi-codegen)
	@$(MAKE) -C apps/api-svc gen
gen/athlete-svc: ## run code-gen for athlete-svc only (no-op; here for parity)
	@$(MAKE) -C apps/athlete-svc gen
gen/indexer:     ## run code-gen for indexer only (no-op; here for parity)
	@$(MAKE) -C apps/indexer gen
gen/web:         ## run code-gen for web only (no-op; here for parity)
	@$(MAKE) -C apps/web gen

.PHONY: test
test: ## run every app's test suite
	@$(MAKE) -C apps/api-svc test
	@$(MAKE) -C apps/athlete-svc test
	@$(MAKE) -C apps/indexer test
	@$(MAKE) -C apps/web test

.PHONY: test/api-svc test/athlete-svc test/indexer test/web
test/api-svc:     ## run api-svc tests only
	@$(MAKE) -C apps/api-svc test
test/athlete-svc: ## run athlete-svc tests only
	@$(MAKE) -C apps/athlete-svc test
test/indexer:     ## run indexer tests only
	@$(MAKE) -C apps/indexer test
test/web:         ## run web tests only
	@$(MAKE) -C apps/web test

# =============================================================================
# Run apps locally (each in the foreground; ^C to stop)
# =============================================================================

.PHONY: api-svc
api-svc: ## run apps/api-svc (Go: REST + Camp publisher)
	@$(MAKE) -C apps/api-svc run

.PHONY: athlete-svc
athlete-svc: ## run apps/athlete-svc (Django HTTP server)
	@$(MAKE) -C apps/athlete-svc run

.PHONY: indexer
indexer: ## run apps/indexer (Watermill consumer + /search)
	@$(MAKE) -C apps/indexer run

.PHONY: web
web: ## run apps/web (Vite dev server, proxies /api -> indexer:8003)
	@$(MAKE) -C apps/web run

.PHONY: dev
dev: ## run all four apps together (overmind > honcho > pure-bash fallback)
	@printf "\033[36m→\033[0m if infra isn't up, ^C and run \033[36mmake up\033[0m first\n"
	@if command -v overmind >/dev/null 2>&1; then \
		exec overmind start -f Procfile; \
	elif command -v honcho >/dev/null 2>&1; then \
		exec honcho start -f Procfile; \
	else \
		printf "\033[33m!\033[0m no overmind/honcho — using pure-bash runner (prefixed logs, ^C stops all)\n"; \
		printf "  for nicer UX: \033[36mbrew install overmind\033[0m or \033[36mpipx install honcho\033[0m\n\n"; \
		exec bash scripts/dev.sh; \
	fi

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help: ## show this help
	@printf "\n\033[1mUsage:\033[0m make \033[36m<target>\033[0m\n\n"
	@grep -hE '^[a-zA-Z_/-]+:.*?## .*$$' Makefile | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' | sort
	@printf "\n\033[1mPer-app Makefiles:\033[0m\n"
	@printf "  apps/api-svc      Go (Gin + sqlc + Watermill)\n"
	@printf "  apps/athlete-svc  Python (Django + outbox)\n"
	@printf "  apps/indexer      Go (Watermill + OpenSearch)\n"
	@printf "  apps/web          TypeScript (React Router v7)\n\n"
