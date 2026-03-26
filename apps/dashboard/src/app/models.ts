export interface Device {
  id: string;
  mac: string;
  ips: string[];
  hostname: string;
  vendor: string;
  device_type: string;
  profile_id: string;
  managed: boolean;
  risk_score: number;
  first_seen_at: string;
  last_seen_at: string;
}

export interface Alert {
  id: string;
  device_id: string;
  type: string;
  severity: 'low' | 'medium' | 'high' | 'critical' | string;
  title: string;
  status: string;
  evidence: Record<string, unknown>;
  created_at: string;
  acknowledged_at?: string;
}

export interface DnsEvent {
  id?: string;
  device_id: string;
  query: string;
  domain: string;
  category: string;
  resolver: string;
  blocked: boolean;
  observed_at: string;
}

export interface FlowEvent {
  id?: string;
  device_id: string;
  src_ip: string;
  dst_ip: string;
  dst_port: number;
  protocol: string;
  bytes_in: number;
  bytes_out: number;
  observed_at: string;
}

export interface DashboardSnapshot {
  devices: Device[] | null;
  alerts: Alert[] | null;
  dnsEvents: DnsEvent[] | null;
  flowEvents: FlowEvent[] | null;
}

