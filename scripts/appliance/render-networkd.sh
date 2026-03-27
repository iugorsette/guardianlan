#!/usr/bin/env bash

set -euo pipefail

ENV_FILE="${1:-deploy/appliance/.env.gateway}"
OUT_DIR="${2:-deploy/appliance/systemd-networkd/rendered}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "env file nao encontrado: $ENV_FILE" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"

: "${APPLIANCE_WAN_IF:?APPLIANCE_WAN_IF obrigatorio}"
: "${APPLIANCE_LAN_IF:?APPLIANCE_LAN_IF obrigatorio}"
: "${APPLIANCE_LAN_ADDRESS:?APPLIANCE_LAN_ADDRESS obrigatorio}"

mkdir -p "$OUT_DIR"

sed \
  -e "s|\$WAN_IF|${APPLIANCE_WAN_IF}|g" \
  deploy/appliance/systemd-networkd/10-wan.network.template \
  > "${OUT_DIR}/10-wan.network"

sed \
  -e "s|\$LAN_IF|${APPLIANCE_LAN_IF}|g" \
  -e "s|\$LAN_ADDRESS|${APPLIANCE_LAN_ADDRESS}|g" \
  deploy/appliance/systemd-networkd/20-lan.network.template \
  > "${OUT_DIR}/20-lan.network"

echo "arquivos gerados em $OUT_DIR"
