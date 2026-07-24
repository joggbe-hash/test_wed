package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv string

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresSSLMode  string
	DBSystem         string

	RedisURL string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioSecure    bool
}

func LoadConfig() (*Config, error) {
	environment := strings.ToLower(strings.TrimSpace(envOr("APP_ENV", "production")))
	postgresHost, err := environmentValue(environment, "POSTGRES_HOST", "localhost")
	if err != nil {
		return nil, err
	}
	postgresPort, err := environmentValue(environment, "POSTGRES_PORT", "5432")
	if err != nil {
		return nil, err
	}
	postgresUser, err := requiredEnv("POSTGRES_USER")
	if err != nil {
		return nil, err
	}
	postgresPassword, err := requiredEnv("POSTGRES_PASSWORD")
	if err != nil {
		return nil, err
	}
	postgresSSLMode, err := environmentValue(environment, "POSTGRES_SSLMODE", "disable")
	if err != nil {
		return nil, err
	}
	if !validPostgresSSLMode(postgresSSLMode) {
		return nil, fmt.Errorf("POSTGRES_SSLMODE has unsupported value %q", postgresSSLMode)
	}
	dbSystem, err := environmentValue(environment, "POSTGRES_DB_SYSTEM", "system_db")
	if err != nil {
		return nil, err
	}
	redisURL, err := environmentValue(environment, "REDIS_URL", "redis://localhost:6379/0")
	if err != nil {
		return nil, err
	}
	minioEndpoint, err := environmentValue(environment, "MINIO_ENDPOINT", "localhost:9000")
	if err != nil {
		return nil, err
	}
	minioAccessKey, err := requiredEnv("MINIO_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	minioSecretKey, err := requiredEnv("MINIO_SECRET_KEY")
	if err != nil {
		return nil, err
	}
	minioBucket, err := environmentValue(environment, "MINIO_BUCKET", "uploads")
	if err != nil {
		return nil, err
	}
	minioSecure, err := environmentBool(environment, "MINIO_SECURE", false)
	if err != nil {
		return nil, err
	}

	return &Config{
		AppEnv:           environment,
		PostgresHost:     postgresHost,
		PostgresPort:     postgresPort,
		PostgresUser:     postgresUser,
		PostgresPassword: postgresPassword,
		PostgresSSLMode:  postgresSSLMode,
		DBSystem:         dbSystem,
		RedisURL:         redisURL,
		MinioEndpoint:    minioEndpoint,
		MinioAccessKey:   minioAccessKey,
		MinioSecretKey:   minioSecretKey,
		MinioBucket:      minioBucket,
		MinioSecure:      minioSecure,
	}, nil
}

func environmentValue(environment, key, developmentFallback string) (string, error) {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value, nil
	}
	if environment == "development" || environment == "test" {
		return developmentFallback, nil
	}
	return "", fmt.Errorf("%s is required when APP_ENV=%s", key, environment)
}

func requiredEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func environmentBool(environment, key string, developmentFallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		if environment == "development" || environment == "test" {
			return developmentFallback, nil
		}
		return false, fmt.Errorf("%s is required when APP_ENV=%s", key, environment)
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return value, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func validPostgresSSLMode(value string) bool {
	switch value {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}
