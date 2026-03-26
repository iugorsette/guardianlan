package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
	"github.com/sette/guardian-lan/services/control-plane/internal/messaging"
	"github.com/sette/guardian-lan/services/control-plane/internal/repository"
	"github.com/sette/guardian-lan/services/control-plane/internal/service"
)

func TestListDevices(t *testing.T) {
	store := repository.NewMemoryStore()
	_, _, err := store.UpsertDevice(context.Background(), domain.Device{
		ID:         "device-1",
		Hostname:   "device-1",
		ProfileID:  "guest",
		DeviceType: "unknown",
		FirstSeen:  time.Now().UTC(),
		LastSeen:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}

	server := NewServer(":0", store, service.NewOrchestrator(store, messaging.NoopPublisher{}, "adguardhome"))
	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rec := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestUpdateDeviceProfile(t *testing.T) {
	store := repository.NewMemoryStore()
	_, _, err := store.UpsertDevice(context.Background(), domain.Device{
		ID:         "device-1",
		Hostname:   "device-1",
		ProfileID:  "guest",
		DeviceType: "unknown",
		FirstSeen:  time.Now().UTC(),
		LastSeen:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}

	server := NewServer(":0", store, service.NewOrchestrator(store, messaging.NoopPublisher{}, "adguardhome"))
	body, _ := json.Marshal(domain.ProfileUpdateRequest{ProfileID: "child"})
	req := httptest.NewRequest(http.MethodPost, "/devices/device-1/profile", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	device, err := store.GetDevice(context.Background(), "device-1")
	if err != nil {
		t.Fatalf("get device: %v", err)
	}

	if device.ProfileID != "child" {
		t.Fatalf("expected profile child, got %s", device.ProfileID)
	}
}

func TestListDeviceInsights(t *testing.T) {
	store := repository.NewMemoryStore()
	now := time.Now().UTC()
	_, _, err := store.UpsertDevice(context.Background(), domain.Device{
		ID:         "device-1",
		Hostname:   "cam-1",
		ProfileID:  "iot",
		DeviceType: "camera",
		FirstSeen:  now,
		LastSeen:   now,
	})
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}

	err = store.StoreObservation(context.Background(), domain.Observation{
		ID:       "obs-1",
		DeviceID: "device-1",
		Source:   "discovery-collector",
		Kind:     "device_discovered",
		Severity: "medium",
		Summary:  "Camera observed",
		EvidenceRef: map[string]any{
			"candidate_snapshot_urls": []string{"http://192.168.1.21:80/snapshot.jpg"},
			"preview_supported":       true,
		},
		ObservedAt: now,
	})
	if err != nil {
		t.Fatalf("seed observation: %v", err)
	}

	server := NewServer(":0", store, service.NewOrchestrator(store, messaging.NoopPublisher{}, "adguardhome"))
	req := httptest.NewRequest(http.MethodGet, "/devices/device-1/insights", nil)
	rec := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var insights []domain.DeviceInsight
	if err := json.Unmarshal(rec.Body.Bytes(), &insights); err != nil {
		t.Fatalf("decode insights: %v", err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}

	if insights[0].Kind != "device_discovered" {
		t.Fatalf("expected insight kind device_discovered, got %s", insights[0].Kind)
	}
}
