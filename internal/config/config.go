package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURI            string
	GrpcApiKey             string
	GrpcPort               string
	GrpcEnableReflection   bool
	GqlPort                string
	GqlEnableInstropection bool
	AllowedOrigins         []string
	GoogleClientID         string
	JwtSecret              string
	JwtTtl                 time.Duration
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
	cfg.GrpcApiKey = getEnv("GRPC_API_KEY", "")
	cfg.GrpcPort = ":" + getEnv("GRPC_PORT", "50051")
	cfg.GrpcEnableReflection = getEnv("GRPC_ENABLE_REFLECTION", "false") == "true"
	cfg.GqlPort = ":" + getEnv("GQL_PORT", "8080")
	cfg.GqlEnableInstropection = getEnv("GQL_ENABLE_INTROSPECTION", "false") == "true"
	cfg.AllowedOrigins = getEnvArray("ALLOWED_ORIGINS", "http://localhost,http://127.0.0.1")
	cfg.GoogleClientID = getEnv("GOOGLE_CLIENT_ID", "")
	cfg.JwtSecret = getEnv("JWT_SECRET", "")

	jwtTtl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_TTL has invalid duration: %v", err)
	}
	cfg.JwtTtl = jwtTtl

	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("DATABASE_URI is required")
	}

	if cfg.GrpcApiKey == "" {
		return nil, fmt.Errorf("GRPC_API_KEY is required")
	}

	if cfg.GoogleClientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}

	if cfg.JwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	} else if len(cfg.JwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be equal or longer than 32")
	}

	return cfg, nil
}
