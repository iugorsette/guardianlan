package service

import (
	"context"
	"errors"
	"fmt"
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

type Orchestrator struct {
	store               repository.Store
	alertPublisher      AlertPublisher
	expectedDNSResolver string
}

func NewOrchestrator(store repository.Store, alertPublisher AlertPublisher, expectedDNSResolver string) *Orchestrator {
	if alertPublisher == nil {
		alertPublisher = noopAlertPublisher{}
	}

	return &Orchestrator{
		store:               store,
		alertPublisher:      alertPublisher,
		expectedDNSResolver: expectedDNSResolver,
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
		ID:         uuid.NewString(),
		DeviceID:   stored.ID,
		Source:     "discovery-collector",
		Kind:       ternary(discovered, "device_discovered", "device_updated"),
		Severity:   ternary(discovered && created, "medium", "info"),
		Summary:    fmt.Sprintf("Device %s observed with profile %s", stored.HostnameOrID(), stored.ProfileID),
		EvidenceRef: discoveryEvidence(stored, event.Evidence),
		ObservedAt: observedAt,
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
	event.ID = dnsEventID(event)
	event.ObservedAt = normalizeTime(event.ObservedAt)
	if err := o.ensureDeviceExists(ctx, event.DeviceID, event.ObservedAt); err != nil {
		return err
	}
	if err := o.store.StoreDNSEvent(ctx, event); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil
		}
		return err
	}

	if o.expectedDNSResolver != "" && event.Resolver != "" && !strings.EqualFold(event.Resolver, o.expectedDNSResolver) {
		_, err := o.raiseAlert(ctx, domain.Alert{
			DeviceID:  event.DeviceID,
			Type:      "dns_bypass",
			Severity:  "high",
			Title:     fmt.Sprintf("Potential DNS bypass detected for %s", event.DeviceID),
			Status:    "open",
			Evidence:  map[string]any{"resolver": event.Resolver, "expected_resolver": o.expectedDNSResolver, "query": event.Query},
			CreatedAt: event.ObservedAt,
		})
		if err != nil {
			return err
		}
	}

	if strings.EqualFold(event.Category, "adult") && !event.Blocked {
		_, err := o.raiseAlert(ctx, domain.Alert{
			DeviceID:  event.DeviceID,
			Type:      "policy_violation",
			Severity:  "high",
			Title:     fmt.Sprintf("Adult domain observed for %s", event.DeviceID),
			Status:    "open",
			Evidence:  map[string]any{"domain": event.Domain, "category": event.Category},
			CreatedAt: event.ObservedAt,
		})
		return err
	}

	return nil
}

func (o *Orchestrator) HandleFlowEvent(ctx context.Context, event domain.FlowEvent) error {
	event.ID = flowEventID(event)
	event.ObservedAt = normalizeTime(event.ObservedAt)
	if err := o.ensureDeviceExists(ctx, event.DeviceID, event.ObservedAt); err != nil {
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
			ID:         uuid.NewString(),
			DeviceID:   event.DeviceID,
			Source:     "flow-collector",
			Kind:       "suspicious_flow",
			Severity:   "medium",
			Summary:    fmt.Sprintf("Observed traffic to port %d", event.DstPort),
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
	return o.store.UpdateDeviceProfile(ctx, id, profileID)
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

func (o *Orchestrator) ensureDeviceExists(ctx context.Context, deviceID string, observedAt time.Time) error {
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
		Hostname:   deviceID,
		DeviceType: "unknown",
		ProfileID:  "guest",
		Managed:    false,
		RiskScore:  25,
		FirstSeen:  observedAt,
		LastSeen:   observedAt,
	})
	return err
}

func defaultProfileForDevice(deviceType string) string {
	switch strings.ToLower(deviceType) {
	case "camera", "iot":
		return "iot"
	default:
		return "guest"
	}
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
