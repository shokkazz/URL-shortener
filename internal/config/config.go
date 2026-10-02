package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Dbc            DatabaseConfig
	Port           string
	ShortURLLength int
	Charset        string
	CleanupSeconds int
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLmode  string
}

func LoadConfig() (*Config, error) {
	length := os.Getenv("SHORT_LENGTH")
	value, err := strconv.Atoi(length)
	if err != nil || value <= 0 {
		return nil, errors.New("SHORT_LENGTH must be a positive integer")
	}
	cleanup, err := strconv.Atoi(os.Getenv("CLEANUP_SECONDS"))
	if err != nil || cleanup <= 0 {
		return nil, errors.New("error during CLEANUP_SECONDS parsing")
	}
	cfg := &Config{
		CleanupSeconds: cleanup,
		Port:           os.Getenv("APP_PORT"),
		ShortURLLength: value,
		Charset:        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		Dbc: DatabaseConfig{
			Name:     os.Getenv("DB_NAME"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			SSLmode:  os.Getenv("DB_SSLMODE"),
		},
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

//func getEnvOrDefault(key, def string) string {
//	value := os.Getenv(key)
//	if value == "" {
//		return def
//	}
//	return value
//}

func (dbc *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbc.User, dbc.Password,
		dbc.Host, dbc.Port, dbc.Name, dbc.SSLmode)
}

func (cfg *Config) validate() error {
	if cfg.Port == "" {
		return errors.New("APP_PORT is required")
	}

	if cfg.Dbc.Host == "" {
		return errors.New("DB_HOST is required")
	}

	if cfg.Dbc.Port == "" {
		return errors.New("DB_PORT is required")
	}

	if cfg.Dbc.User == "" {
		return errors.New("DB_USER is required")
	}

	if cfg.Dbc.Password == "" {
		return errors.New("DB_PASSWORD is required")
	}

	if cfg.Dbc.Name == "" {
		return errors.New("DB_NAME is required")
	}

	if cfg.Dbc.SSLmode == "" {
		return errors.New("DB_SSLMODE is required")
	}

	return nil
}
