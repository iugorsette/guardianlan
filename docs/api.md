# API

Base URL local: `http://localhost:8080`

## Endpoints

### `GET /healthz`

Retorna o status basico da API.

### `GET /profiles`

Lista os perfis base disponiveis com suas politicas DNS padrao.

### `GET /devices`

Lista dispositivos conhecidos, ordenados por `last_seen_at` descendente.

### `GET /devices/:id`

Retorna um dispositivo especifico.

### `POST /devices/:id/profile`

Atualiza o perfil associado ao dispositivo.

Payload:

```json
{
  "profile_id": "child"
}
```

### `POST /devices/:id/name`

Atualiza o nome amigavel do dispositivo.

Payload:

```json
{
  "display_name": "Tablet do Pedro"
}
```

### `POST /devices/:id/dns-policy`

Atualiza o override de politica DNS do dispositivo.

Payload:

```json
{
  "dns_policy": {
    "safe_search": true,
    "blocked_categories": ["adult", "gambling"],
    "blocked_domains": ["xvideos.com"],
    "allowed_domains": ["googleclassroom.com", "escola.local"]
  }
}
```

### `POST /integrations/adguard/sync`

Forca uma resincronizacao completa entre as politicas do `Guardian LAN` e o `AdGuard Home`.

Resposta:

```json
{
  "status": "synced"
}
```

### `GET /activity/dns`

Lista eventos DNS recentes.

Query params:

- `limit` opcional, padrao `50`

Campos relevantes retornados em cada evento:

- `device_id`
- `client_ip`
- `client_name`
- `domain`
- `category`
- `resolver`
- `blocked`

### `GET /activity/flows`

Lista eventos de fluxo recentes.

Query params:

- `limit` opcional, padrao `50`

### `GET /alerts`

Lista alertas recentes.

Query params:

- `status` opcional, por exemplo `open`
- `limit` opcional, padrao `50`

### `POST /alerts/:id/ack`

Marca um alerta como reconhecido.

## Semantica

- A API e local e nao expoe payloads brutos de rede.
- Eventos e alertas retornam metadados normalizados e referencia de evidencia em JSON quando existir.
- Perfis existentes na base inicial: `adult`, `child`, `iot`, `guest`.
- A politica efetiva de DNS vem do perfil base mais o override opcional por dispositivo.
- Quando `ADGUARD_ENABLED=true`, atualizacoes de nome, perfil e politica DNS tentam sincronizar enforcement real com o `AdGuard Home`.
