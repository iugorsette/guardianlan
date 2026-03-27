# Architecture

## Visao geral

A stack e organizada em dois planos:

- `Rust collectors`: proximos da rede e do dado bruto, publicando eventos normalizados
- `Go control plane`: consolida estado, aplica regras, correlaciona sinais e atende a API local
- `Angular dashboard`: painel local para operacao, leitura de telemetria e acoes de perfil/alerta

Acima da arquitetura tecnica, o produto segue uma intencao clara:

- ser local e simples de operar
- ajudar familias, nao apenas operadores tecnicos
- transformar sinais tecnicos em explicacoes e acoes compreensiveis
- observar a casa melhor do que controlar a casa inteira sem lastro tecnico

## Componentes

- `discovery-collector`
  - publica `network.device.discovered` e `network.device.updated`
  - hoje ja descobre vizinhos reais da LAN com `ip neigh`, `fping`, OUI e fingerprint leve
  - futuro: leases do roteador, mDNS, SSDP e conectores mais ricos
- `dns-collector`
  - publica `network.dns.query_observed`
  - hoje usa fixture ou querylog JSON/JSONL do `AdGuard Home`
  - envia `client_ip` e `client_name` quando disponiveis para correlacao no control plane
  - futuro: indicadores mais fortes de bypass e DoH
- `flow-collector`
  - publica `network.flow.observed`
  - hoje usa fixture/export de fluxo
  - futuro: ntopng, NetFlow/sFlow/IPFIX ou captura especializada
- `control-plane`
  - consome eventos do `NATS`
  - atualiza inventario em `PostgreSQL`
  - gera observacoes e alertas
  - resolve eventos DNS para dispositivos reais por IP/nome
  - combina perfil base com override DNS por dispositivo
  - expõe API HTTP local
- `dashboard`
  - consome a API local via `/api`
  - mostra inventario, alertas, DNS e fluxo
  - permite ajustar perfil, nome amigavel, watchlists de interesse e reconhecer alerta

## Fluxo principal

1. Um collector le a fonte de entrada e normaliza o evento.
2. O collector publica o JSON no subject apropriado do `NATS`.
3. O `control-plane` consome o evento.
4. O evento e persistido e correlacionado.
5. Se houver risco, o sistema cria um alerta local e publica `network.alert.raised`.
6. A API local disponibiliza dispositivos, atividades e alertas para UI e automacoes.

## Fronteiras

- `Rust` fica responsavel por ingestao e normalizacao de eventos.
- `Go` concentra regras de negocio, persistencia e API.
- `Angular` concentra a experiencia de operacao local.
- Ferramentas como `AdGuard Home` e `ntopng` continuam sendo fontes externas especializadas.

## Observer mode

A arquitetura atual deve ser lida primeiro como `Observer Mode`:

- `discovery` e a fonte mais ampla e confiavel sem mexer na rede
- `dns`, `fluxo` e `endpoint` entram como fontes opcionais de telemetria
- alertas dependem da existencia dessas fontes

Detalhes em [docs/observer-mode.md](/home/sette/github/parental-local/docs/observer-mode.md).

## Leitmotiv de produto

A arquitetura nao existe para "interceptar tudo" por padrao. Ela existe para equilibrar:

- protecao infantil
- seguranca da rede local
- operacao local-first
- explicacao simples para usuarios leigos

Por isso a v1 privilegia metadados, descoberta, classificacao, DNS, fluxo e correlacao antes de qualquer proposta de captura profunda.

## Decisoes importantes

- `NATS` foi escolhido por ser leve e adequado para processamento local.
- O sistema guarda resumos e metadados por padrao.
- Captura de payload bruto fica fora do caminho padrao da v1.
