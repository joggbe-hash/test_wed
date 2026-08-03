package main

import (
	"strings"
	"testing"
)

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
