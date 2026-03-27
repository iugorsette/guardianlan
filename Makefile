COMPOSE ?= docker compose
APPLIANCE_ENV ?= deploy/appliance/.env.gateway

.PHONY: up down build logs api api-build api-stop api-network ui dev dev-network dev-stop observer-up observer-down appliance-preflight appliance-render-nft appliance-render-networkd appliance-sysctl appliance-nft appliance-up appliance-down appliance-host-dev-up appliance-host-dev-down test-go test-rust test-ui test

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down -v

build:
	$(COMPOSE) build

logs:
	$(COMPOSE) logs -f control-plane discovery-collector dns-collector flow-collector dashboard

api:
	$(COMPOSE) up -d nats postgres control-plane

api-build:
	$(COMPOSE) up -d --build nats postgres control-plane

api-network:
	$(COMPOSE) --profile tooling up -d nats postgres control-plane adguardhome dns-collector discovery-collector

api-stop:
	$(COMPOSE) stop nats postgres control-plane

ui:
	cd apps/dashboard && npm start

dev: api
	cd apps/dashboard && npm start

dev-network: api-network
	cd apps/dashboard && npm start

dev-stop: api-stop

observer-up:
	bash scripts/observer/up.sh

observer-down:
	bash scripts/observer/down.sh

appliance-preflight:
	bash scripts/appliance/preflight.sh $(APPLIANCE_ENV)

appliance-render-nft:
	bash scripts/appliance/render-nftables.sh $(APPLIANCE_ENV)

appliance-render-networkd:
	bash scripts/appliance/render-networkd.sh $(APPLIANCE_ENV)

appliance-sysctl:
	bash scripts/appliance/apply-sysctl.sh

appliance-nft:
	bash scripts/appliance/apply-nftables.sh $(APPLIANCE_ENV)

appliance-up:
	$(COMPOSE) --env-file $(APPLIANCE_ENV) --profile tooling up -d nats postgres control-plane adguardhome dns-collector discovery-collector dashboard

appliance-down:
	$(COMPOSE) --env-file $(APPLIANCE_ENV) down

appliance-host-dev-up:
	$(COMPOSE) --env-file deploy/appliance/.env.host-dev --profile tooling up -d nats postgres control-plane adguardhome dns-collector discovery-collector dashboard

appliance-host-dev-down:
	$(COMPOSE) --env-file deploy/appliance/.env.host-dev down

test-go:
	docker run --rm -v $(PWD):/workspace -w /workspace/services/control-plane golang:1.23 go test ./...

test-rust:
	docker run --rm -v $(PWD):/workspace -w /workspace/collectors rust:1.90-bookworm cargo test --workspace

test-ui:
	cd apps/dashboard && npm run build

test: test-go test-rust test-ui
