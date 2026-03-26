package domain

import "strings"

func MergeDNSPolicies(base DNSPolicy, override DNSPolicy) DNSPolicy {
	result := base
	result.SafeSearch = base.SafeSearch || override.SafeSearch

	if len(override.BlockedCategories) > 0 {
		result.BlockedCategories = append([]string(nil), override.BlockedCategories...)
	}
	if len(override.BlockedDomains) > 0 {
		result.BlockedDomains = append([]string(nil), override.BlockedDomains...)
	}
	if len(override.AllowedDomains) > 0 {
		result.AllowedDomains = append([]string(nil), override.AllowedDomains...)
	}

	return normalizeDNSPolicy(result)
}

func NormalizeDNSPolicy(policy DNSPolicy) DNSPolicy {
	return normalizeDNSPolicy(policy)
}

func normalizeDNSPolicy(policy DNSPolicy) DNSPolicy {
	policy.BlockedCategories = normalizedUnique(policy.BlockedCategories)
	policy.BlockedDomains = normalizedUnique(policy.BlockedDomains)
	policy.AllowedDomains = normalizedUnique(policy.AllowedDomains)
	return policy
}

func DomainMatches(patterns []string, domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return false
	}

	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if domain == pattern || strings.HasSuffix(domain, "."+pattern) {
			return true
		}
	}

	return false
}

func ValueMatches(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}

	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}

	return false
}

func normalizedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}
