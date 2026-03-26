# Events

Todos os eventos trafegam em `JSON` via `NATS`.

## Subjects

- `network.device.discovered`
- `network.device.updated`
- `network.dns.query_observed`
- `network.flow.observed`
- `network.alert.raised`

## Device event

```json
{
  "id": "device-baby-cam",
  "mac": "AA:BB:CC:DD:EE:01",
  "ips": ["192.168.1.20"],
  "hostname": "baby-cam",
  "vendor": "GenericCam",
  "device_type": "camera",
  "profile_id": "iot",
  "managed": false,
  "observed_at": "2026-03-26T18:00:00Z"
}
```

## DNS event

```json
{
  "device_id": "device-kid-tablet",
  "query": "youtube.com",
  "domain": "youtube.com",
  "category": "streaming",
  "resolver": "adguardhome",
  "blocked": false,
  "observed_at": "2026-03-26T18:03:00Z"
}
```

## Flow event

```json
{
  "device_id": "device-baby-cam",
  "src_ip": "192.168.1.20",
  "dst_ip": "203.0.113.44",
  "dst_port": 554,
  "protocol": "tcp",
  "bytes_in": 4096,
  "bytes_out": 8192,
  "observed_at": "2026-03-26T18:05:00Z"
}
```

## Alert event

```json
{
  "id": "alert-123",
  "device_id": "device-baby-cam",
  "type": "camera_exposure",
  "severity": "high",
  "title": "Camera talking over RTSP to an external address",
  "status": "open",
  "created_at": "2026-03-26T18:05:01Z"
}
```

## Versionamento

- Mudancas compativeis adicionam campos opcionais.
- Mudancas incompativeis exigem novo subject ou versao declarada no payload.
- O `control-plane` deve ignorar campos desconhecidos.

