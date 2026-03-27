package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          int
	DatabaseURL   string
	JWTSecret     string
	EncryptionKey string
	CORSOrigins   string
	LogLevel      string
}

func Load() *Config {
	return &Config{
		Port:          getEnvInt("PANELOX_PORT", 8080),
		DatabaseURL:   getEnv("PANELOX_DATABASE_URL", "postgres://panelox:devpassword@localhost:5432/panelox?sslmode=disable"),
		JWTSecret:     getEnv("PANELOX_JWT_SECRET", "dev-secret-change-in-production"),
		EncryptionKey: getEnv("PANELOX_ENCRYPTION_KEY", "dev-encryption-key-change-me"),
		CORSOrigins:   getEnv("PANELOX_CORS_ORIGINS", "http://localhost:5173"),
		LogLevel:      getEnv("PANELOX_LOG_LEVEL", "debug"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
