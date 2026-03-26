CREATE TABLE IF NOT EXISTS profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  schedule JSONB NOT NULL DEFAULT '{}'::jsonb,
  dns_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
  alert_policy JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY,
  mac TEXT NOT NULL DEFAULT '',
  ips JSONB NOT NULL DEFAULT '[]'::jsonb,
  display_name TEXT NOT NULL DEFAULT '',
  hostname TEXT NOT NULL DEFAULT '',
  vendor TEXT NOT NULL DEFAULT '',
  device_type TEXT NOT NULL DEFAULT 'unknown',
  profile_id TEXT NOT NULL DEFAULT 'guest' REFERENCES profiles(id),
  managed BOOLEAN NOT NULL DEFAULT FALSE,
  risk_score INTEGER NOT NULL DEFAULT 0,
  first_seen_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dns_events (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  query TEXT NOT NULL,
  domain TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT 'unknown',
  resolver TEXT NOT NULL DEFAULT '',
  blocked BOOLEAN NOT NULL DEFAULT FALSE,
  observed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS flow_events (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  src_ip TEXT NOT NULL,
  dst_ip TEXT NOT NULL,
  dst_port INTEGER NOT NULL,
  protocol TEXT NOT NULL DEFAULT 'tcp',
  bytes_in BIGINT NOT NULL DEFAULT 0,
  bytes_out BIGINT NOT NULL DEFAULT 0,
  observed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS observations (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  source TEXT NOT NULL,
  kind TEXT NOT NULL,
  severity TEXT NOT NULL,
  summary TEXT NOT NULL,
  evidence_ref JSONB NOT NULL DEFAULT '{}'::jsonb,
  observed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  severity TEXT NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  acknowledged_at TIMESTAMPTZ
);

INSERT INTO profiles (id, name, kind, schedule, dns_policy, alert_policy)
VALUES
  ('adult', 'Adulto', 'adult', '{}'::jsonb, '{"safe_search": false}'::jsonb, '{}'::jsonb),
  ('child', 'Crianca', 'child', '{"curfew": "21:00-07:00"}'::jsonb, '{"safe_search": true, "blocked_categories": ["adult", "gambling"]}'::jsonb, '{"notify_on_bypass": true}'::jsonb),
  ('iot', 'IoT', 'iot', '{}'::jsonb, '{"blocked_categories": ["newly_registered_domains"]}'::jsonb, '{"notify_on_exposure": true}'::jsonb),
  ('guest', 'Visitante', 'guest', '{}'::jsonb, '{"blocked_categories": ["adult"]}'::jsonb, '{"notify_on_new_device": true}'::jsonb)
ON CONFLICT (id) DO NOTHING;
