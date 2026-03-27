# ADR 0006: Appliance Mode Como Direcao Principal da v2

## Status

Accepted

## Contexto

O modo atual de desenvolvimento e suficiente para descoberta, inventario e alguns testes locais, mas nao atende a expectativa de `protecao pela rede inteira` sem configuracao manual em cada dispositivo.

Para familias com baixa familiaridade tecnica, depender de configurar DNS em iPhone, Android, tablet ou TV e um atrito grande demais.

## Decisao

A direcao principal da `v2` passa a ser `appliance mode` em `mini PC` Linux dedicado, com foco em `gateway mode` quando o objetivo for enforcement forte.

Isso implica:

- mini PC 24x7
- duas interfaces de rede para o modo forte
- `Guardian LAN + AdGuard Home` como base local
- uso de `nftables` e `ip_forward` para NAT, forwarding e redirecionamento DNS

## Consequencias

Positivas:

- melhor experiencia para usuarios leigos
- bloqueio e descoberta mais consistentes
- menos dependencia de configuracao por cliente
- melhor base para produto comercial/domestico

Trade-offs:

- exige topologia fisica adequada
- aumenta a responsabilidade operacional do appliance
- onboarding precisa ser muito bem guiado
