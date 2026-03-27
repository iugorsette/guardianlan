#!/usr/bin/env bash

set -euo pipefail

ENV_FILE="${1:-deploy/appliance/.env.gateway}"
OUTPUT_FILE="${2:-deploy/appliance/nftables/guardian-lan.nft}"

"$(dirname "$0")/render-nftables.sh" "$ENV_FILE" "deploy/appliance/nftables/guardian-lan.nft.template" "$OUTPUT_FILE"

echo "Aplicando regras nftables: $OUTPUT_FILE"
sudo nft -f "$OUTPUT_FILE"

echo "Regras aplicadas."
