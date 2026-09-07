package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURI    string
	GRPCPort       string
	GQLPort        string
	AllowedOrigins []string
}

func getEnv(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvArray(key string, fallback string) []string {
	v := getEnv(key, fallback)
	arrV := strings.Split(v, ",")
	return arrV
}

func Load() (*Config, error) {
	// Ignore error - .env not exists on 'prod'
	_ = godotenv.Load()

	cfg := &Config{}
	cfg.DatabaseURI = getEnv("DATABASE_URI", "")
	cfg.GRPCPort = getEnv("GRPC_PORT", ":50051")
	cfg.GQLPort = getEnv("GQL_PORT", ":8080")
	cfg.AllowedOrigins = getEnvArray("ALLOWED_ORIGINS", "localhost,127.0.0.1")

	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("DATABASE_URI is required")
	}

	return cfg, nil
}
