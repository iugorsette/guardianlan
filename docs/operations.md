# Operations

## Subir a stack

Fluxo recomendado do modo atual (`Observer`):

```bash
./scripts/observer/up.sh
```

ou:

```bash
make observer-up
```

Esse comando:

- cria `.env` se faltar
- sobe `nats`, `postgres`, `control-plane`, `dashboard` e collectors
- deixa a aplicacao pronta para inventario e alertas observacionais

Importante:

- `docker compose build` sozinho nao sobe nada
- para executar a stack, use `docker compose up`, `make observer-up` ou o script acima

Fluxo manual:

```bash
cp .env.example .env
docker compose up --build
```

## Observer mode

O fluxo principal de hoje e `Observer Mode`, sem exigir mudanca de topologia da rede:

- descoberta live da LAN
- inventario
- classificacao
- alertas locais
- fontes opcionais de DNS, fluxo e endpoint

Detalhes em [docs/observer-mode.md](/home/sette/github/parental-local/docs/observer-mode.md).

## Parar e limpar

Para parar apenas o modo `Observer`:

```bash
make observer-down
```

Ou para derrubar tudo com volumes:

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

Para telemetria DNS opcional com `AdGuard Home`:

- `ADGUARD_ENABLED=true`
- `ADGUARD_URL=http://adguardhome:3000/control`
- `ADGUARD_USERNAME=admin`
- `ADGUARD_PASSWORD=sua-senha`
- `ADGUARD_WEB_BIND_IP=127.0.0.1`
- `ADGUARD_DNS_BIND_IP=127.0.0.1`
- `DNS_SOURCE=adguard_file`
- `DNS_ADGUARD_QUERYLOG=/adguard-work/querylog.json`
- `DNS_RESOLVER_NAME=adguardhome`

Observacao:

- o collector aceita tanto JSON array quanto JSON lines
- se o querylog ainda nao existir, ele registra aviso e continua rodando
- o sistema tenta correlacionar o evento com o dispositivo por `client_ip` e `client_name`
- o `control-plane` passa a sincronizar clientes e regras customizadas com o `AdGuard Home`
- para a narrativa atual do produto, isso deve ser tratado como `fonte observacional`, nao como bloqueio garantido de toda a rede

Se quiser restringir a descoberta a interfaces especificas, configure:

- `DISCOVERY_INTERFACE_ALLOWLIST=enp67s0,wlp0s20f3`

## Troubleshooting

- se a API subir sem dados, confira os logs dos collectors
- se o DNS real nao aparecer, valide `DNS_SOURCE`, `DNS_ADGUARD_QUERYLOG` e se o arquivo de querylog existe dentro do container `dns-collector`
- se o alerta DNS nao aparecer, confirme que existe uma fonte real de DNS e que o `dns-collector` consegue ler o `querylog`
- se o `adguardhome` nao subir por conflito na porta `53`, confirme se o mapeamento esta preso em `127.0.0.1:53` e se outro processo local nao ocupou esse endereco
- se o dashboard nao subir, valide o build Angular e o container `dashboard`
- se os collectors nao conectarem, valide `NATS_URL`
- se a API falhar ao persistir, valide `DATABASE_URL` e o health do `postgres`
- se quiser incluir ferramentas externas, suba o profile `tooling`

```bash
docker compose --profile tooling up --build
```

## Fluxo minimo para alerta de dominio nesta maquina

1. Ative as variaveis `ADGUARD_*` e `DNS_SOURCE=adguard_file` no `.env`.
2. Suba o stack de rede:

```bash
make api-network
```

3. Garanta que o `dns-collector` consegue ler o `querylog` do `AdGuard Home`.
4. Se o objetivo for observar o dominio a partir desta maquina, faca o navegador ou o sistema consultarem esse DNS observado.
5. Desative `Secure DNS` ou `DNS over HTTPS` no navegador usado para o teste.
6. No painel, salve a watchlist do dispositivo.
7. Gere uma consulta e espere o polling do collector.

Exemplo de validacao:

```bash
dig @127.0.0.1 xvideos.com
```

No modo atual, isso serve para `observar e alertar`, nao para prometer bloqueio automatico da rede inteira.

## Frontend em desenvolvimento

```bash
cd apps/dashboard
npm install
npm start
```

O painel usa `proxy.conf.json` para falar com a API local sem precisar habilitar CORS no backend.
