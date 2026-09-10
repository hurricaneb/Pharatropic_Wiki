package database

import (
	"fmt"
	"os"
	"testing"
	"time"

	"wiki/internal/config"
	"wiki/internal/models"
)

func tempDBPath(t *testing.T) string {
	t.Helper()
	path := fmt.Sprintf("test_db_%d_%d.db", time.Now().UnixNano(), os.Getpid())
	t.Cleanup(func() { os.Remove(path) })
	return path
}

func TestInitDB_SeedsAdminAndWelcomePageOnFreshDB(t *testing.T) {
	cfg := &config.Config{DBDriver: "sqlite", DBPath: tempDBPath(t)}
	db, err := InitDB(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount != 1 {
		t.Fatalf("Expected exactly 1 seeded user, got %d", userCount)
	}

	var admin models.User
	db.Where("username = ?", "admin").First(&admin)
	if admin.Role != "admin" || !admin.MustChangePassword {
		t.Fatalf("Expected seeded admin with role=admin and MustChangePassword=true, got %+v", admin)
	}

	var pageCount int64
	db.Model(&models.Page{}).Count(&pageCount)
	if pageCount != 1 {
		t.Fatalf("Expected exactly 1 seeded welcome page, got %d", pageCount)
	}

	var revisionCount int64
	db.Model(&models.Revision{}).Count(&revisionCount)
	if revisionCount != 1 {
		t.Fatalf("Expected the seeded page to have 1 initial revision, got %d", revisionCount)
	}
}

func TestInitDB_DoesNotReseedExistingDatabase(t *testing.T) {
	path := tempDBPath(t)
	cfg := &config.Config{DBDriver: "sqlite", DBPath: path}

	if _, err := InitDB(cfg); err != nil {
		t.Fatalf("unexpected error on first init: %v", err)
	}

	// Re-opening the same database must not seed a second admin or welcome page
	db2, err := InitDB(cfg)
	if err != nil {
		t.Fatalf("unexpected error on second init: %v", err)
	}

	var userCount, pageCount int64
	db2.Model(&models.User{}).Count(&userCount)
	db2.Model(&models.Page{}).Count(&pageCount)

	if userCount != 1 {
		t.Fatalf("Expected seeding to be skipped, still expected 1 user, got %d", userCount)
	}
	if pageCount != 1 {
		t.Fatalf("Expected seeding to be skipped, still expected 1 page, got %d", pageCount)
	}
}

func TestInitDB_PostgresBranchAttemptsConnection(t *testing.T) {
	// No real Postgres server is available in tests, so this only verifies
	// that the postgres dialector path is taken (DSN built, connection
	// attempted) and that a failed connection surfaces as an error rather
	// than silently falling back to SQLite.
	cfg := &config.Config{
		DBDriver:     "postgres",
		PostgresHost: "127.0.0.1",
		PostgresPort: "1", // nothing listens here
		PostgresUser: "wiki",
		PostgresDB:   "wiki",
	}

	if _, err := InitDB(cfg); err == nil {
		t.Fatal("Expected an error connecting to an unreachable Postgres server")
	}
}

func TestInitDB_PostgresBranchViaDatabaseURL(t *testing.T) {
	cfg := &config.Config{
		DBDriver:    "sqlite", // deliberately mismatched: DatabaseURL alone should still select postgres
		DatabaseURL: "postgres://wiki:wiki@127.0.0.1:1/wiki?sslmode=disable",
	}

	if _, err := InitDB(cfg); err == nil {
		t.Fatal("Expected an error connecting via an unreachable DatabaseURL")
	}
}
