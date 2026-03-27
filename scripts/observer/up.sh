#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -f ".env" ]]; then
  cp .env.example .env
  echo "Arquivo .env criado a partir de .env.example"
fi

# shellcheck disable=SC1090
source ".env"

SERVICES=(nats postgres control-plane dashboard discovery-collector dns-collector flow-collector)
COMPOSE_ARGS=()

if [[ "${ADGUARD_ENABLED:-false}" == "true" || "${DNS_SOURCE:-disabled}" == "adguard_file" ]]; then
  COMPOSE_ARGS+=(--profile tooling)
  SERVICES+=(adguardhome)
fi

echo "Subindo Guardian LAN em Observer Mode..."
docker compose "${COMPOSE_ARGS[@]}" up -d --build "${SERVICES[@]}"

echo
echo "Observer Mode ativo."
echo "Dashboard: http://localhost:4201"
echo "API:       http://localhost:8080/healthz"
echo "NATS:      http://localhost:8222"
if [[ " ${SERVICES[*]} " == *" adguardhome "* ]]; then
  echo "AdGuard:   http://127.0.0.1:3000"
fi
echo
echo "Observacoes:"
echo "- Este modo prioriza descoberta, inventario e alertas."
echo "- Alertas de DNS dependem de uma fonte real configurada no .env."
if [[ " ${SERVICES[*]} " == *" adguardhome "* ]]; then
  echo "- Telemetria DNS com AdGuard foi ativada porque .env pediu ADGUARD_ENABLED=true ou DNS_SOURCE=adguard_file."
fi
echo "- 'docker compose build' sozinho nao sobe a aplicacao; use este script ou 'make observer-up'."
