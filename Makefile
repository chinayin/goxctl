.PHONY: help build run clean test coverage lint lint-fix fmt tidy check install-tools ensure-lint

APP := goxctl

# golangci-lint 版本锁定（随项目走，升级改 .golangci-lint-version）
GOLANGCI_VERSION := $(shell cat .golangci-lint-version)
GOLANGCI := ./bin/golangci-lint

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-13s - %s\n", $$1, $$2}'

build: clean ## Compile
	@go build -o bin/$(APP) ./cmd/$(APP)

run: build ## Run
	@./bin/$(APP) --help

clean: ## Clean build output
	@rm -rf bin/ coverage.txt coverage.html

ensure-lint:
	@if [ ! -x "$(GOLANGCI)" ] || ! "$(GOLANGCI)" version 2>/dev/null | grep -q "$(GOLANGCI_VERSION:v%=%)"; then \
		curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b ./bin "$(GOLANGCI_VERSION)"; \
	fi

install-tools: ensure-lint ## Install pinned golangci-lint to ./bin
	@echo "golangci-lint $(GOLANGCI_VERSION) ready at $(GOLANGCI)"

test: ## Run tests
	@go test -v -race -count=1 -timeout 120s ./...

coverage: ## Generate coverage report
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html

lint: ensure-lint ## Run linter (pinned version)
	@$(GOLANGCI) run ./...

lint-fix: ensure-lint ## Run linter with auto-fix
	@$(GOLANGCI) run --fix ./...

fmt: lint-fix ## Format code (lint + fix)
	@echo "Code formatted!"

check: fmt lint test ## Full local check (run before commit)
	@echo "All checks passed!"
