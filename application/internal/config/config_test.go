package config

import (
	"os"
	"testing"
)

func TestLoadConfig_JSONSecrets(t *testing.T) {
	jsonPayload := `{"admin_api_key":"test-admin-key","jwt_signing_key":"jwt-secret-123","db_host":"10.0.0.1"}`
	os.Setenv("APP_CONFIG_SECRETS", jsonPayload)
	defer os.Unsetenv("APP_CONFIG_SECRETS")

	cfg := LoadConfig()
	if cfg.AdminAPIKey != "test-admin-key" {
		t.Errorf("expected AdminAPIKey test-admin-key, got %s", cfg.AdminAPIKey)
	}
	if cfg.JWTSigningKey != "jwt-secret-123" {
		t.Errorf("expected JWTSigningKey jwt-secret-123, got %s", cfg.JWTSigningKey)
	}
	if cfg.DatabaseHost != "10.0.0.1" {
		t.Errorf("expected DatabaseHost 10.0.0.1, got %s", cfg.DatabaseHost)
	}
}

func TestLoadConfig_Fallback(t *testing.T) {
	os.Unsetenv("APP_CONFIG_SECRETS")
	os.Unsetenv("TODO_ADMIN_KEY")

	cfg := LoadConfig()
	if cfg.AdminAPIKey != "dev-test-admin-secret-key-3b9e4a1" {
		t.Errorf("expected fallback dev key, got %s", cfg.AdminAPIKey)
	}
}
