# Guardian LAN

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

## O que esta base já entrega

- `Collectors` em Rust para descoberta, DNS e fluxo publicando eventos no `NATS`
- `Control plane` em Go para inventário, correlação, alertas e API local
- `Dashboard` em Angular para operar dispositivos, alertas e telemetria local
- `PostgreSQL` para estado durável
- `Docker Compose` para subir tudo de forma previsível
- documentação inicial da arquitetura, segurança, contratos e operação

## Limites reais da v1

- Sem trocar o roteador, a plataforma não enxerga todo o tráfego da rede.
- `HTTPS` impede inspeção de conteúdo; esta base trabalha com metadados, DNS, fluxo e inventário.
- Os collectors da v1 estão preparados para ingestão por arquivo/export e publicação de eventos. Eles são o ponto de integração para fontes reais da rede, como APIs do roteador, exports do AdGuard Home e ntopng, ou futuras capturas com mais privilégios.
- A v1 prioriza descoberta, inventario, classificacao, alertas e sinais de risco; nao existe promessa de controle total sem mudar a topologia da rede.

## Estrutura

- `services/control-plane`: backend em Go
- `collectors`: workspace Rust com collectors e biblioteca comum
- `infra/postgres/init`: schema e seeds
- `fixtures`: dados de exemplo para subir a stack e validar o fluxo
- `docs`: documentação técnica e ADRs

## Como subir

```bash
cp .env.example .env
docker compose up --build
```

API local:

- `GET http://localhost:8080/healthz`
- `GET http://localhost:8080/devices`
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

## Próximos passos sugeridos

- trocar fixtures por conectores reais de roteador, AdGuard Home e export de fluxo
- adicionar autenticação local e perfis mais ricos
- incluir agentes de endpoint para notebooks e PCs
- evoluir para topologia gateway quando houver necessidade de controle total
