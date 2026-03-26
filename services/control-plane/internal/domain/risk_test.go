package domain

import "testing"

func TestScoreDeviceForCamera(t *testing.T) {
	event := DeviceEvent{
		ID:         "cam-1",
		DeviceType: "camera",
		Managed:    false,
	}

	score := ScoreDevice(event)
	if score < 50 {
		t.Fatalf("expected high risk score for camera, got %d", score)
	}
}

func TestScoreDeviceClampedToOneHundred(t *testing.T) {
	event := DeviceEvent{
		ID:         "cam-1",
		Vendor:     "",
		MAC:        "",
		IPs:        []string{"192.168.1.2", "192.168.1.3"},
		DeviceType: "camera",
		Managed:    false,
	}

	score := ScoreDevice(event)
	if score > 100 {
		t.Fatalf("expected score to be clamped, got %d", score)
	}
}
