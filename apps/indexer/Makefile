.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

.PHONY: build
build: ## compile binary
	@go build -o bin/indexer ./

.PHONY: test
test: ## run tests
	@go test -count=1 -race ./...

.PHONY: vet
vet: ## go vet
	@go vet ./...

.PHONY: gen
gen: ## (no codegen for indexer; here for parity with root make gen)
	@:

.PHONY: run
run: ## run consumer + search-server (assumes infra is up)
	@go run ./ run

.PHONY: consume
consume: ## run consumer only
	@go run ./ consume

.PHONY: search-server
search-server: ## run search HTTP server only
	@go run ./ search-server

.PHONY: help
help: ## show help
	@grep -hE '^[a-zA-Z_/-]+:.*?## .*$$' Makefile | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' | sort
