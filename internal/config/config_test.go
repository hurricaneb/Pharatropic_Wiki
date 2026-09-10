package config

import (
	"os"
	"testing"
)

// clearEnv unsets every variable LoadConfig reads, and returns a func that
// restores whatever was there before, so tests can't leak into each other or
// pick up values from the environment the test process itself runs in.
func clearEnv(t *testing.T) {
	t.Helper()
	vars := []string{
		"PORT", "DB_DRIVER", "DB_PATH",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
		"DATABASE_URL", "API_KEY", "JWT_SECRET",
	}
	for _, v := range vars {
		old, existed := os.LookupEnv(v)
		os.Unsetenv(v)
		t.Cleanup(func() {
			if existed {
				os.Setenv(v, old)
			} else {
				os.Unsetenv(v)
			}
		})
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	clearEnv(t)

	cfg := LoadConfig()

	if cfg.Port != "8080" {
		t.Errorf("Expected default Port '8080', got %q", cfg.Port)
	}
	if cfg.DBDriver != "sqlite" {
		t.Errorf("Expected default DBDriver 'sqlite', got %q", cfg.DBDriver)
	}
	if cfg.DBPath != "wiki.db" {
		t.Errorf("Expected default DBPath 'wiki.db', got %q", cfg.DBPath)
	}
	if cfg.PostgresPort != "5432" {
		t.Errorf("Expected default PostgresPort '5432', got %q", cfg.PostgresPort)
	}
	for name, got := range map[string]string{
		"PostgresHost":     cfg.PostgresHost,
		"PostgresUser":     cfg.PostgresUser,
		"PostgresPassword": cfg.PostgresPassword,
		"PostgresDB":       cfg.PostgresDB,
		"DatabaseURL":      cfg.DatabaseURL,
		"MasterAPIKey":     cfg.MasterAPIKey,
		"JWTSecret":        cfg.JWTSecret,
	} {
		if got != "" {
			t.Errorf("Expected %s to default to empty, got %q", name, got)
		}
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	clearEnv(t)

	os.Setenv("PORT", "9090")
	os.Setenv("DB_DRIVER", "postgres")
	os.Setenv("DB_PATH", "/data/custom.db")
	os.Setenv("POSTGRES_HOST", "db.internal")
	os.Setenv("POSTGRES_PORT", "6543")
	os.Setenv("POSTGRES_USER", "wiki")
	os.Setenv("POSTGRES_PASSWORD", "s3cret")
	os.Setenv("POSTGRES_DB", "wikidb")
	os.Setenv("DATABASE_URL", "postgres://wiki:s3cret@db.internal:6543/wikidb")
	os.Setenv("API_KEY", "master-key-value")
	os.Setenv("JWT_SECRET", "jwt-secret-value")

	cfg := LoadConfig()

	want := Config{
		Port:             "9090",
		DBDriver:         "postgres",
		DBPath:           "/data/custom.db",
		PostgresHost:     "db.internal",
		PostgresPort:     "6543",
		PostgresUser:     "wiki",
		PostgresPassword: "s3cret",
		PostgresDB:       "wikidb",
		DatabaseURL:      "postgres://wiki:s3cret@db.internal:6543/wikidb",
		MasterAPIKey:     "master-key-value",
		JWTSecret:        "jwt-secret-value",
	}

	if *cfg != want {
		t.Fatalf("Expected %+v, got %+v", want, *cfg)
	}
}
