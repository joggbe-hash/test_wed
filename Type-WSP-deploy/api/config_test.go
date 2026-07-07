package main

import "testing"

func TestLoadConfigRequiresSecretKeyFromEnvironment(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret-key-with-enough-length")

	cfg := LoadConfig()
	if cfg.SecretKey != "test-secret-key-with-enough-length" {
		t.Fatalf("SecretKey = %q", cfg.SecretKey)
	}
}
