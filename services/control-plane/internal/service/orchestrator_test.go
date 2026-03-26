package service

import (
	"context"
	"testing"
	"time"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
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
	orchestrator := NewOrchestrator(store, publisher, "adguardhome")

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

func TestHandleFlowEventCreatesObservation(t *testing.T) {
	store := repository.NewMemoryStore()
	orchestrator := NewOrchestrator(store, &stubAlertPublisher{}, "adguardhome")

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
