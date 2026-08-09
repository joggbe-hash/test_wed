package main

import (
	"strings"
	"testing"
)

func setRequiredWorkerProductionEnv(t *testing.T) {
	t.Helper()
	values := map[string]string{
		"APP_ENV": "production", "POSTGRES_HOST": "postgres", "POSTGRES_PORT": "5432",
		"POSTGRES_USER": "app", "POSTGRES_PASSWORD": "test-postgres-password-32-bytes",
		"POSTGRES_SSLMODE": "require", "POSTGRES_DB_SYSTEM": "system_db",
		"REDIS_URL":      "redis://:test-redis-password-32-bytes@redis:6379/0",
		"MINIO_ENDPOINT": "minio:9000", "MINIO_ACCESS_KEY": "access",
		"MINIO_SECRET_KEY": "test-minio-secret-key-32-bytes", "MINIO_BUCKET": "uploads",
		"MINIO_SECURE": "true", "SMTP_HOST": "smtp.example.test", "SMTP_PORT": "587",
		"SMTP_FROM": "no-reply@example.test", "SMTP_USERNAME": "mailer",
		"SMTP_PASSWORD": "test-smtp-password-32-bytes", "SMTP_SECURE": "true",
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

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.PostgresPassword != "dev-password" || cfg.MinioSecretKey != "dev-secret" {
		t.Fatal("credentials were not loaded from the environment")
	}
	if cfg.PostgresSSLMode != "disable" || cfg.MinioSecure {
		t.Fatal("development transport defaults were not applied")
	}
	if cfg.SMTPHost != "mailpit" || cfg.SMTPPort != 1025 || cfg.SMTPFrom != "no-reply@type-wsp.local" || cfg.SMTPSecure {
		t.Fatalf("unexpected development SMTP defaults: %#v", cfg)
	}
}

func TestLoadConfigRejectsInvalidSMTPPort(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("POSTGRES_USER", "dev-user")
	t.Setenv("POSTGRES_PASSWORD", "dev-password")
	t.Setenv("MINIO_ACCESS_KEY", "dev-access")
	t.Setenv("MINIO_SECRET_KEY", "dev-secret")
	t.Setenv("SMTP_PORT", "70000")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "SMTP_PORT") {
		t.Fatalf("expected invalid SMTP_PORT error, got %v", err)
	}
}

func TestLoadConfigRejectsMissingProductionSettings(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_HOST", "")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_HOST") {
		t.Fatalf("expected missing POSTGRES_HOST error, got %v", err)
	}
}

func TestLoadConfigRejectsProductionPlaceholderSecrets(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "PostgreSQL password", key: "POSTGRES_PASSWORD", value: "change_me_strong_password"},
		{name: "Redis password", key: "REDIS_URL", value: "redis://:change_me_redis_password@redis:6379/0"},
		{name: "MinIO password", key: "MINIO_SECRET_KEY", value: "change_me_minio_password"},
		{name: "SMTP username", key: "SMTP_USERNAME", value: "replace_me"},
		{name: "SMTP password", key: "SMTP_PASSWORD", value: "replace_me"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setRequiredWorkerProductionEnv(t)
			t.Setenv(test.key, test.value)

			_, err := LoadConfig()
			if err == nil || !strings.Contains(err.Error(), test.key) {
				t.Fatalf("expected %s placeholder rejection, got %v", test.key, err)
			}
		})
	}
}

func TestLoadConfigRejectsShortProductionSecrets(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "PostgreSQL password", key: "POSTGRES_PASSWORD", value: "short"},
		{name: "Redis password", key: "REDIS_URL", value: "redis://:short@redis:6379/0"},
		{name: "MinIO password", key: "MINIO_SECRET_KEY", value: "short"},
		{name: "SMTP password", key: "SMTP_PASSWORD", value: "short"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setRequiredWorkerProductionEnv(t)
			t.Setenv(test.key, test.value)

			_, err := LoadConfig()
			if err == nil || !strings.Contains(err.Error(), test.key) {
				t.Fatalf("expected %s minimum-length rejection, got %v", test.key, err)
			}
		})
	}
}

func TestLoadConfigAllowsProductionSMTPWithoutAuthentication(t *testing.T) {
	setRequiredWorkerProductionEnv(t)
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig without SMTP authentication: %v", err)
	}
	if cfg.SMTPUsername != "" || cfg.SMTPPassword != "" {
		t.Fatal("SMTP authentication was unexpectedly populated")
	}
}

func TestLoadConfigRejectsPlaintextSMTPInProduction(t *testing.T) {
	setRequiredWorkerProductionEnv(t)
	t.Setenv("SMTP_SECURE", "false")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "SMTP_SECURE") {
		t.Fatalf("expected production SMTP TLS requirement, got %v", err)
	}
}

func TestWorkerPostgresDSNUsesConfiguredTLSAndEscapesCredentials(t *testing.T) {
	cfg := &Config{
		PostgresHost: "db.example.test", PostgresPort: "5432",
		PostgresUser: "app user", PostgresPassword: "p@ss/word",
		PostgresSSLMode: "verify-full", DBSystem: "system db",
	}
	dsn := workerPostgresDSN(cfg)
	for _, expected := range []string{"app%20user", "p%40ss%2Fword", "system%20db", "sslmode=verify-full"} {
		if !strings.Contains(dsn, expected) {
			t.Fatalf("DSN %q does not contain %q", dsn, expected)
		}
	}
}
