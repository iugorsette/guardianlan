#!/usr/bin/env bash

set -euo pipefail

echo "Aplicando sysctl para Guardian LAN appliance mode"

sudo sysctl -w net.ipv4.ip_forward=1
sudo sysctl -w net.ipv6.conf.all.forwarding=1

echo
echo "Persistencia sugerida:"
echo "  /etc/sysctl.d/99-guardian-lan.conf"
echo "com:"
echo "  net.ipv4.ip_forward=1"
echo "  net.ipv6.conf.all.forwarding=1"
