package config

import (
	"net/url"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_DRIVER", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("APP_ENCRYPTION_KEY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Driver != "postgres" {
		t.Fatalf("Database.Driver = %q, want postgres", cfg.Database.Driver)
	}
	if cfg.JWT.TTL != 24*time.Hour {
		t.Fatalf("JWT.TTL = %s, want 24h", cfg.JWT.TTL)
	}
	if len(cfg.EncryptionKey) != 32 {
		t.Fatalf("EncryptionKey length = %d, want 32", len(cfg.EncryptionKey))
	}
}

func TestLoadDerivesPostgresURLFromParts(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("JWT_SECRET", "development-only-secret-change-before-release")
	t.Setenv("POSTGRES_USER", "can_i_do")
	t.Setenv("POSTGRES_PASSWORD", "s3cret")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_DB", "app")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := "postgres://can_i_do:s3cret@postgres:5432/app?sslmode=disable"
	if cfg.Database.URL != want {
		t.Fatalf("Database.URL = %q, want %q", cfg.Database.URL, want)
	}
}

func TestLoadEscapesPostgresPassword(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("JWT_SECRET", "development-only-secret-change-before-release")
	t.Setenv("POSTGRES_USER", "can_i_do")
	t.Setenv("POSTGRES_PASSWORD", "p@ss:w/rd?")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	parsed, err := url.Parse(cfg.Database.URL)
	if err != nil {
		t.Fatalf("parsing derived URL %q: %v", cfg.Database.URL, err)
	}
	password, _ := parsed.User.Password()
	if password != "p@ss:w/rd?" {
		t.Fatalf("password round-trip = %q, want %q", password, "p@ss:w/rd?")
	}
	if parsed.Host != "localhost:5432" {
		t.Fatalf("Host = %q, want localhost:5432", parsed.Host)
	}
}

func TestLoadPrefersExplicitDatabaseURL(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("JWT_SECRET", "development-only-secret-change-before-release")
	t.Setenv("DATABASE_URL", "postgres://explicit:pw@db:5432/other?sslmode=require")
	t.Setenv("POSTGRES_USER", "ignored")
	t.Setenv("POSTGRES_PASSWORD", "ignored")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := "postgres://explicit:pw@db:5432/other?sslmode=require"
	if cfg.Database.URL != want {
		t.Fatalf("Database.URL = %q, want %q", cfg.Database.URL, want)
	}
}

func TestLoadDerivesMySQLDSNFromParts(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_DRIVER", "mysql")
	t.Setenv("JWT_SECRET", "development-only-secret-change-before-release")
	t.Setenv("MYSQL_USER", "can_i_do")
	t.Setenv("MYSQL_PASSWORD", "s3cret")
	t.Setenv("MYSQL_HOST", "mysql")
	t.Setenv("MYSQL_DATABASE", "app")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := "can_i_do:s3cret@tcp(mysql:3306)/app?parseTime=true&charset=utf8mb4"
	if cfg.Database.URL != want {
		t.Fatalf("Database.URL = %q, want %q", cfg.Database.URL, want)
	}
}

func TestLoadRejectsInvalidDriver(t *testing.T) {
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("JWT_SECRET", "development-only-secret-change-before-release")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid driver error")
	}
}

func TestLoadRequiresProductionSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want JWT secret error")
	}
}
