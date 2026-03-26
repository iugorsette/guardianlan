COMPOSE ?= docker compose

.PHONY: up down build logs test-go test-rust test-ui test

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down -v

build:
	$(COMPOSE) build

logs:
	$(COMPOSE) logs -f control-plane discovery-collector dns-collector flow-collector dashboard

test-go:
	docker run --rm -v $(PWD):/workspace -w /workspace/services/control-plane golang:1.23 go test ./...

test-rust:
	docker run --rm -v $(PWD):/workspace -w /workspace/collectors rust:1.90-bookworm cargo test --workspace

test-ui:
	cd apps/dashboard && npm run build

test: test-go test-rust test-ui
