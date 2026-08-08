// Package storage provides one storage boundary for uploaded and generated
// assets. Cloudflare R2 is S3-compatible, so the AWS SDK can be used without
// exposing provider-specific details to the rest of the application.
package storage

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	settingEnabled         = "storage_enabled"
	settingEndpoint        = "storage_endpoint"
	settingBucket          = "storage_bucket"
	settingAccessKeyID     = "storage_access_key_id"
	settingSecretAccessKey = "storage_secret_access_key"
	settingPublicURL       = "storage_public_url"
	settingRegion          = "storage_region"
	settingForcePathStyle  = "storage_force_path_style"
)

// R2Config is the provider configuration needed by the S3-compatible API.
type R2Config struct {
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

// Settings is safe to return to an owner console. Secret values are never
// included; HasCredentials only reports whether both credentials are present.
type Settings struct {
	Enabled        bool   `json:"enabled"`
	Endpoint       string `json:"endpoint"`
	Bucket         string `json:"bucket"`
	PublicURL      string `json:"public_url"`
	Region         string `json:"region"`
	ForcePathStyle bool   `json:"force_path_style"`
	HasCredentials bool   `json:"has_credentials"`
	Configured     bool   `json:"configured"`
	UsingR2        bool   `json:"using_r2"`
}

// Service stores objects in R2 when it is configured and transparently falls
// back to the existing local upload directory otherwise.
type Service struct {
	db         *sql.DB
	driver     string
	encryption []byte
	localDir   string
	envR2      R2Config
	max        int64
}

func New(db *sql.DB, driver, encryptionKey, localDir string, maxUploadBytes int64, defaults R2Config) *Service {
	if defaults.Region == "" {
		defaults.Region = "auto"
	}
	defaults.Endpoint = strings.TrimRight(defaults.Endpoint, "/")
	defaults.PublicURL = strings.TrimRight(defaults.PublicURL, "/")
	return &Service{
		db:         db,
		driver:     driver,
		encryption: []byte(encryptionKey),
		localDir:   localDir,
		envR2:      defaults,
		max:        maxUploadBytes,
	}
}

// MaxUploadBytes is used by HTTP handlers to apply the same limit regardless
// of whether the active backend is local disk or R2.
func (s *Service) MaxUploadBytes() int64 { return s.max }

// R2Settings reads owner-managed settings, falling back to environment
// defaults. A missing settings table is treated as an empty override so local
// development remains compatible with older databases during startup.
func (s *Service) R2Settings(ctx context.Context) (Settings, error) {
	cfg, err := s.r2Config(ctx)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		Enabled:        cfg.Enabled,
		Endpoint:       cfg.Endpoint,
		Bucket:         cfg.Bucket,
		PublicURL:      cfg.PublicURL,
		Region:         cfg.Region,
		ForcePathStyle: cfg.ForcePathStyle,
		HasCredentials: cfg.AccessKeyID != "" && cfg.SecretAccessKey != "",
		Configured:     cfg.configured(),
		UsingR2:        cfg.configured(),
	}, nil
}

// SaveR2Settings persists non-secret values and encrypts credentials with the
// application's 32-byte encryption key. Empty credentials keep the current
// encrypted values, which lets the UI safely update a bucket without asking
// the owner to re-enter secrets.
type Update struct {
	Enabled          bool
	Endpoint         string
	Bucket           string
	AccessKeyID      string
	SecretAccessKey  string
	PublicURL        string
	Region           string
	ForcePathStyle   bool
	ClearCredentials bool
}

func (s *Service) SaveR2Settings(ctx context.Context, in Update) error {
	if len(s.encryption) != 32 && (in.AccessKeyID != "" || in.SecretAccessKey != "" || in.ClearCredentials) {
		return ErrEncryptionNotConfigured
	}
	values := map[string]string{
		settingEnabled:        strconv.FormatBool(in.Enabled),
		settingEndpoint:       strings.TrimRight(strings.TrimSpace(in.Endpoint), "/"),
		settingBucket:         strings.TrimSpace(in.Bucket),
		settingPublicURL:      strings.TrimRight(strings.TrimSpace(in.PublicURL), "/"),
		settingRegion:         strings.TrimSpace(in.Region),
		settingForcePathStyle: strconv.FormatBool(in.ForcePathStyle),
	}
	if values[settingRegion] == "" {
		values[settingRegion] = "auto"
	}
	if in.ClearCredentials {
		values[settingAccessKeyID] = ""
		values[settingSecretAccessKey] = ""
	} else {
		if in.AccessKeyID != "" {
			v, err := encrypt(s.encryption, strings.TrimSpace(in.AccessKeyID))
			if err != nil {
				return err
			}
			values[settingAccessKeyID] = v
		}
		if in.SecretAccessKey != "" {
			v, err := encrypt(s.encryption, strings.TrimSpace(in.SecretAccessKey))
			if err != nil {
				return err
			}
			values[settingSecretAccessKey] = v
		}
	}
	for key, value := range values {
		q := "INSERT INTO app_settings (`key`,value) VALUES (?,?) ON DUPLICATE KEY UPDATE value=VALUES(value)"
		if s.driver == "postgres" {
			q = "INSERT INTO app_settings (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value"
		}
		if _, err := s.db.ExecContext(ctx, q, key, value); err != nil {
			return err
		}
	}
	return nil
}

var ErrEncryptionNotConfigured = errors.New("APP_ENCRYPTION_KEY must be 32 bytes")
var ErrR2NotConfigured = errors.New("R2 storage is not configured")

func (s *Service) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	key, err := cleanKey(key)
	if err != nil {
		return err
	}
	cfg, err := s.r2Config(ctx)
	if err != nil {
		return err
	}
	useR2, err := cfg.backend()
	if err != nil {
		return err
	}
	if useR2 {
		client, err := s3Client(ctx, cfg)
		if err != nil {
			return err
		}
		input := &s3.PutObjectInput{Bucket: aws.String(cfg.Bucket), Key: aws.String(key), Body: body}
		if contentType != "" {
			input.ContentType = aws.String(contentType)
		}
		if size >= 0 {
			input.ContentLength = aws.Int64(size)
		}
		_, err = client.PutObject(ctx, input)
		return err
	}
	return s.putLocal(key, body)
}

func (s *Service) Delete(ctx context.Context, key string) error {
	key, err := cleanKey(key)
	if err != nil {
		return err
	}
	cfg, err := s.r2Config(ctx)
	if err != nil {
		return err
	}
	useR2, err := cfg.backend()
	if err != nil {
		return err
	}
	if useR2 {
		client, err := s3Client(ctx, cfg)
		if err != nil {
			return err
		}
		_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(cfg.Bucket), Key: aws.String(key)})
		return err
	}
	err = os.Remove(s.localPath(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Open is used for private/local downloads. R2 callers should use URL, which
// returns a short-lived presigned URL and avoids proxying large objects.
func (s *Service) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	key, err := cleanKey(key)
	if err != nil {
		return nil, 0, err
	}
	cfg, err := s.r2Config(ctx)
	if err != nil {
		return nil, 0, err
	}
	useR2, err := cfg.backend()
	if err != nil {
		return nil, 0, err
	}
	if useR2 {
		client, err := s3Client(ctx, cfg)
		if err != nil {
			return nil, 0, err
		}
		out, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(cfg.Bucket), Key: aws.String(key)})
		if err != nil {
			return nil, 0, err
		}
		var size int64
		if out.ContentLength != nil {
			size = *out.ContentLength
		}
		return out.Body, size, nil
	}
	file, err := os.Open(s.localPath(key))
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}

// URL returns a public object URL when configured, or a presigned R2 URL.
// Local storage returns an empty string because it must be served through an
// authenticated API download route.
func (s *Service) URL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	key, err := cleanKey(key)
	if err != nil {
		return "", err
	}
	cfg, err := s.r2Config(ctx)
	if err != nil {
		return "", err
	}
	useR2, err := cfg.backend()
	if err != nil {
		return "", err
	}
	if !useR2 {
		return "", nil
	}
	if cfg.PublicURL != "" {
		segments := strings.Split(key, "/")
		for i, segment := range segments {
			segments[i] = url.PathEscape(segment)
		}
		return strings.TrimRight(cfg.PublicURL, "/") + "/" + strings.Join(segments, "/"), nil
	}
	client, err := s3Client(ctx, cfg)
	if err != nil {
		return "", err
	}
	presigner := s3.NewPresignClient(client)
	out, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(cfg.Bucket), Key: aws.String(key)}, func(opts *s3.PresignOptions) {
		if expiry > 0 {
			opts.Expires = expiry
		}
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func (s *Service) Test(ctx context.Context) error {
	cfg, err := s.r2Config(ctx)
	if err != nil {
		return err
	}
	if !cfg.configured() {
		return ErrR2NotConfigured
	}
	client, err := s3Client(ctx, cfg)
	if err != nil {
		return err
	}
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.Bucket)})
	return err
}

func (s *Service) r2Config(ctx context.Context) (R2Config, error) {
	cfg := s.envR2
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	if s.db == nil {
		return cfg, nil
	}
	query := "SELECT `key`,value FROM app_settings"
	if s.driver == "postgres" {
		query = "SELECT key,value FROM app_settings"
	}
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		// A pre-R2 database may not have app_settings yet; callers can still
		// use environment-backed local storage until migrations complete.
		return cfg, nil
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		switch key {
		case settingEnabled:
			cfg.Enabled, _ = strconv.ParseBool(value)
		case settingEndpoint:
			cfg.Endpoint = strings.TrimRight(value, "/")
		case settingBucket:
			cfg.Bucket = value
		case settingPublicURL:
			cfg.PublicURL = strings.TrimRight(value, "/")
		case settingRegion:
			cfg.Region = value
		case settingForcePathStyle:
			cfg.ForcePathStyle, _ = strconv.ParseBool(value)
		case settingAccessKeyID:
			if value != "" {
				cfg.AccessKeyID, _ = decrypt(s.encryption, value)
			}
		case settingSecretAccessKey:
			if value != "" {
				cfg.SecretAccessKey, _ = decrypt(s.encryption, value)
			}
		}
	}
	if cfg.Endpoint == "" && cfg.AccountID != "" {
		cfg.Endpoint = "https://" + cfg.AccountID + ".r2.cloudflarestorage.com"
	}
	return cfg, nil
}

func (c R2Config) configured() bool {
	return c.Enabled && c.Endpoint != "" && c.Bucket != "" && c.AccessKeyID != "" && c.SecretAccessKey != ""
}

func (c R2Config) backend() (bool, error) {
	if !c.Enabled {
		return false, nil
	}
	if !c.configured() {
		return false, ErrR2NotConfigured
	}
	return true, nil
}

func (s *Service) putLocal(key string, body io.Reader) error {
	if err := os.MkdirAll(s.localDir, 0750); err != nil {
		return err
	}
	filename := s.localPath(key)
	if err := os.MkdirAll(path.Dir(filename), 0750); err != nil {
		return err
	}
	out, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(filename)
		return copyErr
	}
	return closeErr
}

func (s *Service) localPath(key string) string { return filepathJoin(s.localDir, key) }

// filepathJoin is kept tiny so all key sanitization stays in cleanKey while
// the actual path uses the platform's filesystem semantics.
func filepathJoin(dir, key string) string {
	parts := strings.Split(key, "/")
	return pathToOS(dir, parts...)
}

func pathToOS(dir string, parts ...string) string {
	all := append([]string{dir}, parts...)
	return strings.Join(all, string(os.PathSeparator))
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(strings.ReplaceAll(key, "\\", "/"))
	key = strings.TrimPrefix(path.Clean("/"+key), "/")
	if key == "" || key == "." || strings.HasPrefix(key, "../") {
		return "", fmt.Errorf("invalid object key")
	}
	return key, nil
}

func s3Client(ctx context.Context, cfg R2Config) (*s3.Client, error) {
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	loaded, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}
	loaded.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{URL: cfg.Endpoint, SigningRegion: cfg.Region, HostnameImmutable: true}, nil
	})
	return s3.NewFromConfig(loaded, func(options *s3.Options) {
		options.UsePathStyle = cfg.ForcePathStyle
	}), nil
}

func encrypt(key []byte, plain string) (string, error) {
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func decrypt(key []byte, encoded string) (string, error) {
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return "", err
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid encrypted value")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	return string(plain), err
}

// HTTPErrorCode maps storage failures to stable API error codes without
// leaking provider credentials or request details to clients.
func HTTPErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrR2NotConfigured):
		return "storage_not_configured"
	case errors.Is(err, ErrEncryptionNotConfigured):
		return "encryption_not_configured"
	default:
		return "storage_error"
	}
}
