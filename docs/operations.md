# Operations

## Subir a stack

```bash
cp .env.example .env
docker compose up --build
```

## Parar e limpar

```bash
docker compose down -v
```

## Validacoes rapidas

- `curl http://localhost:8080/healthz`
- `curl http://localhost:8080/profiles`
- `curl http://localhost:8080/devices`
- `curl http://localhost:8080/activity/dns`
- `curl http://localhost:8080/alerts`
- `psql postgresql://postgres:guardian_lan_local_2026@localhost:5433/guardian_lan`
- abrir `http://localhost:4201`

## Fontes de dados

- `fixtures/discovery/devices.json`
- `fixtures/dns/queries.json`
- `fixtures/flows/events.json`

Os arquivos de `fixtures` continuam disponiveis para testes, mas a descoberta de dispositivos agora roda em modo `live` por padrao. Para operacao real da LAN:

- `DISCOVERY_SOURCE=live`
- `DISCOVERY_PING_ENABLED=true`
- `DISCOVERY_FINGERPRINT_ENABLED=true`
- `DISCOVERY_FINGERPRINT_TIMEOUT_MS=180`
- `DISCOVERY_VENDOR_DB=/usr/share/ieee-data/oui.txt`

Para DNS real com `AdGuard Home`:

- `DNS_SOURCE=adguard_file`
- `DNS_ADGUARD_QUERYLOG=/adguard-work/querylog.json`
- `DNS_RESOLVER_NAME=adguardhome`

Observacao:

- o collector aceita tanto JSON array quanto JSON lines
- se o querylog ainda nao existir, ele registra aviso e continua rodando
- o sistema tenta correlacionar o evento com o dispositivo por `client_ip` e `client_name`

Se quiser restringir a descoberta a interfaces especificas, configure:

- `DISCOVERY_INTERFACE_ALLOWLIST=enp67s0,wlp0s20f3`

## Troubleshooting

- se a API subir sem dados, confira os logs dos collectors
- se o DNS real nao aparecer, valide `DNS_SOURCE`, `DNS_ADGUARD_QUERYLOG` e se o arquivo de querylog existe dentro do container `dns-collector`
- se o dashboard nao subir, valide o build Angular e o container `dashboard`
- se os collectors nao conectarem, valide `NATS_URL`
- se a API falhar ao persistir, valide `DATABASE_URL` e o health do `postgres`
- se quiser incluir ferramentas externas, suba o profile `tooling`

```bash
docker compose --profile tooling up --build
```

## Frontend em desenvolvimento

```bash
cd apps/dashboard
npm install
npm start
```

O painel usa `proxy.conf.json` para falar com a API local sem precisar habilitar CORS no backend.
