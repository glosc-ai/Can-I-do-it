package config

import (
	"cmp"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment   string
	HTTPAddr      string
	LogLevel      string
	CORSOrigins   []string
	Database      Database
	Redis         Redis
	JWT           JWT
	SSO           SSO
	Session       Session
	Storage       Storage
	AI            AI
	EncryptionKey string
}

// developmentEncryptionKey keeps locally persisted settings decryptable across
// restarts when APP_ENCRYPTION_KEY has not been provided. It must never be used
// in production; production deployments should always set a random key.
const developmentEncryptionKey = "development-only-key-32-bytes!!!"

type SSO struct{ DiscoveryURL, ClientID, ClientSecret, RedirectURI string }
type Session struct {
	CookieName string
	TTL        time.Duration
	Secure     bool
}
type Storage struct {
	Directory      string
	MaxUploadBytes int64
	R2             R2
}

// R2 contains the S3-compatible credentials used by Cloudflare R2. The
// values are intentionally kept in the server configuration only as defaults;
// owner-managed values are persisted in app_settings and take precedence.
type R2 struct {
	Enabled         bool
	AccountID       string
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	PublicURL       string
	Region          string
	ForcePathStyle  bool
}
type AI struct{ Endpoint, Model, APIKey string }

type Database struct {
	Driver          string
	URL             string
	AutoMigrate     bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

type JWT struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

func Load() (Config, error) {
	environment := cmp.Or(os.Getenv("APP_ENV"), "development")
	driver := cmp.Or(os.Getenv("DB_DRIVER"), "postgres")
	if driver != "postgres" && driver != "mysql" {
		return Config{}, fmt.Errorf("DB_DRIVER must be postgres or mysql, got %q", driver)
	}

	autoMigrate, err := envBool("AUTO_MIGRATE", true)
	if err != nil {
		return Config{}, err
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" && environment != "production" {
		jwtSecret = "development-only-secret-change-before-release"
	}
	if len(jwtSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}

	jwtTTL, err := envDuration("JWT_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	if jwtTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be positive")
	}
	connMaxLifetime, err := envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}

	redisDB, err := envInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	maxOpen, err := envInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	maxIdle, err := envInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
	}
	if maxOpen < 1 || maxIdle < 0 {
		return Config{}, fmt.Errorf("database connection limits must be non-negative and DB_MAX_OPEN_CONNS must be positive")
	}
	sessionTTL, err := envDuration("SESSION_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	if sessionTTL <= 0 {
		return Config{}, fmt.Errorf("SESSION_TTL must be positive")
	}

	encryptionKey := os.Getenv("APP_ENCRYPTION_KEY")
	if encryptionKey == "" && environment != "production" {
		encryptionKey = developmentEncryptionKey
	}

	r2Enabled, err := envBool("R2_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	forcePathStyle, err := envBool("R2_FORCE_PATH_STYLE", false)
	if err != nil {
		return Config{}, err
	}
	r2Endpoint := os.Getenv("R2_ENDPOINT")
	accountID := os.Getenv("R2_ACCOUNT_ID")
	if r2Endpoint == "" && accountID != "" {
		r2Endpoint = "https://" + accountID + ".r2.cloudflarestorage.com"
	}

	return Config{
		Environment: environment,
		HTTPAddr:    cmp.Or(os.Getenv("HTTP_ADDR"), ":8080"),
		LogLevel:    cmp.Or(os.Getenv("LOG_LEVEL"), "info"),
		CORSOrigins: splitList(cmp.Or(os.Getenv("CORS_ORIGINS"), "http://localhost:5173")),
		Database: Database{
			Driver:          driver,
			URL:             databaseURL(driver),
			AutoMigrate:     autoMigrate,
			MaxOpenConns:    maxOpen,
			MaxIdleConns:    maxIdle,
			ConnMaxLifetime: connMaxLifetime,
		},
		Redis: Redis{
			Addr:     cmp.Or(os.Getenv("REDIS_ADDR"), "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
		JWT: JWT{
			Secret: jwtSecret,
			Issuer: cmp.Or(os.Getenv("JWT_ISSUER"), "go-vue-starter"),
			TTL:    jwtTTL,
		},
		SSO:     SSO{DiscoveryURL: cmp.Or(os.Getenv("SSO_DISCOVERY_URL"), "https://sso.gloscai.com/api/.well-known/openid-configuration"), ClientID: os.Getenv("SSO_CLIENT_ID"), ClientSecret: os.Getenv("SSO_CLIENT_SECRET"), RedirectURI: cmp.Or(os.Getenv("SSO_REDIRECT_URI"), "http://localhost:5173/api/v1/auth/callback")},
		Session: Session{CookieName: cmp.Or(os.Getenv("SESSION_COOKIE_NAME"), "can_i_session"), TTL: sessionTTL, Secure: environment == "production"},
		Storage: Storage{
			Directory:      cmp.Or(os.Getenv("UPLOAD_DIR"), "/tmp/can-i-do-it/uploads"),
			MaxUploadBytes: 20 * 1024 * 1024,
			R2: R2{
				Enabled:         r2Enabled,
				AccountID:       accountID,
				Endpoint:        r2Endpoint,
				Bucket:          os.Getenv("R2_BUCKET"),
				AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
				SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
				PublicURL:       strings.TrimRight(os.Getenv("R2_PUBLIC_URL"), "/"),
				Region:          cmp.Or(os.Getenv("R2_REGION"), "auto"),
				ForcePathStyle:  forcePathStyle,
			},
		},
		AI:            AI{Endpoint: os.Getenv("AI_ENDPOINT"), Model: cmp.Or(os.Getenv("AI_MODEL"), "gpt-4o-mini"), APIKey: os.Getenv("AI_API_KEY")},
		EncryptionKey: encryptionKey,
	}, nil
}

// databaseURL resolves the DSN used to reach the database. An explicit
// DATABASE_URL always wins so existing deployments keep working, but when it is
// empty the DSN is assembled from the same discrete variables that provision the
// server (POSTGRES_* / MYSQL_*). Deriving it removes the failure mode where
// rotating POSTGRES_PASSWORD reprovisions the database yet leaves a stale
// password baked into a hand-written DATABASE_URL, which surfaces at startup as
// "password authentication failed (SQLSTATE 28P01)".
func databaseURL(driver string) string {
	if explicit := strings.TrimSpace(os.Getenv("DATABASE_URL")); explicit != "" {
		return explicit
	}
	if driver == "mysql" {
		return mysqlDSN()
	}
	return postgresDSN()
}

func postgresDSN() string {
	user := cmp.Or(os.Getenv("POSTGRES_USER"), "app")
	password := cmp.Or(os.Getenv("POSTGRES_PASSWORD"), "app")
	host := cmp.Or(os.Getenv("POSTGRES_HOST"), "localhost")
	port := cmp.Or(os.Getenv("POSTGRES_PORT"), "5432")
	name := cmp.Or(os.Getenv("POSTGRES_DB"), "app")
	sslMode := cmp.Or(os.Getenv("POSTGRES_SSLMODE"), "disable")

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + name,
		RawQuery: url.Values{"sslmode": {sslMode}}.Encode(),
	}
	return dsn.String()
}

func mysqlDSN() string {
	user := cmp.Or(os.Getenv("MYSQL_USER"), "app")
	password := cmp.Or(os.Getenv("MYSQL_PASSWORD"), "app")
	host := cmp.Or(os.Getenv("MYSQL_HOST"), "localhost")
	port := cmp.Or(os.Getenv("MYSQL_PORT"), "3306")
	name := cmp.Or(os.Getenv("MYSQL_DATABASE"), "app")

	// The go-sql-driver DSN is not a URL, so credentials are interpolated
	// verbatim rather than percent-encoded.
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&charset=utf8mb4",
		user, password, net.JoinHostPort(host, port), name)
}

func envBool(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parsing %s: %w", name, err)
	}
	return parsed, nil
}

func envInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", name, err)
	}
	return parsed, nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", name, err)
	}
	return parsed, nil
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
