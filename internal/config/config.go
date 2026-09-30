package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Dbc            DatabaseConfig
	Port           string
	ShortURLLength int
	Charset        string
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
	length := getEnvOrDefault("SHORT_LENGTH", "5")
	value, err := strconv.Atoi(length)
	if err != nil {
		log.Println("Invalid ShortURL parameter type")
		return nil
	}
	return &Config{
		Port:           getEnvOrDefault("APP_PORT", "8080"),
		ShortURLLength: value,
		Dbc: DatabaseConfig{
			Name:     getEnvOrDefault("DB_NAME", "postgres"),
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
			SSLmode:  getEnvOrDefault("DB_SSLMODE", "disabled"),
		},
		Charset: func() string {
			const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
			const digits = "0123456789"

			return letters + digits
		}(),
	}
}

func getEnvOrDefault(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}

func (dbc *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbc.User, dbc.Password,
		dbc.Host, dbc.Port, dbc.Name, dbc.SSLmode)
}
