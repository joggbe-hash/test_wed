package main

import (
	"strings"
	"testing"
)

func setRequiredProductionEnv(t *testing.T) {
	t.Helper()
	values := map[string]string{
		"APP_ENV": "production", "API_LISTEN_ADDR": ":5000",
		"POSTGRES_HOST": "postgres", "POSTGRES_PORT": "5432",
		"POSTGRES_USER": "app", "POSTGRES_PASSWORD": "secret",
		"POSTGRES_SSLMODE": "require", "POSTGRES_DB_SYSTEM": "system_db",
		"POSTGRES_DB_USER": "user_db", "REDIS_URL": "redis://redis:6379/0",
		"MINIO_ENDPOINT": "minio:9000", "MINIO_ACCESS_KEY": "access",
		"MINIO_SECRET_KEY": "secret", "MINIO_BUCKET": "uploads",
		"MINIO_SECURE": "true", "SECRET_KEY": "test-secret-key-with-enough-length",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
}

func TestLoadConfigUsesDevelopmentDefaultsOnlyWhenExplicit(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("POSTGRES_USER", "dev-user")
	t.Setenv("POSTGRES_PASSWORD", "dev-password")
	t.Setenv("MINIO_ACCESS_KEY", "dev-access")
	t.Setenv("MINIO_SECRET_KEY", "dev-secret")
	t.Setenv("SECRET_KEY", "test-secret-key-with-enough-length")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.SecretKey != "test-secret-key-with-enough-length" {
		t.Fatalf("SecretKey = %q", cfg.SecretKey)
	}
	if cfg.PostgresPassword != "dev-password" || cfg.MinioSecretKey != "dev-secret" {
		t.Fatal("credentials were not loaded from the environment")
	}
	if cfg.PostgresSSLMode != "disable" || cfg.MinioSecure {
		t.Fatal("development transport defaults were not applied")
	}
}

func TestLoadConfigRejectsMissingProductionSettings(t *testing.T) {
	setRequiredProductionEnv(t)
	t.Setenv("API_LISTEN_ADDR", "")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "API_LISTEN_ADDR") {
		t.Fatalf("expected missing API_LISTEN_ADDR error, got %v", err)
	}
}

func TestPostgresDSNUsesConfiguredTLSAndEscapesCredentials(t *testing.T) {
	cfg := &Config{
		PostgresHost: "db.example.test", PostgresPort: "5432",
		PostgresUser: "app user", PostgresPassword: "p@ss/word",
		PostgresSSLMode: "verify-full",
	}
	dsn := postgresDSN(cfg, "system db")
	for _, expected := range []string{"app%20user", "p%40ss%2Fword", "system%20db", "sslmode=verify-full"} {
		if !strings.Contains(dsn, expected) {
			t.Fatalf("DSN %q does not contain %q", dsn, expected)
		}
	}
}
