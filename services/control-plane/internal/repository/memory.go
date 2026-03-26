package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
)

type MemoryStore struct {
	mu           sync.RWMutex
	devices      map[string]domain.Device
	profiles     map[string]domain.Profile
	dnsEvents    []domain.DNSEvent
	flowEvents   []domain.FlowEvent
	observations []domain.Observation
	alerts       map[string]domain.Alert
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		devices:  map[string]domain.Device{},
		profiles: defaultProfiles(),
		alerts:   map[string]domain.Alert{},
	}
}

func (s *MemoryStore) UpsertDevice(_ context.Context, device domain.Device) (domain.Device, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.devices[device.ID]
	if exists {
		current := s.devices[device.ID]
		device.FirstSeen = current.FirstSeen
		if current.DisplayName != "" && device.DisplayName == "" {
			device.DisplayName = current.DisplayName
		}
		if len(device.DNSPolicyOverride.BlockedCategories) == 0 && len(device.DNSPolicyOverride.BlockedDomains) == 0 && len(device.DNSPolicyOverride.AllowedDomains) == 0 && !device.DNSPolicyOverride.SafeSearch {
			device.DNSPolicyOverride = current.DNSPolicyOverride
		}
	}
	s.devices[device.ID] = device

	return device, !exists, nil
}

func (s *MemoryStore) GetDevice(_ context.Context, id string) (domain.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, ok := s.devices[id]
	if !ok {
		return domain.Device{}, pgx.ErrNoRows
	}

	return device, nil
}

func (s *MemoryStore) ListDevices(_ context.Context) ([]domain.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]domain.Device, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, device)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].LastSeen.After(devices[j].LastSeen)
	})

	return devices, nil
}

func (s *MemoryStore) GetProfile(_ context.Context, id string) (domain.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, ok := s.profiles[id]
	if !ok {
		return domain.Profile{}, pgx.ErrNoRows
	}

	return profile, nil
}

func (s *MemoryStore) ListProfiles(_ context.Context) ([]domain.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profiles := make([]domain.Profile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		profiles = append(profiles, profile)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

func (s *MemoryStore) UpdateDeviceProfile(_ context.Context, id string, profileID string) (domain.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[id]
	if !ok {
		return domain.Device{}, pgx.ErrNoRows
	}

	device.ProfileID = profileID
	s.devices[id] = device

	return device, nil
}

func (s *MemoryStore) UpdateDeviceName(_ context.Context, id string, displayName string) (domain.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[id]
	if !ok {
		return domain.Device{}, pgx.ErrNoRows
	}

	device.DisplayName = displayName
	s.devices[id] = device

	return device, nil
}

func (s *MemoryStore) UpdateDeviceDNSPolicy(_ context.Context, id string, policy domain.DNSPolicy) (domain.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[id]
	if !ok {
		return domain.Device{}, pgx.ErrNoRows
	}

	device.DNSPolicyOverride = domain.NormalizeDNSPolicy(policy)
	s.devices[id] = device

	return device, nil
}

func (s *MemoryStore) StoreDNSEvent(_ context.Context, event domain.DNSEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dnsEvents = append(s.dnsEvents, event)
	sort.Slice(s.dnsEvents, func(i, j int) bool {
		return s.dnsEvents[i].ObservedAt.After(s.dnsEvents[j].ObservedAt)
	})
	return nil
}

func defaultProfiles() map[string]domain.Profile {
	return map[string]domain.Profile{
		"adult": {
			ID:          "adult",
			Name:        "Adulto",
			Kind:        "adult",
			Schedule:    map[string]any{},
			DNSPolicy:   domain.DNSPolicy{},
			AlertPolicy: domain.AlertPolicy{},
		},
		"child": {
			ID:       "child",
			Name:     "Crianca",
			Kind:     "child",
			Schedule: map[string]any{"curfew": "21:00-07:00"},
			DNSPolicy: domain.DNSPolicy{
				SafeSearch:        true,
				BlockedCategories: []string{"adult", "gambling"},
			},
			AlertPolicy: domain.AlertPolicy{NotifyOnBypass: true},
		},
		"iot": {
			ID:       "iot",
			Name:     "IoT",
			Kind:     "iot",
			Schedule: map[string]any{},
			DNSPolicy: domain.DNSPolicy{
				BlockedCategories: []string{"newly_registered_domains"},
			},
			AlertPolicy: domain.AlertPolicy{NotifyOnExposure: true},
		},
		"guest": {
			ID:       "guest",
			Name:     "Visitante",
			Kind:     "guest",
			Schedule: map[string]any{},
			DNSPolicy: domain.DNSPolicy{
				BlockedCategories: []string{"adult"},
			},
			AlertPolicy: domain.AlertPolicy{NotifyOnNewDevice: true},
		},
	}
}

func (s *MemoryStore) ListDNSEvents(_ context.Context, limit int) ([]domain.DNSEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return limitedCopy(s.dnsEvents, limit), nil
}

func (s *MemoryStore) StoreFlowEvent(_ context.Context, event domain.FlowEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flowEvents = append(s.flowEvents, event)
	sort.Slice(s.flowEvents, func(i, j int) bool {
		return s.flowEvents[i].ObservedAt.After(s.flowEvents[j].ObservedAt)
	})
	return nil
}

func (s *MemoryStore) ListFlowEvents(_ context.Context, limit int) ([]domain.FlowEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return limitedCopy(s.flowEvents, limit), nil
}

func (s *MemoryStore) StoreObservation(_ context.Context, observation domain.Observation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.observations = append(s.observations, observation)
	sort.Slice(s.observations, func(i, j int) bool {
		return s.observations[i].ObservedAt.After(s.observations[j].ObservedAt)
	})
	return nil
}

func (s *MemoryStore) ListDeviceObservations(_ context.Context, deviceID string, limit int) ([]domain.Observation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var observations []domain.Observation
	for _, observation := range s.observations {
		if observation.DeviceID == deviceID {
			observations = append(observations, observation)
		}
	}

	return limitedCopy(observations, limit), nil
}

func (s *MemoryStore) CreateAlert(_ context.Context, alert domain.Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.alerts[alert.ID] = alert
	return nil
}

func (s *MemoryStore) ListAlerts(_ context.Context, limit int, status string) ([]domain.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]domain.Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		if status == "" || alert.Status == status {
			alerts = append(alerts, alert)
		}
	}

	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].CreatedAt.After(alerts[j].CreatedAt)
	})

	return limitedCopy(alerts, limit), nil
}

func (s *MemoryStore) AckAlert(_ context.Context, id string) (domain.Alert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, ok := s.alerts[id]
	if !ok {
		return domain.Alert{}, pgx.ErrNoRows
	}

	now := time.Now().UTC()
	alert.Status = "acknowledged"
	alert.Acknowledged = &now
	s.alerts[id] = alert

	return alert, nil
}

func limitedCopy[T any](items []T, limit int) []T {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	copied := make([]T, limit)
	copy(copied, items[:limit])
	return copied
}

var ErrNotFound = errors.New("not found")
