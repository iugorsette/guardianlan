#!/usr/bin/env bash

set -euo pipefail

ENV_FILE="${1:-deploy/appliance/.env.gateway}"
TEMPLATE_FILE="${2:-deploy/appliance/nftables/guardian-lan.nft.template}"
OUTPUT_FILE="${3:-deploy/appliance/nftables/guardian-lan.nft}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "env file nao encontrado: $ENV_FILE" >&2
  exit 1
fi

if [[ ! -f "$TEMPLATE_FILE" ]]; then
  echo "template nao encontrado: $TEMPLATE_FILE" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"

: "${APPLIANCE_WAN_IF:?APPLIANCE_WAN_IF obrigatorio}"
: "${APPLIANCE_LAN_IF:?APPLIANCE_LAN_IF obrigatorio}"
: "${APPLIANCE_LAN_CIDR:?APPLIANCE_LAN_CIDR obrigatorio}"
: "${APPLIANCE_DNS_IP:?APPLIANCE_DNS_IP obrigatorio}"

SSH_RULE="# ssh remoto desabilitado"
if [[ "${APPLIANCE_ALLOW_SSH:-false}" == "true" ]]; then
  SSH_RULE="iifname \"${APPLIANCE_LAN_IF}\" tcp dport 22 accept"
fi

DNS_REDIRECT_UDP="# redirecionamento DNS UDP desabilitado"
DNS_REDIRECT_TCP="# redirecionamento DNS TCP desabilitado"
if [[ "${APPLIANCE_ENABLE_DNS_REDIRECT:-true}" == "true" ]]; then
  DNS_REDIRECT_UDP="iifname \"${APPLIANCE_LAN_IF}\" udp dport 53 redirect to 53"
  DNS_REDIRECT_TCP="iifname \"${APPLIANCE_LAN_IF}\" tcp dport 53 redirect to 53"
fi

DOT_BLOCK="# bloqueio de DoT desabilitado"
if [[ "${APPLIANCE_BLOCK_DOT:-true}" == "true" ]]; then
  DOT_BLOCK="iifname \"${APPLIANCE_LAN_IF}\" tcp dport 853 reject"
fi

sed \
  -e "s|\$WAN_IF|${APPLIANCE_WAN_IF}|g" \
  -e "s|\$LAN_IF|${APPLIANCE_LAN_IF}|g" \
  -e "s|\$LAN_CIDR|${APPLIANCE_LAN_CIDR}|g" \
  -e "s|\$SSH_RULE|${SSH_RULE}|g" \
  -e "s|\$DNS_REDIRECT_UDP|${DNS_REDIRECT_UDP}|g" \
  -e "s|\$DNS_REDIRECT_TCP|${DNS_REDIRECT_TCP}|g" \
  -e "s|\$DOT_BLOCK|${DOT_BLOCK}|g" \
  "$TEMPLATE_FILE" > "$OUTPUT_FILE"

echo "arquivo gerado em $OUTPUT_FILE"
