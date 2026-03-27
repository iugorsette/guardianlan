package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
	"github.com/sette/guardian-lan/services/control-plane/internal/repository"
)

type AlertPublisher interface {
	PublishAlert(context.Context, domain.Alert) error
}

type DNSPolicySyncer interface {
	SyncDevice(context.Context, domain.Device, domain.Profile, domain.DNSPolicy, []domain.Device, map[string]domain.Profile) error
	SyncAll(context.Context, []domain.Device, map[string]domain.Profile) error
}

type Orchestrator struct {
	store               repository.Store
	alertPublisher      AlertPublisher
	expectedDNSResolver string
	dnsPolicySyncer     DNSPolicySyncer
}

func NewOrchestrator(store repository.Store, alertPublisher AlertPublisher, expectedDNSResolver string, dnsPolicySyncer DNSPolicySyncer) *Orchestrator {
	if alertPublisher == nil {
		alertPublisher = noopAlertPublisher{}
	}

	return &Orchestrator{
		store:               store,
		alertPublisher:      alertPublisher,
		expectedDNSResolver: expectedDNSResolver,
		dnsPolicySyncer:     dnsPolicySyncer,
	}
}

func (o *Orchestrator) HandleDeviceEvent(ctx context.Context, event domain.DeviceEvent, discovered bool) error {
	observedAt := normalizeTime(event.ObservedAt)
	profileID := event.ProfileID
	if profileID == "" {
		profileID = defaultProfileForDevice(event.DeviceType)
	}

	device := domain.Device{
		ID:         event.ID,
		MAC:        event.MAC,
		IPs:        event.IPs,
		Hostname:   event.Hostname,
		Vendor:     event.Vendor,
		DeviceType: normalizeDeviceType(event.DeviceType),
		ProfileID:  profileID,
		Managed:    event.Managed,
		RiskScore:  domain.ScoreDevice(event),
		FirstSeen:  observedAt,
		LastSeen:   observedAt,
	}

	stored, created, err := o.store.UpsertDevice(ctx, device)
	if err != nil {
		return err
	}

	observation := domain.Observation{
		ID:          uuid.NewString(),
		DeviceID:    stored.ID,
		Source:      "discovery-collector",
		Kind:        ternary(discovered, "device_discovered", "device_updated"),
		Severity:    ternary(discovered && created, "medium", "info"),
		Summary:     fmt.Sprintf("Device %s observed with profile %s", stored.HostnameOrID(), stored.ProfileID),
		EvidenceRef: discoveryEvidence(stored, event.Evidence),
		ObservedAt:  observedAt,
	}
	if err := o.store.StoreObservation(ctx, observation); err != nil {
		return err
	}

	if discovered && created && !stored.Managed {
		_, err := o.raiseAlert(ctx, domain.Alert{
			DeviceID:  stored.ID,
			Type:      "unknown_device",
			Severity:  "medium",
			Title:     fmt.Sprintf("New unmanaged device detected: %s", stored.HostnameOrID()),
			Status:    "open",
			Evidence:  map[string]any{"ips": stored.IPs, "vendor": stored.Vendor},
			CreatedAt: observedAt,
		})
		return err
	}

	_ = o.syncDeviceDNSPolicy(ctx, stored)
	return nil
}

func discoveryEvidence(device domain.Device, evidence domain.DeviceEvidence) map[string]any {
	result := map[string]any{
		"hostname":    device.Hostname,
		"ips":         device.IPs,
		"vendor":      device.Vendor,
		"device_type": device.DeviceType,
	}

	if len(evidence.OpenPorts) > 0 {
		result["open_ports"] = evidence.OpenPorts
	}
	if len(evidence.Services) > 0 {
		result["services"] = evidence.Services
	}
	if len(evidence.CandidateSnapshotURLs) > 0 {
		result["candidate_snapshot_urls"] = evidence.CandidateSnapshotURLs
	}
	if len(evidence.CandidateStreamURLs) > 0 {
		result["candidate_stream_urls"] = evidence.CandidateStreamURLs
	}
	if evidence.PreviewSupported {
		result["preview_supported"] = evidence.PreviewSupported
	}
	if evidence.PreviewRequiresAuth {
		result["preview_requires_auth"] = evidence.PreviewRequiresAuth
	}
	if evidence.Confidence != "" {
		result["confidence"] = evidence.Confidence
	}

	return result
}

func (o *Orchestrator) HandleDNSEvent(ctx context.Context, event domain.DNSEvent) error {
	resolvedDeviceID, err := o.resolveDeviceIDForDNS(ctx, event)
	if err != nil {
		return err
	}
	event.DeviceID = resolvedDeviceID
	event.ID = dnsEventID(event)
	event.ObservedAt = normalizeTime(event.ObservedAt)
	if err := o.ensureDeviceExists(ctx, event.DeviceID, event.ClientName, event.ClientIP, event.ObservedAt); err != nil {
		return err
	}
	if err := o.store.StoreDNSEvent(ctx, event); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil
		}
		return err
	}

	device, err := o.store.GetDevice(ctx, event.DeviceID)
	if err != nil {
		return err
	}

	profile, err := o.store.GetProfile(ctx, device.ProfileID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	policy := domain.MergeDNSPolicies(profile.DNSPolicy, device.DNSPolicyOverride)

	if o.expectedDNSResolver != "" && event.Resolver != "" && !strings.EqualFold(event.Resolver, o.expectedDNSResolver) {
		_, err := o.raiseAlert(ctx, domain.Alert{
			DeviceID:  event.DeviceID,
			Type:      "dns_bypass",
			Severity:  "high",
			Title:     fmt.Sprintf("Potential DNS bypass detected for %s", device.HostnameOrID()),
			Status:    "open",
			Evidence:  map[string]any{"resolver": event.Resolver, "expected_resolver": o.expectedDNSResolver, "query": event.Query, "client_ip": event.ClientIP},
			CreatedAt: event.ObservedAt,
		})
		if err != nil {
			return err
		}
	}

	violations := dnsPolicyAlerts(device, event, policy)
	for _, alert := range violations {
		if _, err := o.raiseAlert(ctx, alert); err != nil {
			return err
		}

		observation := domain.Observation{
			ID:          uuid.NewString(),
			DeviceID:    event.DeviceID,
			Source:      "dns-collector",
			Kind:        alert.Type,
			Severity:    alert.Severity,
			Summary:     alert.Title,
			EvidenceRef: alert.Evidence,
			ObservedAt:  event.ObservedAt,
		}
		if err := o.store.StoreObservation(ctx, observation); err != nil {
			return err
		}
	}

	return nil
}

func (o *Orchestrator) HandleFlowEvent(ctx context.Context, event domain.FlowEvent) error {
	event.ID = flowEventID(event)
	event.ObservedAt = normalizeTime(event.ObservedAt)
	if err := o.ensureDeviceExists(ctx, event.DeviceID, "", event.SrcIP, event.ObservedAt); err != nil {
		return err
	}
	if err := o.store.StoreFlowEvent(ctx, event); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil
		}
		return err
	}

	if event.DstPort == 554 || event.DstPort == 8554 || event.DstPort == 23 {
		_, err := o.raiseAlert(ctx, domain.Alert{
			DeviceID:  event.DeviceID,
			Type:      "suspicious_flow",
			Severity:  "medium",
			Title:     fmt.Sprintf("Sensitive service exposure seen for %s", event.DeviceID),
			Status:    "open",
			Evidence:  map[string]any{"dst_ip": event.DstIP, "dst_port": event.DstPort, "protocol": event.Protocol},
			CreatedAt: event.ObservedAt,
		})
		if err != nil {
			return err
		}

		observation := domain.Observation{
			ID:       uuid.NewString(),
			DeviceID: event.DeviceID,
			Source:   "flow-collector",
			Kind:     "suspicious_flow",
			Severity: "medium",
			Summary:  fmt.Sprintf("Observed traffic to port %d", event.DstPort),
			EvidenceRef: map[string]any{
				"dst_ip":   event.DstIP,
				"dst_port": event.DstPort,
			},
			ObservedAt: event.ObservedAt,
		}
		return o.store.StoreObservation(ctx, observation)
	}

	return nil
}

func (o *Orchestrator) UpdateDeviceProfile(ctx context.Context, id string, profileID string) (domain.Device, error) {
	device, err := o.store.UpdateDeviceProfile(ctx, id, profileID)
	if err != nil {
		return domain.Device{}, err
	}

	return device, o.syncDeviceDNSPolicy(ctx, device)
}

func (o *Orchestrator) UpdateDeviceDNSPolicy(ctx context.Context, id string, policy domain.DNSPolicy) (domain.Device, error) {
	device, err := o.store.UpdateDeviceDNSPolicy(ctx, id, domain.NormalizeDNSPolicy(policy))
	if err != nil {
		return domain.Device{}, err
	}

	return device, o.syncDeviceDNSPolicy(ctx, device)
}

func (o *Orchestrator) UpdateDeviceName(ctx context.Context, id string, displayName string) (domain.Device, error) {
	device, err := o.store.UpdateDeviceName(ctx, id, displayName)
	if err != nil {
		return domain.Device{}, err
	}

	return device, o.syncDeviceDNSPolicy(ctx, device)
}

func (o *Orchestrator) SyncDNSPolicies(ctx context.Context) error {
	if o.dnsPolicySyncer == nil {
		return nil
	}

	devices, profiles, err := o.loadDevicesAndProfiles(ctx)
	if err != nil {
		return err
	}

	return o.dnsPolicySyncer.SyncAll(ctx, devices, profiles)
}

func (o *Orchestrator) raiseAlert(ctx context.Context, alert domain.Alert) (domain.Alert, error) {
	if alert.Status == "" {
		alert.Status = "open"
	}
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}
	if alert.Evidence == nil {
		alert.Evidence = map[string]any{}
	}
	alert.ID = alertID(alert)

	if err := o.store.CreateAlert(ctx, alert); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return alert, nil
		}
		return domain.Alert{}, err
	}

	if err := o.alertPublisher.PublishAlert(ctx, alert); err != nil {
		return domain.Alert{}, err
	}

	return alert, nil
}

type noopAlertPublisher struct{}

func (noopAlertPublisher) PublishAlert(context.Context, domain.Alert) error {
	return nil
}

func fallbackID(id string) string {
	if id != "" {
		return id
	}
	return uuid.NewString()
}

func stableID(parts ...string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(strings.Join(parts, "|"))).String()
}

func dnsEventID(event domain.DNSEvent) string {
	if event.ID != "" {
		return event.ID
	}

	return stableID(
		"dns",
		event.DeviceID,
		event.ClientIP,
		event.ClientName,
		event.Query,
		event.Domain,
		event.Category,
		event.Resolver,
		strconv.FormatBool(event.Blocked),
		normalizeTime(event.ObservedAt).Format(time.RFC3339Nano),
	)
}

func flowEventID(event domain.FlowEvent) string {
	if event.ID != "" {
		return event.ID
	}

	return stableID(
		"flow",
		event.DeviceID,
		event.SrcIP,
		event.DstIP,
		strconv.Itoa(event.DstPort),
		event.Protocol,
		strconv.FormatInt(event.BytesIn, 10),
		strconv.FormatInt(event.BytesOut, 10),
		normalizeTime(event.ObservedAt).Format(time.RFC3339Nano),
	)
}

func alertID(alert domain.Alert) string {
	if alert.ID != "" {
		return alert.ID
	}

	return stableID(
		"alert",
		alert.DeviceID,
		alert.Type,
		alert.Title,
		alert.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

func (o *Orchestrator) syncDeviceDNSPolicy(ctx context.Context, device domain.Device) error {
	if o.dnsPolicySyncer == nil {
		return nil
	}

	profile, err := o.store.GetProfile(ctx, device.ProfileID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	mergedPolicy := domain.MergeDNSPolicies(profile.DNSPolicy, device.DNSPolicyOverride)
	devices, profiles, err := o.loadDevicesAndProfiles(ctx)
	if err != nil {
		return err
	}

	return o.dnsPolicySyncer.SyncDevice(ctx, device, profile, mergedPolicy, devices, profiles)
}

func (o *Orchestrator) loadDevicesAndProfiles(ctx context.Context) ([]domain.Device, map[string]domain.Profile, error) {
	devices, err := o.store.ListDevices(ctx)
	if err != nil {
		return nil, nil, err
	}

	profilesList, err := o.store.ListProfiles(ctx)
	if err != nil {
		return nil, nil, err
	}

	profiles := make(map[string]domain.Profile, len(profilesList))
	for _, profile := range profilesList {
		profiles[profile.ID] = profile
	}

	return devices, profiles, nil
}

func (o *Orchestrator) ensureDeviceExists(ctx context.Context, deviceID string, clientName string, clientIP string, observedAt time.Time) error {
	if deviceID == "" {
		return nil
	}

	_, err := o.store.GetDevice(ctx, deviceID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	_, _, err = o.store.UpsertDevice(ctx, domain.Device{
		ID:         deviceID,
		Hostname:   fallbackHostname(clientName, deviceID),
		IPs:        placeholderIPs(clientIP),
		DeviceType: "unknown",
		ProfileID:  "guest",
		Managed:    false,
		RiskScore:  25,
		FirstSeen:  observedAt,
		LastSeen:   observedAt,
	})
	return err
}

func (o *Orchestrator) resolveDeviceIDForDNS(ctx context.Context, event domain.DNSEvent) (string, error) {
	if event.DeviceID != "" {
		if _, err := o.store.GetDevice(ctx, event.DeviceID); err == nil {
			return event.DeviceID, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}

	devices, err := o.store.ListDevices(ctx)
	if err != nil {
		return "", err
	}

	for _, device := range devices {
		if event.ClientIP != "" {
			for _, ip := range device.IPs {
				if ip == event.ClientIP {
					return device.ID, nil
				}
			}
		}

		if event.ClientName != "" && strings.EqualFold(event.ClientName, device.Hostname) {
			return device.ID, nil
		}

		if event.ClientName != "" && strings.EqualFold(event.ClientName, device.DisplayName) {
			return device.ID, nil
		}
	}

	if event.ClientIP != "" {
		return "device-ip-" + sanitizeForID(event.ClientIP), nil
	}
	if event.ClientName != "" {
		return "device-name-" + sanitizeForID(event.ClientName), nil
	}

	return event.DeviceID, nil
}

func defaultProfileForDevice(deviceType string) string {
	switch strings.ToLower(deviceType) {
	case "camera", "iot":
		return "iot"
	default:
		return "guest"
	}
}

func dnsPolicyAlerts(device domain.Device, event domain.DNSEvent, policy domain.DNSPolicy) []domain.Alert {
	var alerts []domain.Alert

	baseEvidence := map[string]any{
		"domain":      event.Domain,
		"query":       event.Query,
		"category":    event.Category,
		"blocked":     event.Blocked,
		"resolver":    event.Resolver,
		"client_ip":   event.ClientIP,
		"client_name": event.ClientName,
		"profile_id":  device.ProfileID,
	}

	if len(policy.AllowedDomains) > 0 && !domain.DomainMatches(policy.AllowedDomains, event.Domain) {
		alerts = append(alerts, domain.Alert{
			DeviceID:  device.ID,
			Type:      "whitelist_violation",
			Severity:  "high",
			Title:     fmt.Sprintf("%s accessed domain outside whitelist: %s", device.HostnameOrID(), event.Domain),
			Status:    "open",
			Evidence:  withPolicyEvidence(baseEvidence, policy),
			CreatedAt: event.ObservedAt,
		})
	}

	if domain.DomainMatches(policy.BlockedDomains, event.Domain) {
		title := fmt.Sprintf("%s attempted blocked domain: %s", device.HostnameOrID(), event.Domain)
		if event.Blocked {
			title = fmt.Sprintf("%s tried blocked domain and DNS blocked it: %s", device.HostnameOrID(), event.Domain)
		}

		alerts = append(alerts, domain.Alert{
			DeviceID:  device.ID,
			Type:      "blocked_domain_attempt",
			Severity:  "high",
			Title:     title,
			Status:    "open",
			Evidence:  withPolicyEvidence(baseEvidence, policy),
			CreatedAt: event.ObservedAt,
		})
	}

	if domain.ValueMatches(policy.BlockedCategories, event.Category) {
		title := fmt.Sprintf("%s matched blocked category %s on %s", device.HostnameOrID(), event.Category, event.Domain)
		if event.Blocked {
			title = fmt.Sprintf("%s tried blocked category %s and DNS blocked it", device.HostnameOrID(), event.Category)
		}
		alerts = append(alerts, domain.Alert{
			DeviceID:  device.ID,
			Type:      "policy_violation",
			Severity:  "high",
			Title:     title,
			Status:    "open",
			Evidence:  withPolicyEvidence(baseEvidence, policy),
			CreatedAt: event.ObservedAt,
		})
	}

	if len(policy.BlockedCategories) == 0 && strings.EqualFold(event.Category, "adult") && !event.Blocked {
		alerts = append(alerts, domain.Alert{
			DeviceID:  device.ID,
			Type:      "policy_violation",
			Severity:  "high",
			Title:     fmt.Sprintf("Adult domain observed for %s", device.HostnameOrID()),
			Status:    "open",
			Evidence:  withPolicyEvidence(baseEvidence, policy),
			CreatedAt: event.ObservedAt,
		})
	}

	return alerts
}

func withPolicyEvidence(base map[string]any, policy domain.DNSPolicy) map[string]any {
	result := map[string]any{}
	for key, value := range base {
		result[key] = value
	}
	result["policy"] = policy
	return result
}

func placeholderIPs(clientIP string) []string {
	if clientIP == "" {
		return nil
	}
	if net.ParseIP(clientIP) == nil {
		return nil
	}
	return []string{clientIP}
}

func fallbackHostname(clientName string, deviceID string) string {
	clientName = strings.TrimSpace(clientName)
	if clientName != "" {
		return clientName
	}
	return deviceID
}

func sanitizeForID(value string) string {
	replacer := strings.NewReplacer(".", "-", ":", "-", " ", "-", "/", "-", "\\", "-")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(value)))
}

func normalizeDeviceType(deviceType string) string {
	if deviceType == "" {
		return "unknown"
	}
	return strings.ToLower(deviceType)
}

func ternary[T any](condition bool, whenTrue T, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
