package users

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID       int64  `json:"id"`
	Sub      string `json:"-"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// AdminUser adds account timestamps to the owner-facing user list. updated_at
// is touched on every SSO login, so it doubles as "last active".
type AdminUser struct {
	User
	UpdatedAt time.Time `json:"updated_at"`
}
type Config struct {
	Driver, DiscoveryURL, ClientID, ClientSecret, RedirectURI, CookieName string
	TTL                                                                   time.Duration
	Secure                                                                bool
}
type Service struct {
	db          *sql.DB
	cfg         Config
	client      *http.Client
	firstUserMu sync.Mutex
}
type ctxKey struct{}

func New(db *sql.DB, cfg Config) *Service {
	return &Service{db: db, cfg: cfg, client: http.DefaultClient}
}
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/auth/callback", s.callback)
	mux.HandleFunc("GET /api/v1/auth/me", s.me)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/admin/users", s.adminUsers)
	mux.HandleFunc("PATCH /api/v1/admin/users/{id}", s.adminUserStatus)
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") || strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		u, ok := s.current(r)
		if !ok {
			writeError(w, 401, "unauthorized", "login required")
			return
		}
		if u.Status != "active" {
			writeError(w, 403, "disabled", "account disabled")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}
func (s *Service) RequireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || u.Role != "owner" {
			writeError(w, 403, "owner_required", "owner permission required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type discovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
}

func (s *Service) endpoints(ctx context.Context) (discovery, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.DiscoveryURL, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return discovery{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return discovery{}, fmt.Errorf("discovery returned %s", resp.Status)
	}
	var d discovery
	return d, json.NewDecoder(resp.Body).Decode(&d)
}
func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ClientID == "" || s.cfg.ClientSecret == "" {
		writeError(w, 503, "sso_not_configured", "SSO is not configured")
		return
	}
	d, err := s.endpoints(r.Context())
	if err != nil {
		writeError(w, 502, "sso_unavailable", "could not load SSO discovery")
		return
	}
	state := random(24)
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state, Path: "/", HttpOnly: true, Secure: s.cfg.Secure, SameSite: http.SameSiteLaxMode, MaxAge: 300})
	q := url.Values{"response_type": {"code"}, "client_id": {s.cfg.ClientID}, "redirect_uri": {s.cfg.RedirectURI}, "scope": {"user:read"}, "state": {state}}
	http.Redirect(w, r, d.AuthorizationEndpoint+"?"+q.Encode(), http.StatusFound)
}
func (s *Service) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("oauth_state")
	if err != nil || c.Value == "" || c.Value != r.URL.Query().Get("state") {
		writeError(w, 400, "invalid_state", "invalid OAuth state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, 400, "oauth_error", r.URL.Query().Get("error_description"))
		return
	}
	d, err := s.endpoints(r.Context())
	if err != nil {
		writeError(w, 502, "sso_unavailable", "could not load SSO discovery")
		return
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {s.cfg.RedirectURI}}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, d.TokenEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.cfg.ClientID, s.cfg.ClientSecret)
	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode/100 != 2 {
		if resp != nil {
			resp.Body.Close()
		}
		writeError(w, 502, "token_exchange_failed", "could not exchange OAuth code")
		return
	}
	defer resp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil || tok.AccessToken == "" {
		writeError(w, 502, "token_exchange_failed", "invalid token response")
		return
	}
	req, _ = http.NewRequestWithContext(r.Context(), http.MethodGet, d.UserInfoEndpoint, nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	resp, err = s.client.Do(req)
	if err != nil || resp.StatusCode/100 != 2 {
		if resp != nil {
			resp.Body.Close()
		}
		writeError(w, 502, "userinfo_failed", "could not read SSO user")
		return
	}
	defer resp.Body.Close()
	var info struct{ Sub, Name, Nickname, Email, Phone, Avatar string }
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil || info.Sub == "" {
		writeError(w, 502, "userinfo_failed", "SSO user has no sub")
		return
	}
	u, err := s.upsertUser(r.Context(), info.Sub, info.Name, info.Nickname, info.Email, info.Avatar)
	if err != nil {
		writeError(w, 500, "user_error", "could not create local user")
		return
	}
	sid := random(48)
	q := s.placeholder("INSERT INTO sessions (id,user_id,expires_at) VALUES (?, ?, ?)")
	if _, err = s.db.ExecContext(r.Context(), q, sid, u.ID, time.Now().Add(s.cfg.TTL)); err != nil {
		writeError(w, 500, "session_error", "could not create session")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.CookieName, Value: sid, Path: "/", HttpOnly: true, Secure: s.cfg.Secure, SameSite: http.SameSiteLaxMode, MaxAge: int(s.cfg.TTL.Seconds())})
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}
func (s *Service) upsertUser(ctx context.Context, sub, name, nickname, email, avatar string) (User, error) {
	s.firstUserMu.Lock()
	defer s.firstUserMu.Unlock()
	q := s.placeholder("SELECT id,sso_sub,name,nickname,email,avatar,role,status FROM users WHERE sso_sub=?")
	var u User
	err := s.db.QueryRowContext(ctx, q, sub).Scan(&u.ID, &u.Sub, &u.Name, &u.Nickname, &u.Email, &u.Avatar, &u.Role, &u.Status)
	if err == nil {
		_, err = s.db.ExecContext(ctx, s.placeholder("UPDATE users SET name=?,nickname=?,email=?,avatar=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), name, nickname, email, avatar, u.ID)
		return u, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	role := "user"
	var count int
	if err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return User{}, err
	}
	if count == 0 {
		role = "owner"
	}
	var id int64
	if s.cfg.Driver == "postgres" {
		err = s.db.QueryRowContext(ctx, "INSERT INTO users (sso_sub,name,nickname,email,avatar,role) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id", sub, name, nickname, email, avatar, role).Scan(&id)
	} else {
		res, e := s.db.ExecContext(ctx, "INSERT INTO users (sso_sub,name,nickname,email,avatar,role) VALUES (?,?,?,?,?,?)", sub, name, nickname, email, avatar, role)
		err = e
		if err == nil {
			id, _ = res.LastInsertId()
		}
	}
	return User{ID: id, Sub: sub, Name: name, Nickname: nickname, Email: email, Avatar: avatar, Role: role, Status: "active"}, err
}
func (s *Service) current(r *http.Request) (User, bool) {
	c, err := r.Cookie(s.cfg.CookieName)
	if err != nil {
		return User{}, false
	}
	q := s.placeholder("SELECT u.id,u.sso_sub,u.name,u.nickname,u.email,u.avatar,u.role,u.status FROM sessions x JOIN users u ON u.id=x.user_id WHERE x.id=? AND x.revoked_at IS NULL AND x.expires_at>? ")
	var u User
	if s.db.QueryRowContext(r.Context(), q, c.Value, time.Now()).Scan(&u.ID, &u.Sub, &u.Name, &u.Nickname, &u.Email, &u.Avatar, &u.Role, &u.Status) != nil {
		return User{}, false
	}
	return u, true
}
func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	u, ok := s.current(r)
	if !ok {
		writeJSON(w, 401, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "login required"}})
		return
	}
	writeJSON(w, 200, map[string]any{"data": u})
}
func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie(s.cfg.CookieName); e == nil {
		_, _ = s.db.ExecContext(r.Context(), s.placeholder("UPDATE sessions SET revoked_at=? WHERE id=?"), time.Now(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.CookieName, MaxAge: -1, Path: "/"})
	w.WriteHeader(http.StatusNoContent)
}
func (s *Service) adminUsers(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok || u.Role != "owner" {
		writeError(w, 403, "owner_required", "owner permission required")
		return
	}
	rows, err := s.db.QueryContext(r.Context(), "SELECT id,sso_sub,name,nickname,email,avatar,role,status,updated_at FROM users ORDER BY id")
	if err != nil {
		writeError(w, 500, "internal_error", "could not list users")
		return
	}
	defer rows.Close()
	items := []AdminUser{}
	for rows.Next() {
		var u AdminUser
		if rows.Scan(&u.ID, &u.Sub, &u.Name, &u.Nickname, &u.Email, &u.Avatar, &u.Role, &u.Status, &u.UpdatedAt) == nil {
			items = append(items, u)
		}
	}
	writeJSON(w, 200, map[string]any{"data": items})
}
func (s *Service) adminUserStatus(w http.ResponseWriter, r *http.Request) {
	me, ok := UserFromContext(r.Context())
	if !ok || me.Role != "owner" {
		writeError(w, 403, "owner_required", "owner permission required")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid_id", "invalid user id")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || (in.Status != "active" && in.Status != "disabled") {
		writeError(w, 422, "invalid_status", "status must be active or disabled")
		return
	}
	var role string
	if s.db.QueryRowContext(r.Context(), s.placeholder("SELECT role FROM users WHERE id=?"), id).Scan(&role) != nil {
		writeError(w, 404, "not_found", "user not found")
		return
	}
	if role == "owner" {
		writeError(w, 422, "owner_protected", "owner cannot be disabled")
		return
	}
	res, e := s.db.ExecContext(r.Context(), s.placeholder("UPDATE users SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), in.Status, id)
	if e != nil {
		writeError(w, 500, "internal_error", "could not update user")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "not_found", "user not found")
		return
	}
	writeJSON(w, 200, map[string]any{"data": map[string]string{"status": in.Status}})
}
func (s *Service) placeholder(q string) string {
	if s.cfg.Driver == "postgres" {
		n := 0
		var b strings.Builder
		for _, p := range strings.Split(q, "?") {
			if n > 0 {
				b.WriteString(fmt.Sprintf("$%d", n))
			}
			b.WriteString(p)
			n++
		}
		return b.String()
	}
	return q
}
func random(n int) string {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
