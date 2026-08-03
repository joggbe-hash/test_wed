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

	SMTPHost     string
	SMTPPort     int
	SMTPFrom     string
	SMTPUsername string
	SMTPPassword string
	SMTPSecure   bool
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
	smtpHost, err := environmentValue(environment, "SMTP_HOST", "mailpit")
	if err != nil {
		return nil, err
	}
	smtpPort, err := positiveEnvIntForEnvironment(environment, "SMTP_PORT", 1025)
	if err != nil {
		return nil, err
	}
	smtpFrom, err := environmentValue(environment, "SMTP_FROM", "no-reply@type-wsp.local")
	if err != nil {
		return nil, err
	}
	smtpSecure, err := environmentBool(environment, "SMTP_SECURE", false)
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
		SMTPHost:         smtpHost,
		SMTPPort:         smtpPort,
		SMTPFrom:         smtpFrom,
		SMTPUsername:     strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:     os.Getenv("SMTP_PASSWORD"),
		SMTPSecure:       smtpSecure,
	}, nil
}

func positiveEnvIntForEnvironment(environment, key string, developmentFallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		if environment == "development" || environment == "test" {
			return developmentFallback, nil
		}
		return 0, fmt.Errorf("%s is required when APP_ENV=%s", key, environment)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 || value > 65535 {
		return 0, fmt.Errorf("%s must be a valid TCP port", key)
	}
	return value, nil
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
