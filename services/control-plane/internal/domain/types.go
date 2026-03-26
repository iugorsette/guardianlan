package domain

import "time"

type DNSPolicy struct {
	SafeSearch        bool     `json:"safe_search"`
	BlockedCategories []string `json:"blocked_categories,omitempty"`
	BlockedDomains    []string `json:"blocked_domains,omitempty"`
	AllowedDomains    []string `json:"allowed_domains,omitempty"`
}

type AlertPolicy struct {
	NotifyOnBypass    bool `json:"notify_on_bypass"`
	NotifyOnExposure  bool `json:"notify_on_exposure"`
	NotifyOnNewDevice bool `json:"notify_on_new_device"`
}

type Profile struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	Schedule    map[string]any `json:"schedule"`
	DNSPolicy   DNSPolicy      `json:"dns_policy"`
	AlertPolicy AlertPolicy    `json:"alert_policy"`
}

type DeviceEvidence struct {
	OpenPorts             []int    `json:"open_ports,omitempty"`
	Services              []string `json:"services,omitempty"`
	CandidateSnapshotURLs []string `json:"candidate_snapshot_urls,omitempty"`
	CandidateStreamURLs   []string `json:"candidate_stream_urls,omitempty"`
	PreviewSupported      bool     `json:"preview_supported,omitempty"`
	PreviewRequiresAuth   bool     `json:"preview_requires_auth,omitempty"`
	Confidence            string   `json:"confidence,omitempty"`
}

type Device struct {
	ID                string    `json:"id"`
	MAC               string    `json:"mac"`
	IPs               []string  `json:"ips"`
	DisplayName       string    `json:"display_name"`
	Hostname          string    `json:"hostname"`
	Vendor            string    `json:"vendor"`
	DeviceType        string    `json:"device_type"`
	ProfileID         string    `json:"profile_id"`
	Managed           bool      `json:"managed"`
	RiskScore         int       `json:"risk_score"`
	DNSPolicyOverride DNSPolicy `json:"dns_policy_override"`
	FirstSeen         time.Time `json:"first_seen_at"`
	LastSeen          time.Time `json:"last_seen_at"`
}

type DeviceEvent struct {
	ID         string         `json:"id"`
	MAC        string         `json:"mac"`
	IPs        []string       `json:"ips"`
	Hostname   string         `json:"hostname"`
	Vendor     string         `json:"vendor"`
	DeviceType string         `json:"device_type"`
	ProfileID  string         `json:"profile_id"`
	Managed    bool           `json:"managed"`
	Evidence   DeviceEvidence `json:"evidence"`
	ObservedAt time.Time      `json:"observed_at"`
}

type DNSEvent struct {
	ID         string    `json:"id,omitempty"`
	DeviceID   string    `json:"device_id"`
	ClientIP   string    `json:"client_ip,omitempty"`
	ClientName string    `json:"client_name,omitempty"`
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
	ID          string         `json:"id"`
	DeviceID    string         `json:"device_id"`
	Source      string         `json:"source"`
	Kind        string         `json:"kind"`
	Severity    string         `json:"severity"`
	Summary     string         `json:"summary"`
	EvidenceRef map[string]any `json:"evidence_ref"`
	ObservedAt  time.Time      `json:"observed_at"`
}

type DeviceInsight struct {
	DeviceID   string         `json:"device_id"`
	Source     string         `json:"source"`
	Kind       string         `json:"kind"`
	Severity   string         `json:"severity"`
	Summary    string         `json:"summary"`
	Evidence   map[string]any `json:"evidence"`
	ObservedAt time.Time      `json:"observed_at"`
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

type DeviceNameUpdateRequest struct {
	DisplayName string `json:"display_name"`
}

type DeviceDNSPolicyUpdateRequest struct {
	DNSPolicy DNSPolicy `json:"dns_policy"`
}
