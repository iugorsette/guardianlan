#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

COMPOSE_ARGS=()
if [[ -f ".env" ]]; then
  # shellcheck disable=SC1090
  source ".env"
  if [[ "${ADGUARD_ENABLED:-false}" == "true" || "${DNS_SOURCE:-disabled}" == "adguard_file" ]]; then
    COMPOSE_ARGS+=(--profile tooling)
  fi
fi

echo "Parando Guardian LAN em Observer Mode..."
docker compose "${COMPOSE_ARGS[@]}" stop nats postgres control-plane dashboard discovery-collector dns-collector flow-collector adguardhome
