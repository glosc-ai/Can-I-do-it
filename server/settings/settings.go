package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/gloscai/template-go-vue3-docker/server/users"
	"io"
	"net/http"
	"strconv"
)

type Service struct {
	db     *sql.DB
	driver string
	key    []byte
}

func New(db *sql.DB, driver, key string) *Service {
	return &Service{db: db, driver: driver, key: []byte(key)}
}
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/settings/ai", s.getAI)
	mux.HandleFunc("PATCH /api/v1/admin/settings/ai", s.setAI)
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
