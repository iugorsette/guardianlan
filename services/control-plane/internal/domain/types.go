package domain

import "time"

type Device struct {
	ID         string    `json:"id"`
	MAC        string    `json:"mac"`
	IPs        []string  `json:"ips"`
	Hostname   string    `json:"hostname"`
	Vendor     string    `json:"vendor"`
	DeviceType string    `json:"device_type"`
	ProfileID  string    `json:"profile_id"`
	Managed    bool      `json:"managed"`
	RiskScore  int       `json:"risk_score"`
	FirstSeen  time.Time `json:"first_seen_at"`
	LastSeen   time.Time `json:"last_seen_at"`
}

type DeviceEvent struct {
	ID         string    `json:"id"`
	MAC        string    `json:"mac"`
	IPs        []string  `json:"ips"`
	Hostname   string    `json:"hostname"`
	Vendor     string    `json:"vendor"`
	DeviceType string    `json:"device_type"`
	ProfileID  string    `json:"profile_id"`
	Managed    bool      `json:"managed"`
	ObservedAt time.Time `json:"observed_at"`
}

type DNSEvent struct {
	ID         string    `json:"id,omitempty"`
	DeviceID   string    `json:"device_id"`
	Query      string    `json:"query"`
	Domain     string    `json:"domain"`
	Category   string    `json:"category"`
	Resolver   string    `json:"resolver"`
	Blocked    bool      `json:"blocked"`
	ObservedAt time.Time `json:"observed_at"`
}

type FlowEvent struct {
	ID         string    `json:"id,omitempty"`
	DeviceID   string    `json:"device_id"`
	SrcIP      string    `json:"src_ip"`
	DstIP      string    `json:"dst_ip"`
	DstPort    int       `json:"dst_port"`
	Protocol   string    `json:"protocol"`
	BytesIn    int64     `json:"bytes_in"`
	BytesOut   int64     `json:"bytes_out"`
	ObservedAt time.Time `json:"observed_at"`
}

type Observation struct {
	ID          string                 `json:"id"`
	DeviceID    string                 `json:"device_id"`
	Source      string                 `json:"source"`
	Kind        string                 `json:"kind"`
	Severity    string                 `json:"severity"`
	Summary     string                 `json:"summary"`
	EvidenceRef map[string]any         `json:"evidence_ref"`
	ObservedAt  time.Time              `json:"observed_at"`
}

type Alert struct {
	ID           string         `json:"id"`
	DeviceID     string         `json:"device_id"`
	Type         string         `json:"type"`
	Severity     string         `json:"severity"`
	Title        string         `json:"title"`
	Status       string         `json:"status"`
	Evidence     map[string]any `json:"evidence"`
	CreatedAt    time.Time      `json:"created_at"`
	Acknowledged *time.Time     `json:"acknowledged_at,omitempty"`
}

type ProfileUpdateRequest struct {
	ProfileID string `json:"profile_id"`
}

