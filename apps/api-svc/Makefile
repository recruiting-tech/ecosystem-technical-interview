.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

DB_URL ?= postgres://emc:emc@localhost:5532/api_svc?sslmode=disable

.PHONY: gen
gen: ## regenerate sqlc + oapi-codegen output
	@go tool sqlc generate
	@cd oapi && go tool oapi-codegen -o server.gen.go --config=server.cfg.yaml openapi.yaml
	@cd oapi && go tool oapi-codegen -o types.gen.go  --config=types.cfg.yaml openapi.yaml
	@go mod tidy
	@printf "\033[32m✓\033[0m gen complete\n"

.PHONY: build
build: ## compile the server binary
	@go build -o bin/server ./cmd/server

.PHONY: test
test: ## run tests (uses testcontainers — needs Docker running)
	@go test -count=1 -race ./...

.PHONY: vet
vet: ## go vet
	@go vet ./...

.PHONY: run
run: ## run the server (assumes infra is up via root `make up`)
	@go run ./cmd/server

.PHONY: migrate/up
migrate/up: ## run goose migrations against $(DB_URL)
	@go tool goose -dir db/migrations postgres "$(DB_URL)" up

.PHONY: migrate/down
migrate/down: ## rollback last migration
	@go tool goose -dir db/migrations postgres "$(DB_URL)" down

.PHONY: migrate/create
migrate/create: ## create a new sequential migration (usage: make migrate/create name=add_column)
	@go tool goose -s -dir db/migrations create $(name) sql

.PHONY: help
help: ## show help
	@grep -hE '^[a-zA-Z_/-]+:.*?## .*$$' Makefile | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' | sort
