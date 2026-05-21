package config

import (
	"log"
	"os"
)

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	Environment string
}

func LoadConfig() *Config {
	cfg := &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://cadastreia:dev_password@localhost:5432/cadastreia?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "dev_secret_change_in_prod"),
		Environment: getEnv("ENV", "development"),
	}

	log.Printf("✅ Configuration loaded: ENV=%s PORT=%s", cfg.Environment, cfg.Port)
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
