package config

import (
	"encoding/json"
	"log"
	"os"
)

// AppConfig represents parsed configuration loaded at runtime.
type AppConfig struct {
	Environment   string
	DebugMode     bool
	AdminAPIKey   string
	DatabaseHost  string
	DatabaseUser  string
	DatabaseName  string
	JWTSigningKey string
	SentryDSN     string
}

// LoadConfig demonstrates realistic configuration loading logic in modern microservices:
// Teams commonly bundle all production secrets (database, JWT, admin keys, SaaS tokens)
// into a single Secret Manager JSON payload (APP_CONFIG_SECRETS) to save cost and avoid
// managing dozens of individual secrets.
func LoadConfig() *AppConfig {
	cfg := &AppConfig{
		Environment:  os.Getenv("ENVIRONMENT"),
		DebugMode:    os.Getenv("DEBUG") == "true",
		DatabaseHost: os.Getenv("DB_HOST"),
	}

	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// 1. Try parsing JSON secret payload from Secret Manager (APP_CONFIG_SECRETS)
	if rawSecrets := os.Getenv("APP_CONFIG_SECRETS"); rawSecrets != "" {
		var secretMap map[string]string
		if err := json.Unmarshal([]byte(rawSecrets), &secretMap); err == nil {
			if key, ok := secretMap["admin_api_key"]; ok && key != "" {
				cfg.AdminAPIKey = key
			}
			if host, ok := secretMap["db_host"]; ok && host != "" {
				cfg.DatabaseHost = host
			}
			if user, ok := secretMap["db_user"]; ok && user != "" {
				cfg.DatabaseUser = user
			}
			if name, ok := secretMap["db_name"]; ok && name != "" {
				cfg.DatabaseName = name
			}
			if jwt, ok := secretMap["jwt_signing_key"]; ok && jwt != "" {
				cfg.JWTSigningKey = jwt
			}
			if dsn, ok := secretMap["sentry_dsn"]; ok && dsn != "" {
				cfg.SentryDSN = dsn
			}
		} else {
			log.Printf("[WARN] Failed to parse APP_CONFIG_SECRETS JSON: %v", err)
		}
	}

	// 2. Fallback to standard environment variable
	if cfg.AdminAPIKey == "" {
		cfg.AdminAPIKey = os.Getenv("TODO_ADMIN_KEY")
	}

	// Security Hardening (A04/A07): Fail-closed design.
	// Never embed fallback credentials in compiled code.
	if cfg.AdminAPIKey == "" {
		log.Println("[WARN] No ADMIN_API_KEY detected. Administrative actions will remain disabled.")
	}

	return cfg
}
