package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gloscai/template-go-vue3-docker/server/storage"
	"github.com/gloscai/template-go-vue3-docker/server/users"
)

type Service struct {
	db     *sql.DB
	driver string
	key    []byte
	store  *storage.Service
}

func New(db *sql.DB, driver, key string, stores ...*storage.Service) *Service {
	var store *storage.Service
	if len(stores) > 0 {
		store = stores[0]
	}
	return &Service{db: db, driver: driver, key: []byte(key), store: store}
}
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/settings/ai", s.getAI)
	mux.HandleFunc("PATCH /api/v1/admin/settings/ai", s.setAI)
	mux.HandleFunc("GET /api/v1/admin/settings/storage", s.getStorage)
	mux.HandleFunc("PATCH /api/v1/admin/settings/storage", s.setStorage)
	mux.HandleFunc("POST /api/v1/admin/settings/storage/test", s.testStorage)
	// Keep an explicit R2 alias for integrations that name the provider in
	// their admin API clients.
	mux.HandleFunc("GET /api/v1/admin/settings/r2", s.getStorage)
	mux.HandleFunc("PATCH /api/v1/admin/settings/r2", s.setStorage)
	mux.HandleFunc("POST /api/v1/admin/settings/r2/test", s.testStorage)
}

func (s *Service) getStorage(w http.ResponseWriter, r *http.Request) {
	if !s.owner(r) {
		err(w, 403, "owner_required", "owner permission required")
		return
	}
	if s.store == nil {
		err(w, 503, "storage_not_available", "storage service is not available")
		return
	}
	settings, e := s.store.R2Settings(r.Context())
	if e != nil {
		err(w, 500, "internal_error", "could not load storage settings")
		return
	}
	jsonOut(w, 200, map[string]any{"data": settings})
}

func (s *Service) setStorage(w http.ResponseWriter, r *http.Request) {
	if !s.owner(r) {
		err(w, 403, "owner_required", "owner permission required")
		return
	}
	if s.store == nil {
		err(w, 503, "storage_not_available", "storage service is not available")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var in struct {
		Enabled          bool   `json:"enabled"`
		Endpoint         string `json:"endpoint"`
		Bucket           string `json:"bucket"`
		AccessKeyID      string `json:"access_key_id"`
		SecretAccessKey  string `json:"secret_access_key"`
		PublicURL        string `json:"public_url"`
		Region           string `json:"region"`
		ForcePathStyle   bool   `json:"force_path_style"`
		ClearCredentials bool   `json:"clear_credentials"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		err(w, 400, "invalid_request", "invalid JSON")
		return
	}
	if in.Enabled && (in.Endpoint == "" || in.Bucket == "") {
		err(w, 422, "invalid_settings", "endpoint and bucket are required when R2 is enabled")
		return
	}
	if in.Endpoint != "" && !isHTTPURL(in.Endpoint) {
		err(w, 422, "invalid_settings", "endpoint must use http or https")
		return
	}
	if in.PublicURL != "" && !isHTTPURL(in.PublicURL) {
		err(w, 422, "invalid_settings", "public URL must use http or https")
		return
	}
	if in.Region == "" {
		in.Region = "auto"
	}
	e := s.store.SaveR2Settings(r.Context(), storage.Update{
		Enabled: in.Enabled, Endpoint: in.Endpoint, Bucket: in.Bucket,
		AccessKeyID: in.AccessKeyID, SecretAccessKey: in.SecretAccessKey,
		PublicURL: in.PublicURL, Region: in.Region, ForcePathStyle: in.ForcePathStyle,
		ClearCredentials: in.ClearCredentials,
	})
	if e != nil {
		status, code := 500, "internal_error"
		if e == storage.ErrEncryptionNotConfigured {
			status, code = 503, "encryption_not_configured"
		}
		err(w, status, code, e.Error())
		return
	}
	updated, e := s.store.R2Settings(r.Context())
	if e != nil {
		err(w, 500, "internal_error", "could not load storage settings")
		return
	}
	jsonOut(w, 200, map[string]any{"data": updated})
}

func isHTTPURL(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *Service) testStorage(w http.ResponseWriter, r *http.Request) {
	if !s.owner(r) {
		err(w, 403, "owner_required", "owner permission required")
		return
	}
	if s.store == nil {
		err(w, 503, "storage_not_available", "storage service is not available")
		return
	}
	if e := s.store.Test(r.Context()); e != nil {
		err(w, 502, storage.HTTPErrorCode(e), "could not connect to R2 bucket")
		return
	}
	jsonOut(w, 200, map[string]any{"data": map[string]string{"status": "ok"}})
}
func (s *Service) owner(r *http.Request) bool {
	u, ok := users.UserFromContext(r.Context())
	return ok && u.Role == "owner"
}
func (s *Service) getAI(w http.ResponseWriter, r *http.Request) {
	if !s.owner(r) {
		err(w, 403, "owner_required", "owner permission required")
		return
	}
	endpoint, model := "", ""
	hasKey := false
	query := "SELECT `key`,value FROM app_settings"
	if s.driver == "postgres" {
		query = "SELECT key,value FROM app_settings"
	}
	rows, e := s.db.QueryContext(r.Context(), query)
	if e == nil {
		defer rows.Close()
		for rows.Next() {
			var k, v string
			_ = rows.Scan(&k, &v)
			if k == "ai_endpoint" {
				endpoint = v
			}
			if k == "ai_model" {
				model = v
			}
			if k == "ai_api_key" {
				hasKey = true
			}
		}
	}
	jsonOut(w, 200, map[string]any{"data": map[string]any{"endpoint": endpoint, "model": model, "has_api_key": hasKey}})
}
func (s *Service) setAI(w http.ResponseWriter, r *http.Request) {
	if !s.owner(r) {
		err(w, 403, "owner_required", "owner permission required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var in struct{ Endpoint, Model, APIKey string }
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		err(w, 400, "invalid_request", "invalid JSON")
		return
	}
	if in.Endpoint == "" || in.Model == "" {
		err(w, 422, "invalid_settings", "endpoint and model are required")
		return
	}
	values := map[string]string{"ai_endpoint": in.Endpoint, "ai_model": in.Model}
	if in.APIKey != "" {
		if len(s.key) != 32 {
			err(w, 503, "encryption_not_configured", "APP_ENCRYPTION_KEY must be 32 bytes")
			return
		}
		v, e := encrypt(s.key, in.APIKey)
		if e != nil {
			err(w, 500, "internal_error", "could not encrypt setting")
			return
		}
		values["ai_api_key"] = v
	}
	for k, v := range values {
		q := "INSERT INTO app_settings (`key`,value) VALUES (?,?) ON DUPLICATE KEY UPDATE value=VALUES(value)"
		if s.driver == "postgres" {
			q = "INSERT INTO app_settings (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value"
		}
		if _, e := s.db.ExecContext(r.Context(), q, k, v); e != nil {
			err(w, 500, "internal_error", "could not save settings")
			return
		}
	}
	jsonOut(w, 200, map[string]any{"data": map[string]string{"endpoint": in.Endpoint, "model": in.Model, "has_api_key": strconv.FormatBool(in.APIKey != "")}})
}
func encrypt(key []byte, plain string) (string, error) {
	b, e := aes.NewCipher(key)
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return "", e
	}
	n := make([]byte, g.NonceSize())
	if _, e = io.ReadFull(rand.Reader, n); e != nil {
		return "", e
	}
	return base64.RawStdEncoding.EncodeToString(g.Seal(n, n, []byte(plain), nil)), nil
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func err(w http.ResponseWriter, status int, code, msg string) {
	jsonOut(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
