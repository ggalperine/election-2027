package config

import "os"

// Config holds env-driven settings shared by all services.
type Config struct {
	DatabaseURL string
	AMQPURL     string
	HTTPAddr    string
}

func Load() Config {
	return Config{
		DatabaseURL: env("DATABASE_URL", "postgres://election:election@localhost:5432/election2027?sslmode=disable"),
		AMQPURL:     env("AMQP_URL", "amqp://election:election@localhost:5672/"),
		HTTPAddr:    env("HTTP_ADDR", ":8080"),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
