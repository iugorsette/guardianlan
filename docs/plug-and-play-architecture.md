# Plug And Play Architecture

## Objetivo

Esta e a arquitetura final desejada para o `Guardian LAN` como produto domestico `plug and play`.

A meta e simples:

- ligar um mini PC
- colocar ele no caminho da rede
- conectar um access point atras dele
- proteger automaticamente qualquer celular, tablet, TV ou IoT que entrar nessa rede

## Topologia final

```text
internet/modem/ONU
        |
        v
mini PC Guardian LAN
  - WAN
  - LAN
        |
        v
AP ou roteador em modo bridge/AP
        |
        v
dispositivos da casa
```

## Responsabilidade de cada peca

### Mini PC Guardian LAN

Responsavel por:

- `roteamento`
- `NAT`
- `DNS`
- `DHCP`
- `bloqueio de categorias e dominios`
- `inventario de dispositivos`
- `alertas e telemetria`
- `painel local`

Stack recomendada:

- `systemd-networkd` ou `NetworkManager` para interfaces
- `nftables` para NAT, forwarding e DNS redirect
- `AdGuard Home` para DNS e DHCP
- `Guardian LAN control-plane`
- `Rust collectors`
- `PostgreSQL`
- `NATS`
- `dashboard Angular`

### Access Point

Responsavel por:

- fornecer Wi-Fi
- operar em `bridge/AP mode`
- nao distribuir DHCP
- nao tomar decisao de DNS

### Dispositivos da casa

Esperado:

- entrar na LAN e receber configuracao automaticamente
- passar pelo DNS do appliance
- obedecer politicas sem configuracao manual por cliente

## Fluxo de rede esperado

1. O cliente conecta no Wi-Fi.
2. O cliente recebe IP e DNS do `Guardian LAN`.
3. O cliente usa o `AdGuard Home` do appliance como resolvedor.
4. Regras de bloqueio valem antes do acesso sair para a internet.
5. O `Guardian LAN` correlaciona consultas, dispositivo, perfil e alerta.

## Por que isso atende o `plug and play`

Sem essa topologia, o produto vira "mais um app" que exige configuracao manual por celular.

Com essa topologia:

- a rede ja nasce protegida
- nao depende de abrir iPhone por iPhone
- novos clientes entram sob a mesma politica
- a experiencia fica muito mais proxima de produto de prateleira

## Configuracao recomendada do mini PC

Padrao sugerido:

- `WAN`: DHCP do provedor/modem
- `LAN`: `192.168.50.1/24`
- `DHCP range`: `192.168.50.50` ate `192.168.50.250`
- `DNS`: `192.168.50.1`
- painel: `http://192.168.50.1`

## Politicas de enforcement recomendadas

Base:

- redirecionar toda porta `53` da LAN para o appliance
- bloquear `853` para reduzir DoT
- usar perfis por dispositivo e por categoria

Complementares:

- lista de `DoH` conhecidos
- deteccao de `Private Relay`
- alerta de cliente com DNS inesperado

## One-time setup aceitavel

Mesmo em um produto `plug and play`, alguns passos iniciais ainda existem:

- ligar `WAN` e `LAN` nas portas certas
- colocar o roteador antigo em `AP/bridge`
- aplicar configuracao inicial do mini PC

Depois disso, a meta do produto e que o resto da operacao seja local, simples e guiada.
