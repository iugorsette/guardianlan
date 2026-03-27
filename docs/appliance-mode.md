# Appliance Mode

## Objetivo

O `appliance mode` e o caminho principal da `v2` para entregar o que familias realmente esperam:

- protecao aplicada na `rede inteira`
- onboarding simples
- menos configuracao manual por dispositivo
- comportamento mais proximo de `plug and play`

Esse modo roda em um `mini PC` Linux dedicado e coloca o `Guardian LAN` no centro da topologia da casa.

## Quando usar

Use este modo quando o objetivo for:

- bloquear dominios e categorias para qualquer dispositivo que entrar na rede
- evitar depender de configuracao manual em iPhone, Android, tablet ou smart TV
- reduzir bypass por DNS manual
- ter uma base unica de seguranca domestica

## Topologia recomendada

O modo forte depende de `duas interfaces de rede` no mini PC:

- `WAN`: ligada ao modem, ONU ou ao equipamento que entrega internet
- `LAN`: ligada a um switch, access point ou roteador em modo `AP/bridge`

Fluxo fisico:

1. internet/modem
2. mini PC `Guardian LAN`
3. access point ou switch da casa
4. dispositivos da familia

## O que muda em relacao ao modo atual

No modo atual de desenvolvimento, o projeto roda como um servico local e observa partes da rede.

No `appliance mode`, o mini PC passa a ser:

- resolvedor DNS da rede
- ponto de DHCP da rede local, quando desejado
- gateway/NAT da rede local
- base de descoberta, inventario e alertas

## Por que esse e o modo certo para familias leigas

Esse modo reduz o numero de passos por dispositivo:

- o pai ou a mae nao precisa entrar em cada celular para configurar DNS
- quem entra na rede ja nasce sob a politica da casa
- tablets, TVs e IoT ficam cobertos pelo mesmo caminho
- o painel consegue explicar risco e acao em um unico lugar

## Requisitos minimos

- mini PC Linux 24x7
- `2 NICs` fisicas ou uma NIC + adaptador USB Ethernet confiavel
- Docker e Docker Compose
- access point ou roteador que aceite operar em `AP/bridge`

## Escopo da base implementada no repositório

Esta base prepara o repositório para o modo appliance com:

- `env` dedicado para gateway/appliance
- templates de rede do mini PC
- `preflight` de maquina e interfaces
- template de `nftables` para NAT, forwarding e redirecionamento DNS
- alvos de `Makefile` para validar e operar esse modo

## Limites honestos

Mesmo em `appliance mode`, ainda existem limites:

- `DoH` e `Private Relay` exigem mitigacoes adicionais
- algumas TVs e apps usam DNS proprio
- bloquear `853` ajuda com DoT, mas DoH exige listas e correlacao extras
- o produto pode ser muito simples na UX, mas a topologia fisica ainda precisa existir

## Padrao recomendado de rollout

1. preparar mini PC
2. validar interfaces com `preflight`
3. configurar LAN do mini PC
4. aplicar `ip_forward` e regras de `nftables`
5. subir `Guardian LAN + AdGuard Home`
6. colocar o equipamento antigo em `AP/bridge`
7. conectar clientes na nova LAN

## Resultado esperado

Quando o modo estiver ativo:

- qualquer dispositivo que entrar na rede recebe DNS e rota do appliance
- regras por perfil/dispositivo passam a valer sem configuracao manual no cliente
- o painel passa a refletir melhor o que realmente acontece na casa
