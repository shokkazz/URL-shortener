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
	length := os.Getenv("SHORT_LENGTH")
	value, err := strconv.Atoi(length)
	if err != nil {
		log.Println("Invalid ShortURL parameter type")
		return nil
	}
	return &Config{
		Port:           os.Getenv("APP_PORT"),
		ShortURLLength: value,
		Dbc: DatabaseConfig{
			Name:     os.Getenv("DB_NAME"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			SSLmode:  os.Getenv("DB_SSLMODE"),
		},
		Charset: func() string {
			const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
			const digits = "0123456789"

			return letters + digits
		}(),
	}
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
