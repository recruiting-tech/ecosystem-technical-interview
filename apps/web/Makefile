.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

.PHONY: install
install: node_modules ## install npm deps (idempotent)

node_modules: package.json
	@npm install
	@touch node_modules

.PHONY: dev
dev: ## run vite dev server (proxies /api -> indexer search-server)
	@npm run dev

.PHONY: run
run: install ## install if needed, then run vite dev server
	@npm run dev

.PHONY: build
build: ## production build
	@npm run build

.PHONY: test
test: ## run vitest once
	@npm test

.PHONY: gen
gen: ## (no codegen here; here for parity)
	@:

.PHONY: help
help: ## show help
	@grep -hE '^[a-zA-Z_/-]+:.*?## .*$$' Makefile | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' | sort
