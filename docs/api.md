# API

Base URL local: `http://localhost:8080`

## Endpoints

### `GET /healthz`

Retorna o status basico da API.

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

### `GET /activity/dns`

Lista eventos DNS recentes.

Query params:

- `limit` opcional, padrao `50`

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

