.DEFAULT_GOAL := help

SHELL := /usr/bin/env bash

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

.PHONY: test
test: ## run every app's test suite
	@$(MAKE) -C apps/api-svc test
	@$(MAKE) -C apps/athlete-svc test
	@$(MAKE) -C apps/indexer test
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
