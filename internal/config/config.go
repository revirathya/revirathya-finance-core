package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURI          string
	GRPCPort             string
	GQLPort              string
	AllowedOrigins       []string
	EnableGRPCReflection bool
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
	for i, s := range arrV {
		arrV[i] = strings.TrimSpace(s)
	}
	return arrV
}

func Load() (*Config, error) {
	// Ignore error - .env not exists on 'prod'
	_ = godotenv.Load()

	cfg := &Config{}
	cfg.DatabaseURI = getEnv("DATABASE_URI", "")
	cfg.GRPCPort = ":" + getEnv("GRPC_PORT", "50051")
	cfg.GQLPort = ":" + getEnv("GQL_PORT", "8080")
	cfg.AllowedOrigins = getEnvArray("ALLOWED_ORIGINS", "http://localhost,http://127.0.0.1")
	cfg.EnableGRPCReflection = getEnv("ENABLE_GRPC_REFLECTION", "false") == "true"

	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("DATABASE_URI is required")
	}

	return cfg, nil
}
