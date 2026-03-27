package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
)

func TestBuildManagedRules(t *testing.T) {
	devices := []domain.Device{
		{
			ID:         "device-kid",
			IPs:        []string{"192.168.1.25"},
			ProfileID:  "child",
			DeviceType: "tablet",
			DNSPolicyOverride: domain.DNSPolicy{
				BlockedDomains: []string{"xvideos.com"},
				AllowedDomains: []string{"escola.local", "googleclassroom.com"},
			},
		},
	}
	profiles := map[string]domain.Profile{
		"child": {
			ID:        "child",
			DNSPolicy: domain.DNSPolicy{BlockedCategories: []string{"adult"}},
		},
	}

	rules := buildManagedRules(devices, profiles)
	joined := strings.Join(rules, "\n")
	if !strings.Contains(joined, "||*^$client=192.168.1.25,denyallow=escola.local|googleclassroom.com") {
		t.Fatalf("expected whitelist catch-all rule, got %v", rules)
	}
	if !strings.Contains(joined, "||xvideos.com^$client=192.168.1.25") {
		t.Fatalf("expected blocked domain rule, got %v", rules)
	}
}

func TestStripManagedRules(t *testing.T) {
	rules := []string{
		"||manual.example^",
		guardLANManagedStart,
		"||xvideos.com^$client=192.168.1.25",
		guardLANManagedEnd,
		"@@||allowed.example^",
	}

	filtered := stripManagedRules(rules)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 manual rules, got %v", filtered)
	}
}

func TestSyncAllPreservesManualRules(t *testing.T) {
	var postedRules adGuardSetRulesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/control/filtering/status":
			_ = json.NewEncoder(w).Encode(adGuardFilteringStatus{
				UserRules: []string{
					"||manual.example^",
					guardLANManagedStart,
					"||old.example^$client=192.168.1.25",
					guardLANManagedEnd,
				},
			})
		case "/control/filtering/set_rules":
			if err := json.NewDecoder(r.Body).Decode(&postedRules); err != nil {
				t.Fatalf("decode posted rules: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewAdGuardClient(server.URL+"/control", "admin", "secret")
	devices := []domain.Device{
		{
			ID:         "device-kid",
			IPs:        []string{"192.168.1.25"},
			ProfileID:  "guest",
			DeviceType: "tablet",
			DNSPolicyOverride: domain.DNSPolicy{
				BlockedDomains: []string{"xvideos.com"},
			},
		},
	}
	profiles := map[string]domain.Profile{
		"guest": {ID: "guest"},
	}

	if err := client.SyncAll(context.Background(), devices, profiles); err != nil {
		t.Fatalf("sync all: %v", err)
	}

	joined := strings.Join(postedRules.Rules, "\n")
	if !strings.Contains(joined, "||manual.example^") {
		t.Fatalf("expected manual rule preserved, got %v", postedRules.Rules)
	}
	if !strings.Contains(joined, "||xvideos.com^$client=192.168.1.25") {
		t.Fatalf("expected managed rule present, got %v", postedRules.Rules)
	}
	if strings.Contains(joined, "old.example") {
		t.Fatalf("expected old managed rules to be replaced, got %v", postedRules.Rules)
	}
}
