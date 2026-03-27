# Guardian LAN

## Preview do app

<p align="center">
  <img src="https://github.com/user-attachments/assets/cd210f08-b978-4519-bed7-c8d0fdbd21ff" alt="Dashboard principal do Guardian LAN" width="31%" />
  <img src="https://github.com/user-attachments/assets/9c9dff50-fc4a-405a-bdca-f05ff873ad6f" alt="Inventario e alertas do Guardian LAN" width="31%" />
  <img src="https://github.com/user-attachments/assets/f20f272a-0f48-49ea-90cc-8b2698c9e0d5" alt="Detalhes e observacoes do Guardian LAN" width="31%" />
</p>

Plataforma `local-first` para gerenciamento de rede domestica, controle parental e observabilidade de seguranca rodando em uma maquina da casa.

## Briefing do produto

O `Guardian LAN` foi pensado para familias e cuidadores que querem proteger criancas e adolescentes dentro da propria rede da casa, sem depender de um servico online como ponto central.

Mais do que um painel tecnico, a ideia do produto e ser uma camada de orientacao e protecao domestica para:

- descobrir dispositivos reais da rede
- identificar cameras, baba eletronica, tablet, celular, TV e IoT
- facilitar controle parental com perfis e politicas simples
- ajudar pais com pouca familiaridade tecnica a entender risco e agir

Resumo da proposta:

- `local-first`
- focado em `seguranca domestica + controle parental`
- pensado para pais "analfabyte" e operacao simples
- com baixo custo e sem exigir cloud como requisito central

O briefing detalhado esta em [docs/product-brief.md](/home/sette/github/parental-local/docs/product-brief.md).
O modo principal do produto hoje esta em [docs/observer-mode.md](/home/sette/github/parental-local/docs/observer-mode.md).

## O que esta base já entrega

- `Collectors` em Rust para descoberta, DNS e fluxo publicando eventos no `NATS`
- `Control plane` em Go para inventário, correlação, alertas e API local
- `Dashboard` em Angular para operar dispositivos, alertas e telemetria local
- watchlists de dominios e categorias por dispositivo para telemetria e alertas
- `PostgreSQL` para estado durável
- `Docker Compose` para subir tudo de forma previsível
- documentação inicial da arquitetura, segurança, contratos e operação

## Limites reais da v1

- Sem trocar o roteador, a plataforma não enxerga todo o tráfego da rede.
- `HTTPS` impede inspeção de conteúdo; esta base trabalha com metadados, DNS, fluxo e inventário.
- Os collectors da v1 estão preparados para ingestão por arquivo/export e publicação de eventos. Eles são o ponto de integração para fontes reais da rede, como APIs do roteador, exports do AdGuard Home e ntopng, ou futuras capturas com mais privilégios.
- A v1 prioriza descoberta, inventario, classificacao, alertas e sinais de risco.
- Em `Observer Mode`, o produto `ve, explica, alerta e ajuda pais`; ele nao promete controle automatico da rede inteira.

## Direcao do produto hoje

O produto fica mais honesto e mais util se for apresentado primeiro como `Observer`:

- descobre e classifica a rede da casa
- explica risco em linguagem simples
- alerta pais e cuidadores
- usa fontes opcionais de DNS, fluxo e endpoint quando existirem

Quando no futuro houver necessidade de enforcement de rede inteira, o repositório tambem guarda pesquisas de `appliance mode` e `gateway mode`, mas isso fica explicitamente fora da promessa principal de hoje.

## Estrutura

- `services/control-plane`: backend em Go
- `collectors`: workspace Rust com collectors e biblioteca comum
- `infra/postgres/init`: schema e seeds
- `fixtures`: dados de exemplo para subir a stack e validar o fluxo
- `docs`: documentação técnica e ADRs

## Como subir

Jeito recomendado para o produto hoje, em `Observer Mode`:

```bash
./scripts/observer/up.sh
```

ou:

```bash
make observer-up
```

Isso:

- cria `.env` a partir de `.env.example` se faltar
- sobe a stack principal do modo `Observer`
- deixa o painel em `http://localhost:4201`

Observacao importante:

- `docker compose build` apenas recompila imagens
- para rodar a aplicacao, use `./scripts/observer/up.sh`, `make observer-up` ou `docker compose up`

Fluxo manual, se preferir:

```bash
cp .env.example .env
docker compose up --build
```

API local:

- `GET http://localhost:8080/healthz`
- `GET http://localhost:8080/profiles`
- `GET http://localhost:8080/devices`
- `POST http://localhost:8080/devices/:id/name`
- `POST http://localhost:8080/integrations/adguard/sync`
- `GET http://localhost:8080/activity/dns`
- `GET http://localhost:8080/activity/flows`
- `GET http://localhost:8080/alerts`

Dashboard:

- `http://localhost:4201`

Banco local para inspecao manual:

- `postgres://postgres:guardian_lan_local_2026@localhost:5433/guardian_lan`

## Como testar

```bash
make test
```

Os testes usam containers oficiais de `golang` e `rust`, porque o ambiente de execução pode não ter essas toolchains instaladas localmente.

## Frontend Angular

Para desenvolver o painel fora do Docker:

```bash
cd apps/dashboard
npm install
npm start
```

O `proxy.conf.json` envia `/api/*` para `http://localhost:8080`.

## Observer mode

O modo principal do produto hoje e `Observer Mode`.

Ele funciona melhor para:

- inventario da rede da casa
- descoberta de camera, IoT e dispositivos novos
- explicacao de risco
- alertas por DNS, fluxo e endpoint quando houver fonte real do evento

Detalhes em [docs/observer-mode.md](/home/sette/github/parental-local/docs/observer-mode.md).

Para parar a stack do Observer:

```bash
make observer-down
```

## Descoberta real da LAN

O `discovery-collector` ja esta configurado para operar em modo `live` por padrao. Nesta versao ele:

- descobre vizinhos reais da LAN via `ip neigh` e `fping`
- tenta identificar fabricante pelo prefixo do MAC usando a base `ieee-data`
- faz fingerprint leve por portas comuns para diferenciar `camera`, `router`, `printer`, `tv`, `iot` e `computer`

Variaveis uteis no `.env`:

- `DISCOVERY_FINGERPRINT_ENABLED=true`
- `DISCOVERY_FINGERPRINT_TIMEOUT_MS=180`
- `DISCOVERY_VENDOR_DB=/usr/share/ieee-data/oui.txt`
- `DASHBOARD_PORT=4201`

## Telemetria DNS opcional

O `dns-collector` agora entende:

- `DNS_SOURCE=fixture` para testes com arquivo
- `DNS_SOURCE=adguard_file` para ler um querylog JSON/JSONL do `AdGuard Home`

Variaveis uteis:

- `ADGUARD_ENABLED=true`
- `ADGUARD_URL=http://adguardhome:3000/control`
- `ADGUARD_USERNAME=admin`
- `ADGUARD_PASSWORD=senha-definida-no-adguard`
- `DNS_ADGUARD_QUERYLOG=/adguard-work/querylog.json`
- `DNS_RESOLVER_NAME=adguardhome`

Quando eventos DNS chegam, o `control-plane`:

- tenta vincular o evento ao dispositivo real por `client_ip` e `client_name`
- aplica a politica do perfil base (`adult`, `child`, `iot`, `guest`)
- aplica override por dispositivo
- gera alertas para:
  - `dns_bypass`
  - dominio fora da lista esperada
  - dominio em watchlist
  - categoria sensivel, como `adult`

No dashboard, cada dispositivo agora pode ter:

- watchlist de dominios
- lista de dominios esperados
- categorias sensiveis
- indicacao de Safe Search esperado

Exemplo de uso:

- marque um tablet como perfil `Crianca`
- adicione `xvideos.com` na watchlist
- opcionalmente defina dominios esperados de estudo
- quando o dominio aparecer na telemetria DNS, o painel vai alertar

Importante:

- sem uma fonte de DNS real, nao ha alerta de dominio
- isso funciona para a propria maquina, para dispositivos com agente ou para clientes que usem o DNS observado pelo GuardianLAN
- isso nao significa bloqueio automatico da casa inteira

## Próximos passos sugeridos

- trocar fixtures por conectores reais de roteador, AdGuard Home e export de fluxo
- adicionar autenticação local e perfis mais ricos
- incluir agentes de endpoint para notebooks e PCs
- evoluir a cobertura observacional antes de qualquer promessa de enforcement
