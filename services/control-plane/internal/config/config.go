package config

import "os"

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	NATSURL             string
	ExpectedDNSResolver string
	AdGuardEnabled      bool
	AdGuardURL          string
	AdGuardUsername     string
	AdGuardPassword     string
}

func Load() Config {
	return Config{
		HTTPAddr:            getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:         getenv("DATABASE_URL", "postgres://postgres:guardian_lan_local_2026@postgres:5432/guardian_lan?sslmode=disable"),
		NATSURL:             getenv("NATS_URL", "nats://nats:4222"),
		ExpectedDNSResolver: getenv("EXPECTED_DNS_RESOLVER", "adguardhome"),
		AdGuardEnabled:      getenvBool("ADGUARD_ENABLED", false),
		AdGuardURL:          getenv("ADGUARD_URL", "http://adguardhome:3000/control"),
		AdGuardUsername:     getenv("ADGUARD_USERNAME", ""),
		AdGuardPassword:     getenv("ADGUARD_PASSWORD", ""),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}
