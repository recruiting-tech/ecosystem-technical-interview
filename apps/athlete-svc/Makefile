.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

VENV := .venv
PY   := $(VENV)/bin/python
PIP  := $(VENV)/bin/pip

.PHONY: install
install: ## create venv and install deps (Poetry-style pyproject)
	@if [ ! -d $(VENV) ]; then python3 -m venv $(VENV); fi
	@$(PIP) install -q --upgrade pip
	@$(PIP) install -q -e ".[dev]"
	@printf "\033[32m✓\033[0m installed into $(VENV)\n"

.PHONY: migrate
migrate: ## apply Django migrations against $(DB_HOST):$(DB_PORT)
	@$(PY) manage.py migrate

.PHONY: run
run: ## run the HTTP server (assumes infra is up via root `make up`)
	@$(PY) manage.py runserver 0.0.0.0:8002

.PHONY: dispatcher
dispatcher: ## drain the event outbox to Kafka once
	@$(PY) manage.py poll_event_outbox

.PHONY: test
test: ## run pytest (uses the live compose stack — `make up` first)
	@$(VENV)/bin/pytest

.PHONY: lint
lint: ## ruff check
	@$(VENV)/bin/ruff check .

.PHONY: gen
gen: ## (no codegen here; here for parity)
	@:

.PHONY: help
help: ## show help
	@grep -hE '^[a-zA-Z_/-]+:.*?## .*$$' Makefile | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' | sort
