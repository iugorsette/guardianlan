package config

import "os"

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	NATSURL             string
	ExpectedDNSResolver string
}

func Load() Config {
	return Config{
		HTTPAddr:            getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:         getenv("DATABASE_URL", "postgres://postgres:guardian_lan_local_2026@postgres:5432/guardian_lan?sslmode=disable"),
		NATSURL:             getenv("NATS_URL", "nats://nats:4222"),
		ExpectedDNSResolver: getenv("EXPECTED_DNS_RESOLVER", "adguardhome"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
