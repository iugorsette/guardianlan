package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
)

const (
	guardLANManagedStart = "! guardlan:managed:start"
	guardLANManagedEnd   = "! guardlan:managed:end"
)

var safeBrowsingCategories = []string{"malware", "phishing", "security"}

type AdGuardSyncer interface {
	SyncDevice(context.Context, domain.Device, domain.Profile, domain.DNSPolicy, []domain.Device, map[string]domain.Profile) error
	SyncAll(context.Context, []domain.Device, map[string]domain.Profile) error
	ValidateHost(context.Context, string, string) (AdGuardHostCheck, error)
}

type NoopAdGuardSyncer struct{}

func (NoopAdGuardSyncer) SyncDevice(context.Context, domain.Device, domain.Profile, domain.DNSPolicy, []domain.Device, map[string]domain.Profile) error {
	return nil
}

func (NoopAdGuardSyncer) SyncAll(context.Context, []domain.Device, map[string]domain.Profile) error {
	return nil
}

func (NoopAdGuardSyncer) ValidateHost(context.Context, string, string) (AdGuardHostCheck, error) {
	return AdGuardHostCheck{}, nil
}

type AdGuardClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

type adGuardClientsResponse struct {
	Clients []adGuardClientConfig `json:"clients"`
}

type adGuardClientConfig struct {
	Name                string            `json:"name"`
	IDs                 []string          `json:"ids"`
	UseGlobalSettings   bool              `json:"use_global_settings"`
	FilteringEnabled    bool              `json:"filtering_enabled"`
	ParentalEnabled     bool              `json:"parental_enabled"`
	SafebrowsingEnabled bool              `json:"safebrowsing_enabled"`
	SafeSearch          adGuardSafeSearch `json:"safe_search"`
	Tags                []string          `json:"tags"`
	IgnoreQueryLog      bool              `json:"ignore_querylog,omitempty"`
	IgnoreStatistics    bool              `json:"ignore_statistics,omitempty"`
}

type adGuardSafeSearch struct {
	Enabled    bool `json:"enabled"`
	Google     bool `json:"google"`
	Bing       bool `json:"bing"`
	DuckDuckGo bool `json:"duckduckgo"`
	Ecosia     bool `json:"ecosia"`
	Pixabay    bool `json:"pixabay"`
	Yandex     bool `json:"yandex"`
	Youtube    bool `json:"youtube"`
}

type adGuardClientUpdate struct {
	Name string              `json:"name"`
	Data adGuardClientConfig `json:"data"`
}

type adGuardClientDelete struct {
	Name string `json:"name"`
}

type adGuardFilteringStatus struct {
	UserRules []string `json:"user_rules"`
}

type adGuardSetRulesRequest struct {
	Rules []string `json:"rules"`
}

type AdGuardHostCheck struct {
	Reason      string   `json:"reason"`
	CanFiltered bool     `json:"can_filtered"`
	Rule        string   `json:"rule"`
	Rules       []string `json:"rules"`
}

func NewAdGuardClient(baseURL string, username string, password string) *AdGuardClient {
	return &AdGuardClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

func (c *AdGuardClient) SyncDevice(ctx context.Context, device domain.Device, profile domain.Profile, policy domain.DNSPolicy, devices []domain.Device, profiles map[string]domain.Profile) error {
	if err := c.upsertClient(ctx, device, profile, policy); err != nil {
		return err
	}

	return c.SyncAll(ctx, devices, profiles)
}

func (c *AdGuardClient) SyncAll(ctx context.Context, devices []domain.Device, profiles map[string]domain.Profile) error {
	filteringStatus, err := c.getFilteringStatus(ctx)
	if err != nil {
		return err
	}

	manualRules := stripManagedRules(filteringStatus.UserRules)
	managedRules := buildManagedRules(devices, profiles)
	nextRules := append(append([]string{}, manualRules...), managedRules...)

	return c.setRules(ctx, nextRules)
}

func (c *AdGuardClient) ValidateHost(ctx context.Context, host string, client string) (AdGuardHostCheck, error) {
	values := url.Values{}
	values.Set("name", host)
	if client != "" {
		values.Set("client", client)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/filtering/check_host?"+values.Encode(), nil)
	if err != nil {
		return AdGuardHostCheck{}, fmt.Errorf("create adguard check host request: %w", err)
	}

	var result AdGuardHostCheck
	if err := c.doJSON(request, nil, &result); err != nil {
		return AdGuardHostCheck{}, err
	}

	return result, nil
}

func (c *AdGuardClient) upsertClient(ctx context.Context, device domain.Device, profile domain.Profile, policy domain.DNSPolicy) error {
	ids := adGuardClientIDs(device)
	if len(ids) == 0 {
		return nil
	}

	clients, err := c.getClients(ctx)
	if err != nil {
		return err
	}

	desired := adGuardClientConfig{
		Name:                adGuardClientName(device),
		IDs:                 ids,
		UseGlobalSettings:   false,
		FilteringEnabled:    true,
		ParentalEnabled:     domain.ValueMatches(policy.BlockedCategories, "adult"),
		SafebrowsingEnabled: hasAnyCategory(policy.BlockedCategories, safeBrowsingCategories),
		SafeSearch:          toAdGuardSafeSearch(policy.SafeSearch),
	}

	existing, found := findManagedClient(clients.Clients, device.ID, ids...)
	if !found {
		request, err := newJSONRequest(ctx, http.MethodPost, c.baseURL+"/clients/add", desired)
		if err != nil {
			return err
		}
		return c.doJSON(request, desired, nil)
	}

	updateRequest := adGuardClientUpdate{
		Name: existing.Name,
		Data: desired,
	}
	request, err := newJSONRequest(ctx, http.MethodPost, c.baseURL+"/clients/update", updateRequest)
	if err != nil {
		return err
	}
	return c.doJSON(request, updateRequest, nil)
}

func (c *AdGuardClient) getClients(ctx context.Context) (adGuardClientsResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/clients", nil)
	if err != nil {
		return adGuardClientsResponse{}, fmt.Errorf("create adguard clients request: %w", err)
	}

	var response adGuardClientsResponse
	if err := c.doJSON(request, nil, &response); err != nil {
		return adGuardClientsResponse{}, err
	}

	return response, nil
}

func (c *AdGuardClient) getFilteringStatus(ctx context.Context) (adGuardFilteringStatus, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/filtering/status", nil)
	if err != nil {
		return adGuardFilteringStatus{}, fmt.Errorf("create adguard filtering status request: %w", err)
	}

	var response adGuardFilteringStatus
	if err := c.doJSON(request, nil, &response); err != nil {
		return adGuardFilteringStatus{}, err
	}

	return response, nil
}

func (c *AdGuardClient) setRules(ctx context.Context, rules []string) error {
	requestBody := adGuardSetRulesRequest{Rules: rules}
	request, err := newJSONRequest(ctx, http.MethodPost, c.baseURL+"/filtering/set_rules", requestBody)
	if err != nil {
		return err
	}

	return c.doJSON(request, requestBody, nil)
}

func (c *AdGuardClient) doJSON(request *http.Request, body any, out any) error {
	request.SetBasicAuth(c.username, c.password)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call adguard api %s: %w", request.URL.Path, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("adguard api %s failed with status %d: %s", request.URL.Path, response.StatusCode, strings.TrimSpace(string(payload)))
	}

	if out == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("decode adguard api %s response: %w", request.URL.Path, err)
	}

	return nil
}

func newJSONRequest(ctx context.Context, method string, endpoint string, body any) (*http.Request, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal adguard request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create adguard request: %w", err)
	}

	return request, nil
}

func toAdGuardSafeSearch(enabled bool) adGuardSafeSearch {
	return adGuardSafeSearch{
		Enabled:    enabled,
		Google:     enabled,
		Bing:       enabled,
		DuckDuckGo: enabled,
		Ecosia:     enabled,
		Pixabay:    enabled,
		Yandex:     enabled,
		Youtube:    enabled,
	}
}

func adGuardClientName(device domain.Device) string {
	base := strings.TrimSpace(device.DisplayName)
	if base == "" {
		base = strings.TrimSpace(device.Hostname)
	}
	if base == "" {
		base = device.ID
	}

	return fmt.Sprintf("%s [%s]", base, device.ID)
}

func adGuardClientIDs(device domain.Device) []string {
	ids := make([]string, 0, len(device.IPs)+1)
	for _, ip := range device.IPs {
		ip = strings.TrimSpace(ip)
		if ip != "" && !slices.Contains(ids, ip) {
			ids = append(ids, ip)
		}
	}

	if device.MAC != "" && !slices.Contains(ids, device.MAC) {
		ids = append(ids, device.MAC)
	}

	return ids
}

func findManagedClientByIDs(clients []adGuardClientConfig, ids []string) (adGuardClientConfig, bool) {
	for _, client := range clients {
		for _, id := range ids {
			if id != "" && slices.Contains(client.IDs, id) {
				return client, true
			}
		}
	}

	return adGuardClientConfig{}, false
}

func findManagedClient(clients []adGuardClientConfig, deviceID string, ids ...string) (adGuardClientConfig, bool) {
	if client, found := findManagedClientByIDs(clients, ids); found {
		return client, true
	}

	for _, client := range clients {
		if strings.HasSuffix(strings.TrimSpace(client.Name), " ["+deviceID+"]") {
			return client, true
		}
	}

	return adGuardClientConfig{}, false
}

func buildManagedRules(devices []domain.Device, profiles map[string]domain.Profile) []string {
	rules := []string{guardLANManagedStart}

	for _, device := range devices {
		profile := profiles[device.ProfileID]
		policy := domain.MergeDNSPolicies(profile.DNSPolicy, device.DNSPolicyOverride)
		clientSelector := adGuardClientRuleSelector(device)
		if clientSelector == "" {
			continue
		}

		if len(policy.AllowedDomains) > 0 {
			rules = append(rules, fmt.Sprintf("||*^$client=%s,denyallow=%s", clientSelector, strings.Join(policy.AllowedDomains, "|")))
		}

		for _, domainName := range policy.BlockedDomains {
			rules = append(rules, fmt.Sprintf("||%s^$client=%s", domainName, clientSelector))
		}
	}

	rules = append(rules, guardLANManagedEnd)
	return normalizeManagedRules(rules)
}

func stripManagedRules(rules []string) []string {
	if len(rules) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(rules))
	skipping := false
	for _, rule := range rules {
		switch strings.TrimSpace(rule) {
		case guardLANManagedStart:
			skipping = true
			continue
		case guardLANManagedEnd:
			skipping = false
			continue
		}

		if !skipping {
			filtered = append(filtered, rule)
		}
	}

	return filtered
}

func adGuardClientRuleSelector(device domain.Device) string {
	parts := make([]string, 0, len(device.IPs))
	for _, ip := range device.IPs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		parts = append(parts, escapeClientSelectorValue(ip))
	}

	return strings.Join(parts, "|")
}

func escapeClientSelectorValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `,`, `\,`, `|`, `\|`, `'`, `\'`, `"`, `\"`)
	return replacer.Replace(value)
}

func normalizeManagedRules(rules []string) []string {
	result := make([]string, 0, len(rules))
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule != "" {
			result = append(result, rule)
		}
	}
	return result
}

func hasAnyCategory(values []string, categories []string) bool {
	for _, category := range categories {
		if domain.ValueMatches(values, category) {
			return true
		}
	}

	return false
}
