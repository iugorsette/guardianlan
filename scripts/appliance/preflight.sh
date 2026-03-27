#!/usr/bin/env bash

set -euo pipefail

ENV_FILE="${1:-deploy/appliance/.env.gateway}"

if [[ -f "$ENV_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$ENV_FILE"
fi

WAN_IF="${APPLIANCE_WAN_IF:-}"
LAN_IF="${APPLIANCE_LAN_IF:-}"
LAN_CIDR="${APPLIANCE_LAN_CIDR:-}"
LAN_GW="${APPLIANCE_LAN_GW:-}"
DNS_IP="${APPLIANCE_DNS_IP:-}"
ADGUARD_DNS_BIND_IP="${ADGUARD_DNS_BIND_IP:-0.0.0.0}"
ADGUARD_WEB_BIND_IP="${ADGUARD_WEB_BIND_IP:-127.0.0.1}"
APPLIANCE_MODE="${APPLIANCE_MODE:-gateway}"

failures=0
warnings=0

pass() {
  printf '[ok] %s\n' "$1"
}

warn() {
  printf '[warn] %s\n' "$1"
  warnings=$((warnings + 1))
}

fail() {
  printf '[fail] %s\n' "$1"
  failures=$((failures + 1))
}

require_cmd() {
  if command -v "$1" >/dev/null 2>&1; then
    pass "comando disponivel: $1"
  else
    fail "comando ausente: $1"
  fi
}

port_in_use() {
  local bind_ip="$1"
  local port="$2"

  if [[ "$bind_ip" == "0.0.0.0" ]]; then
    (ss -ltnu 2>/dev/null || true) | grep -qE "[:.]${port}\b"
    return
  fi

  (ss -ltnu 2>/dev/null || true) | grep -F "${bind_ip}:${port}" >/dev/null 2>&1
}

check_interface() {
  local ifname="$1"
  if [[ -z "$ifname" ]]; then
    fail "interface nao definida no env"
    return
  fi

  if ip link show "$ifname" >/dev/null 2>&1; then
    pass "interface encontrada: $ifname"
  else
    fail "interface nao encontrada: $ifname"
    return
  fi

  local state
  state="$(ip -brief link show "$ifname" | awk '{print $2}')"
  if [[ "$state" == "UP" || "$state" == "UNKNOWN" ]]; then
    pass "interface ativa: $ifname ($state)"
  else
    warn "interface encontrada mas nao esta UP: $ifname ($state)"
  fi
}

echo "Guardian LAN appliance preflight"
echo "env file: $ENV_FILE"
echo

require_cmd docker
require_cmd ip
require_cmd nft
require_cmd sysctl

if docker compose version >/dev/null 2>&1 || docker-compose version >/dev/null 2>&1; then
  pass "docker compose disponivel"
else
  fail "docker compose indisponivel"
fi

if [[ "$(uname -s)" == "Linux" ]]; then
  pass "host Linux detectado"
else
  fail "appliance mode requer Linux"
fi

if [[ "$APPLIANCE_MODE" == "gateway" ]]; then
  check_interface "$WAN_IF"
  check_interface "$LAN_IF"

  if [[ -n "$WAN_IF" && -n "$LAN_IF" && "$WAN_IF" == "$LAN_IF" ]]; then
    fail "WAN e LAN nao podem usar a mesma interface"
  fi

  if (ip route show default 2>/dev/null || true) | grep -q "dev ${WAN_IF:-__missing__}"; then
    pass "rota default usa a interface WAN esperada"
  else
    warn "rota default nao esta passando pela WAN configurada"
  fi

  if [[ -n "$LAN_GW" ]] && ip -brief addr show "$LAN_IF" 2>/dev/null | grep -q "$LAN_GW"; then
    pass "gateway LAN ja aparece configurado em $LAN_IF"
  else
    warn "IP de gateway LAN ainda nao aparece em $LAN_IF: esperado $LAN_GW"
  fi

  if [[ -n "$DNS_IP" && -n "$LAN_GW" && "$DNS_IP" != "$LAN_GW" ]]; then
    warn "APPLIANCE_DNS_IP difere de APPLIANCE_LAN_GW; confirme se isso e intencional"
  fi

  if [[ -n "$LAN_CIDR" ]]; then
    pass "LAN CIDR definido: $LAN_CIDR"
  else
    fail "APPLIANCE_LAN_CIDR nao definido"
  fi
else
  warn "preflight em modo host-dev: validacoes de WAN/LAN foram puladas"
fi

if port_in_use "$ADGUARD_DNS_BIND_IP" 53; then
  fail "porta DNS ja ocupada no bind configurado (${ADGUARD_DNS_BIND_IP}:53)"
  warn "em host de desenvolvimento, prefira um bind dedicado como o IP LAN da maquina ou 127.0.0.1"
else
  pass "porta DNS livre para o bind configurado (${ADGUARD_DNS_BIND_IP}:53)"
fi

if port_in_use "$ADGUARD_WEB_BIND_IP" 3000; then
  fail "porta web do AdGuard ja ocupada no bind configurado (${ADGUARD_WEB_BIND_IP}:3000)"
else
  pass "porta web do AdGuard livre no bind configurado (${ADGUARD_WEB_BIND_IP}:3000)"
fi

if [[ "$(sysctl -n net.ipv4.ip_forward 2>/dev/null || echo 0)" == "1" ]]; then
  pass "ip_forward ja esta habilitado"
else
  warn "ip_forward ainda nao esta habilitado"
fi

if (ss -ltnu 2>/dev/null || true) | grep -qE '(:53|:80|:443)\b'; then
  warn "existem servicos locais usando portas sensiveis; valide conflitos antes do deploy final"
else
  pass "nenhum conflito obvio em 53/80/443 detectado"
fi

echo
if (( failures > 0 )); then
  printf 'Preflight terminou com %d falha(s) e %d aviso(s).\n' "$failures" "$warnings"
  exit 1
fi

printf 'Preflight terminou com %d aviso(s) e nenhuma falha bloqueante.\n' "$warnings"
