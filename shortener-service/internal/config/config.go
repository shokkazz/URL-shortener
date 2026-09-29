package config

import "os"

type Config struct {
	dbc          DatabaseConfig
	port         string
	shortenerURL string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLmode  string
}

func LoadConfig() *Config {
	return &Config{
		port:         getEnvOrDefault("APP_PORT", "8080"),
		shortenerURL: getEnvOrDefault("SHORTENER_URL", "hz"),
		dbc: DatabaseConfig{
			Name:     getEnvOrDefault("DB_NAME", "postgres"),
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
			SSLmode:  getEnvOrDefault("DB_SSLMODE", "disabled"),
		},
	}
}

func getEnvOrDefault(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}
