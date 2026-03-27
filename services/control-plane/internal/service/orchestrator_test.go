package service

import (
	"context"
	"testing"
	"time"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
	"github.com/sette/guardian-lan/services/control-plane/internal/integration"
	"github.com/sette/guardian-lan/services/control-plane/internal/repository"
)

type stubAlertPublisher struct {
	alerts []domain.Alert
}

func (s *stubAlertPublisher) PublishAlert(_ context.Context, alert domain.Alert) error {
	s.alerts = append(s.alerts, alert)
	return nil
}

func TestHandleDNSEventRaisesBypassAlert(t *testing.T) {
	store := repository.NewMemoryStore()
	publisher := &stubAlertPublisher{}
	orchestrator := NewOrchestrator(store, publisher, "adguardhome", integration.NoopAdGuardSyncer{})

	_, _, err := store.UpsertDevice(context.Background(), domain.Device{
		ID:         "device-kid-tablet",
		Hostname:   "kid-tablet",
		ProfileID:  "child",
		DeviceType: "tablet",
		FirstSeen:  time.Now().UTC(),
		LastSeen:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}

	err = orchestrator.HandleDNSEvent(context.Background(), domain.DNSEvent{
		DeviceID:   "device-kid-tablet",
		ClientIP:   "192.168.1.25",
		ClientName: "kid-tablet",
		Query:      "adult-example.test",
		Domain:     "adult-example.test",
		Category:   "adult",
		Resolver:   "8.8.8.8",
		Blocked:    false,
		ObservedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("handle dns event: %v", err)
	}

	alerts, err := store.ListAlerts(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}

	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}

	if len(publisher.alerts) != 2 {
		t.Fatalf("expected 2 published alerts, got %d", len(publisher.alerts))
	}
}

func TestHandleDNSEventMatchesDeviceByClientIPAndDomainPolicy(t *testing.T) {
	store := repository.NewMemoryStore()
	publisher := &stubAlertPublisher{}
	orchestrator := NewOrchestrator(store, publisher, "adguardhome", integration.NoopAdGuardSyncer{})

	_, _, err := store.UpsertDevice(context.Background(), domain.Device{
		ID:         "device-kid-tablet",
		Hostname:   "kid-tablet",
		IPs:        []string{"192.168.1.25"},
		ProfileID:  "child",
		DeviceType: "tablet",
		DNSPolicyOverride: domain.DNSPolicy{
			BlockedDomains: []string{"xvideos.com"},
		},
		FirstSeen: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}

	err = orchestrator.HandleDNSEvent(context.Background(), domain.DNSEvent{
		ClientIP:   "192.168.1.25",
		ClientName: "kid-tablet",
		Query:      "www.xvideos.com",
		Domain:     "www.xvideos.com",
		Category:   "adult",
		Resolver:   "adguardhome",
		Blocked:    false,
		ObservedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("handle dns event: %v", err)
	}

	events, err := store.ListDNSEvents(context.Background(), 10)
	if err != nil {
		t.Fatalf("list dns events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 dns event, got %d", len(events))
	}

	if events[0].DeviceID != "device-kid-tablet" {
		t.Fatalf("expected dns event linked to device-kid-tablet, got %s", events[0].DeviceID)
	}

	alerts, err := store.ListAlerts(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}

	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}
}

func TestHandleFlowEventCreatesObservation(t *testing.T) {
	store := repository.NewMemoryStore()
	orchestrator := NewOrchestrator(store, &stubAlertPublisher{}, "adguardhome", integration.NoopAdGuardSyncer{})

	err := orchestrator.HandleFlowEvent(context.Background(), domain.FlowEvent{
		DeviceID:   "device-baby-cam",
		SrcIP:      "192.168.1.20",
		DstIP:      "203.0.113.44",
		DstPort:    554,
		Protocol:   "tcp",
		BytesIn:    100,
		BytesOut:   200,
		ObservedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("handle flow event: %v", err)
	}

	alerts, err := store.ListAlerts(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
}
