package plans

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gloscai/template-go-vue3-docker/server/assets"
	"github.com/gloscai/template-go-vue3-docker/server/database"
	"github.com/gloscai/template-go-vue3-docker/server/httputil"
	"github.com/gloscai/template-go-vue3-docker/server/storage"
	"github.com/gloscai/template-go-vue3-docker/server/users"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	db          *sql.DB
	driver      string
	store       *storage.Service
	assetRecord *assets.Service
	redis       *redis.Client
	dir         string // retained for compatibility with callers of New
	max         int64
}
type Plan struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id,omitempty"`
	Title       string    `json:"title"`
	Filename    string    `json:"filename"`
	MimeType    string    `json:"mime_type"`
	Status      string    `json:"status"`
	Size        int64     `json:"size_bytes"`
	Version     int       `json:"version"`
	Visibility  string    `json:"visibility"`
	AssetID     *int64    `json:"asset_id,omitempty"`
	DownloadURL string    `json:"download_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GalleryPlan is the public, read-only projection of a Plan shown in the
// gallery. It intentionally omits UserID and only exposes the author's
// display identity.
type GalleryPlan struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mime_type"`
	OverallScore *float64  `json:"overall_score,omitempty"`
	Verdict      string    `json:"verdict,omitempty"`
	AuthorName   string    `json:"author_name"`
	AuthorAvatar string    `json:"author_avatar,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// SimilarPlan is the compact shape returned by the fuzzy-match search used
// to warn users about existing public analyses before they submit a new one.
type SimilarPlan struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	OverallScore *float64  `json:"overall_score,omitempty"`
	Verdict      string    `json:"verdict,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Analysis struct {
	ID           int64           `json:"id"`
	PlanID       int64           `json:"plan_id"`
	Status       string          `json:"status"`
	Error        string          `json:"error,omitempty"`
	Summary      string          `json:"summary,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	OverallScore *float64        `json:"overall_score,omitempty"`
	Verdict      string          `json:"verdict,omitempty"`
	Dimensions   []Dimension     `json:"dimensions,omitempty"`
	Process      []Step          `json:"analysis_process,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type Dimension struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Score      float64  `json:"score"`
	Weight     float64  `json:"weight"`
	Confidence float64  `json:"confidence"`
	Reasoning  string   `json:"reasoning"`
	Evidence   []string `json:"evidence"`
	Gaps       []string `json:"gaps"`
}

type Step struct {
	Step      string   `json:"step"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Questions []string `json:"questions"`
}

type AdminAnalysis struct {
	Analysis
	UserID    int64  `json:"user_id"`
	PlanTitle string `json:"plan_title"`
}

func New(db *sql.DB, driver, dir string, max int64) *Service {
	return &Service{db: db, driver: driver, dir: dir, max: max, store: storage.New(nil, driver, "", dir, max, storage.R2Config{})}
}

func NewWithStorage(db *sql.DB, driver string, store *storage.Service, recorder *assets.Service, redisClient *redis.Client, max int64) *Service {
	return &Service{db: db, driver: driver, store: store, assetRecord: recorder, redis: redisClient, max: max}
}
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/plans", s.list)
	mux.HandleFunc("POST /api/v1/plans", s.create)
	mux.HandleFunc("POST /api/v1/plans/{id}/analyze", s.analyze)
	mux.HandleFunc("GET /api/v1/plans/{id}/analysis", s.analysis)
	mux.HandleFunc("POST /api/v1/plans/{id}/analysis/retry", s.retryAnalysis)
	mux.HandleFunc("PATCH /api/v1/plans/{id}/visibility", s.setVisibility)
	mux.HandleFunc("GET /api/v1/plans/{id}", s.get)
	mux.HandleFunc("GET /api/v1/admin/plans", s.adminList)
	mux.HandleFunc("GET /api/v1/admin/analysis", s.adminAnalysis)
	mux.HandleFunc("GET /api/v1/gallery/plans", s.galleryList)
	mux.HandleFunc("GET /api/v1/gallery/plans/{id}", s.galleryGet)
	mux.HandleFunc("GET /api/v1/gallery/similar", s.gallerySimilar)
}
func (s *Service) list(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	rows, e := s.db.QueryContext(r.Context(), s.q("SELECT id,user_id,title,filename,mime_type,size_bytes,version,status,visibility,created_at,updated_at FROM business_plans WHERE user_id=? ORDER BY created_at DESC"), u.ID)
	if e != nil {
		httputil.WriteError(w, 500, "internal_error", "could not list plans")
		return
	}
	defer rows.Close()
	items := []Plan{}
	for rows.Next() {
		var p Plan
		if rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Filename, &p.MimeType, &p.Size, &p.Version, &p.Status, &p.Visibility, &p.CreatedAt, &p.UpdatedAt) == nil {
			items = append(items, p)
		}
	}
	s.enrichAssets(r.Context(), items)
	httputil.WriteJSON(w, 200, map[string]any{"data": items})
}
func (s *Service) create(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	// Bound the complete multipart body (including form overhead), not just
	// the reported file header size, before ParseMultipartForm buffers it.
	r.Body = http.MaxBytesReader(w, r.Body, s.max+1024*1024)
	if err := r.ParseMultipartForm(s.max); err != nil {
		httputil.WriteError(w, 413, "file_too_large", "file is too large")
		return
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		httputil.WriteError(w, 400, "file_required", "file is required")
		return
	}
	defer file.Close()
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = header.Filename
	}
	visibility := strings.TrimSpace(r.FormValue("visibility"))
	if visibility == "" {
		visibility = "private"
	}
	if visibility != "public" && visibility != "private" {
		httputil.WriteError(w, 400, "invalid_visibility", "visibility must be public or private")
		return
	}
	if header.Size > s.max {
		httputil.WriteError(w, 413, "file_too_large", "file is too large")
		return
	}
	if !supportedPlanFile(header.Filename, header.Header.Get("Content-Type")) {
		httputil.WriteError(w, 400, "unsupported_file_type", "supported formats are PDF, DOC, DOCX, TXT, Markdown, PNG, JPG, and WEBP")
		return
	}
	key := fmt.Sprintf("users/%d/upload/%d-%s", u.ID, time.Now().UnixNano(), safeFilename(header.Filename))
	if e = s.store.Put(r.Context(), key, file, header.Size, header.Header.Get("Content-Type")); e != nil {
		httputil.WriteError(w, 500, storage.HTTPErrorCode(e), "could not save file")
		return
	}
	var p Plan
	p.UserID = u.ID
	p.Title = title
	p.Filename = header.Filename
	p.MimeType = header.Header.Get("Content-Type")
	p.Size = header.Size
	p.Version = 1
	p.Status = "uploaded"
	p.Visibility = visibility
	if s.driver == "postgres" {
		e = s.db.QueryRowContext(r.Context(), "INSERT INTO business_plans (user_id,title,filename,mime_type,object_key,size_bytes,visibility) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id,created_at", u.ID, title, p.Filename, p.MimeType, key, p.Size, visibility).Scan(&p.ID, &p.CreatedAt)
	} else {
		res, x := s.db.ExecContext(r.Context(), "INSERT INTO business_plans (user_id,title,filename,mime_type,object_key,size_bytes,visibility) VALUES (?,?,?,?,?,?,?)", u.ID, title, p.Filename, p.MimeType, key, p.Size, visibility)
		e = x
		if e == nil {
			p.ID, _ = res.LastInsertId()
			p.CreatedAt = time.Now()
		}
	}
	if e != nil {
		_ = s.store.Delete(context.Background(), key)
		httputil.WriteError(w, 500, "internal_error", "could not create plan")
		return
	}
	if s.assetRecord != nil {
		planID := p.ID
		var recorded assets.Asset
		if recorded, e = s.assetRecord.RecordExisting(r.Context(), u.ID, &planID, "upload", p.Filename, key, p.MimeType, p.Size, "{}"); e != nil {
			_, _ = s.db.ExecContext(r.Context(), s.q("DELETE FROM business_plans WHERE id=?"), p.ID)
			_ = s.store.Delete(context.Background(), key)
			httputil.WriteError(w, 500, "internal_error", "could not record uploaded asset")
			return
		}
		p.AssetID = &recorded.ID
		p.DownloadURL = fmt.Sprintf("/api/v1/assets/%d/download", recorded.ID)
	}
	httputil.WriteJSON(w, 201, map[string]any{"data": p})
}

func safeFilename(filename string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." {
		return "upload"
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r) {
			return r
		}
		return '-'
	}, filename)
}

func supportedPlanFile(filename, mimeType string) bool {
	name := strings.ToLower(strings.TrimSpace(filename))
	for _, extension := range []string{".pdf", ".doc", ".docx", ".txt", ".md", ".png", ".jpg", ".jpeg", ".webp"} {
		if strings.HasSuffix(name, extension) {
			return true
		}
	}
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	return strings.HasPrefix(mimeType, "image/") || strings.Contains(mimeType, "pdf") || strings.Contains(mimeType, "word") || strings.HasPrefix(mimeType, "text/")
}
func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var p Plan
	e := s.db.QueryRowContext(r.Context(), s.q("SELECT id,user_id,title,filename,mime_type,size_bytes,version,status,visibility,created_at,updated_at FROM business_plans WHERE id=? AND user_id=?"), id, u.ID).Scan(&p.ID, &p.UserID, &p.Title, &p.Filename, &p.MimeType, &p.Size, &p.Version, &p.Status, &p.Visibility, &p.CreatedAt, &p.UpdatedAt)
	if e == sql.ErrNoRows {
		httputil.WriteError(w, 404, "not_found", "plan not found")
		return
	}
	if e != nil {
		httputil.WriteError(w, 500, "internal_error", "could not load plan")
		return
	}
	plans := []Plan{p}
	s.enrichAssets(r.Context(), plans)
	httputil.WriteJSON(w, 200, map[string]any{"data": plans[0]})
}
func (s *Service) analyze(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httputil.WriteError(w, 400, "invalid_id", "plan id must be a positive integer")
		return
	}
	var owner int64
	switch e := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=?"), id).Scan(&owner); {
	case e == sql.ErrNoRows:
		httputil.WriteError(w, 404, "not_found", "plan not found")
		return
	case e != nil:
		// A real database error — do not pretend it is an authorisation failure.
		httputil.WriteError(w, 500, "internal_error", "could not load plan")
		return
	case owner != u.ID:
		httputil.WriteError(w, 403, "forbidden", "not your plan")
		return
	}
	var jobID int64
	if s.driver == "postgres" {
		e := s.db.QueryRowContext(r.Context(), "INSERT INTO analysis_jobs (plan_id) VALUES ($1) RETURNING id", id).Scan(&jobID)
		if e != nil {
			httputil.WriteError(w, 500, "internal_error", "could not queue analysis")
			return
		}
	} else {
		res, e := s.db.ExecContext(r.Context(), "INSERT INTO analysis_jobs (plan_id) VALUES (?)", id)
		if e != nil {
			httputil.WriteError(w, 500, "internal_error", "could not queue analysis")
			return
		}
		jobID, _ = res.LastInsertId()
	}
	_, _ = s.db.ExecContext(r.Context(), s.q("UPDATE business_plans SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), "queued", id)
	httputil.WriteJSON(w, 202, map[string]any{"data": map[string]any{"id": jobID, "status": "queued"}})
}

func (s *Service) analysis(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httputil.WriteError(w, 400, "invalid_id", "plan id must be a positive integer")
		return
	}
	var ownerID int64
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=? AND user_id=?"), id, u.ID).Scan(&ownerID); err != nil {
		if err == sql.ErrNoRows {
			httputil.WriteError(w, 404, "not_found", "plan not found")
		} else {
			httputil.WriteError(w, 500, "internal_error", "could not load plan")
		}
		return
	}
	job, err := s.latestAnalysis(r.Context(), id)
	if err == sql.ErrNoRows {
		httputil.WriteJSON(w, 200, map[string]any{"data": nil})
		return
	}
	if err != nil {
		httputil.WriteError(w, 500, "internal_error", "could not load analysis")
		return
	}
	httputil.WriteJSON(w, 200, map[string]any{"data": job})
}

func (s *Service) retryAnalysis(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httputil.WriteError(w, 400, "invalid_id", "plan id must be a positive integer")
		return
	}
	var ownerID int64
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=?"), id).Scan(&ownerID); err != nil {
		if err == sql.ErrNoRows {
			httputil.WriteError(w, 404, "not_found", "plan not found")
		} else {
			httputil.WriteError(w, 500, "internal_error", "could not load plan")
		}
		return
	}
	if ownerID != u.ID && u.Role != "owner" {
		httputil.WriteError(w, 403, "forbidden", "not your plan")
		return
	}
	var jobID int64
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT id FROM analysis_jobs WHERE plan_id=? AND status='failed' ORDER BY id DESC LIMIT 1"), id).Scan(&jobID); err != nil {
		if err == sql.ErrNoRows {
			httputil.WriteError(w, 409, "not_retryable", "plan has no failed analysis")
		} else {
			httputil.WriteError(w, 500, "internal_error", "could not find failed analysis")
		}
		return
	}
	if _, err := s.db.ExecContext(r.Context(), s.q("UPDATE analysis_jobs SET status=?,error='',summary='',result=NULL,overall_score=NULL,verdict='',updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='failed'"), "queued", jobID); err != nil {
		httputil.WriteError(w, 500, "internal_error", "could not retry analysis")
		return
	}
	_, _ = s.db.ExecContext(r.Context(), s.q("UPDATE business_plans SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), "queued", id)
	httputil.WriteJSON(w, 202, map[string]any{"data": map[string]any{"id": jobID, "status": "queued"}})
}

func (s *Service) setVisibility(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httputil.WriteError(w, 400, "invalid_id", "plan id must be a positive integer")
		return
	}
	var in struct {
		Visibility string `json:"visibility"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || (in.Visibility != "public" && in.Visibility != "private") {
		httputil.WriteError(w, 422, "invalid_visibility", "visibility must be public or private")
		return
	}
	var ownerID int64
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=?"), id).Scan(&ownerID); err != nil {
		if err == sql.ErrNoRows {
			httputil.WriteError(w, 404, "not_found", "plan not found")
		} else {
			httputil.WriteError(w, 500, "internal_error", "could not load plan")
		}
		return
	}
	if ownerID != u.ID {
		httputil.WriteError(w, 403, "forbidden", "not your plan")
		return
	}
	if _, err := s.db.ExecContext(r.Context(), s.q("UPDATE business_plans SET visibility=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), in.Visibility, id); err != nil {
		httputil.WriteError(w, 500, "internal_error", "could not update visibility")
		return
	}
	httputil.WriteJSON(w, 200, map[string]any{"data": map[string]string{"visibility": in.Visibility}})
}

// galleryRateLimit enforces a per-IP request budget on the anonymous gallery
// endpoints using a Redis fixed window counter, so the fuzzy-match search and
// listing cannot be scraped or hammered by unauthenticated clients.
func (s *Service) galleryRateLimit(w http.ResponseWriter, r *http.Request) bool {
	if s.redis == nil {
		return true
	}
	const limit = 30
	const window = time.Minute
	key := "ratelimit:gallery:" + clientIP(r)
	count, err := s.redis.Incr(r.Context(), key).Result()
	if err != nil {
		// Redis being unavailable should not take down a public read endpoint.
		return true
	}
	if count == 1 {
		s.redis.Expire(r.Context(), key, window)
	}
	if count > limit {
		httputil.WriteError(w, 429, "rate_limited", "too many requests, please slow down")
		return false
	}
	return true
}

// clientIP identifies the caller for rate limiting. It deliberately does not
// trust X-Forwarded-For: nginx's $proxy_add_x_forwarded_for appends to
// whatever the client already sent, so the leftmost entry is attacker
// controlled and would let anyone reset their own rate-limit key on every
// request. X-Real-IP is safe because nginx sets it from $remote_addr via
// proxy_set_header, which replaces rather than appends, so a client-supplied
// value is discarded before it reaches this handler.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// latestJobFragment returns the SELECT column expressions and (for
// Postgres) the JOIN clause needed to pull each plan's most recent
// analysis_jobs row inline. Postgres uses a LATERAL join; MySQL, which has
// no LATERAL support in the versions targeted here, inlines the same lookup
// as two correlated scalar subqueries instead. Centralising this here keeps
// the "latest job" definition in one place for every gallery query that
// needs it.
func latestJobFragment(driver string) (cols, join string) {
	if driver == "postgres" {
		return "j.overall_score,j.verdict", "LEFT JOIN LATERAL (SELECT overall_score,verdict FROM analysis_jobs WHERE plan_id=p.id ORDER BY id DESC LIMIT 1) j ON true"
	}
	return "(SELECT overall_score FROM analysis_jobs WHERE plan_id=p.id ORDER BY id DESC LIMIT 1),(SELECT verdict FROM analysis_jobs WHERE plan_id=p.id ORDER BY id DESC LIMIT 1)", ""
}

func (s *Service) galleryList(w http.ResponseWriter, r *http.Request) {
	if !s.galleryRateLimit(w, r) {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	jobCols, jobJoin := latestJobFragment(s.driver)
	query := s.q(fmt.Sprintf(`SELECT p.id,p.title,p.filename,p.mime_type,p.created_at,%s,
		COALESCE(NULLIF(u.nickname,''),u.name) AS author_name,u.avatar
		FROM business_plans p
		JOIN users u ON u.id=p.user_id
		%s
		WHERE p.visibility='public' AND p.status='completed'
		ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, jobCols, jobJoin))
	rows, err := s.db.QueryContext(r.Context(), query, pageSize, (page-1)*pageSize)
	if err != nil {
		httputil.WriteError(w, 500, "internal_error", "could not list gallery plans")
		return
	}
	defer rows.Close()
	items := []GalleryPlan{}
	for rows.Next() {
		var g GalleryPlan
		var score sql.NullFloat64
		var verdict, avatar sql.NullString
		if rows.Scan(&g.ID, &g.Title, &g.Filename, &g.MimeType, &g.CreatedAt, &score, &verdict, &g.AuthorName, &avatar) == nil {
			if score.Valid {
				g.OverallScore = &score.Float64
			}
			g.Verdict = verdict.String
			g.AuthorAvatar = avatar.String
			items = append(items, g)
		}
	}
	httputil.WriteJSON(w, 200, map[string]any{"data": items})
}

func (s *Service) galleryGet(w http.ResponseWriter, r *http.Request) {
	if !s.galleryRateLimit(w, r) {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httputil.WriteError(w, 400, "invalid_id", "plan id must be a positive integer")
		return
	}
	var p Plan
	var authorName string
	var avatar sql.NullString
	e := s.db.QueryRowContext(r.Context(), s.q(`SELECT p.id,p.user_id,p.title,p.filename,p.mime_type,p.size_bytes,p.version,p.status,p.visibility,p.created_at,p.updated_at,
		COALESCE(NULLIF(u.nickname,''),u.name),u.avatar
		FROM business_plans p JOIN users u ON u.id=p.user_id
		WHERE p.id=? AND p.visibility='public' AND p.status='completed'`), id).
		Scan(&p.ID, &p.UserID, &p.Title, &p.Filename, &p.MimeType, &p.Size, &p.Version, &p.Status, &p.Visibility, &p.CreatedAt, &p.UpdatedAt, &authorName, &avatar)
	if e == sql.ErrNoRows {
		// Deliberately identical to "does not exist" for a private plan: the
		// gallery must not leak which private plan IDs exist.
		httputil.WriteError(w, 404, "not_found", "plan not found")
		return
	}
	if e != nil {
		httputil.WriteError(w, 500, "internal_error", "could not load plan")
		return
	}
	// Anonymous visitors get the author's display identity via author_name/
	// author_avatar below; the internal numeric user_id must not leak here,
	// unlike the owner-scoped /api/v1/plans/{id} endpoint.
	p.UserID = 0
	job, err := s.latestAnalysis(r.Context(), id)
	var analysis *Analysis
	if err == nil {
		analysis = &job
	} else if err != sql.ErrNoRows {
		httputil.WriteError(w, 500, "internal_error", "could not load analysis")
		return
	}
	httputil.WriteJSON(w, 200, map[string]any{"data": map[string]any{
		"plan":          p,
		"author_name":   authorName,
		"author_avatar": avatar.String,
		"analysis":      analysis,
	}})
}

func (s *Service) gallerySimilar(w http.ResponseWriter, r *http.Request) {
	if !s.galleryRateLimit(w, r) {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		httputil.WriteJSON(w, 200, map[string]any{"data": []SimilarPlan{}})
		return
	}
	if len(q) > 200 {
		q = q[:200]
	}
	jobCols, jobJoin := latestJobFragment(s.driver)
	var query string
	var args []any
	if s.driver == "postgres" {
		query = s.q(fmt.Sprintf(`SELECT p.id,p.title,%s,p.created_at
			FROM business_plans p
			%s
			WHERE p.visibility='public' AND p.status='completed' AND similarity(p.title,?) > 0.3
			ORDER BY similarity(p.title,?) DESC LIMIT 5`, jobCols, jobJoin))
		args = []any{q, q}
	} else {
		query = s.q(fmt.Sprintf(`SELECT p.id,p.title,%s,p.created_at
			FROM business_plans p
			WHERE p.visibility='public' AND p.status='completed' AND MATCH(p.title) AGAINST(? IN NATURAL LANGUAGE MODE)
			LIMIT 5`, jobCols))
		args = []any{q}
	}
	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		httputil.WriteError(w, 500, "internal_error", "could not search similar plans")
		return
	}
	defer rows.Close()
	items := []SimilarPlan{}
	for rows.Next() {
		var sp SimilarPlan
		var score sql.NullFloat64
		var verdict sql.NullString
		if rows.Scan(&sp.ID, &sp.Title, &score, &verdict, &sp.CreatedAt) == nil {
			if score.Valid {
				sp.OverallScore = &score.Float64
			}
			sp.Verdict = verdict.String
			items = append(items, sp)
		}
	}
	httputil.WriteJSON(w, 200, map[string]any{"data": items})
}

func (s *Service) latestAnalysis(ctx context.Context, planID int64) (Analysis, error) {
	var a Analysis
	var raw []byte
	var errText, summary sql.NullString
	var score sql.NullFloat64
	var verdict sql.NullString
	var created, updated sql.NullTime
	err := s.db.QueryRowContext(ctx, s.q("SELECT id,plan_id,status,error,summary,result,overall_score,verdict,created_at,updated_at FROM analysis_jobs WHERE plan_id=? ORDER BY id DESC LIMIT 1"), planID).Scan(&a.ID, &a.PlanID, &a.Status, &errText, &summary, &raw, &score, &verdict, &created, &updated)
	if err != nil {
		return Analysis{}, err
	}
	a.Error, a.Summary = errText.String, summary.String
	if score.Valid {
		a.OverallScore = &score.Float64
	}
	a.Verdict = verdict.String
	if len(raw) > 0 {
		a.Result = json.RawMessage(raw)
		var details struct {
			OverallScore *float64    `json:"overall_score"`
			Verdict      string      `json:"verdict"`
			Dimensions   []Dimension `json:"dimensions"`
			Process      []Step      `json:"analysis_process"`
		}
		if json.Unmarshal(raw, &details) == nil {
			if details.OverallScore != nil {
				a.OverallScore = details.OverallScore
			}
			if details.Verdict != "" {
				a.Verdict = details.Verdict
			}
			if len(details.Dimensions) > 0 {
				a.Dimensions = details.Dimensions
			}
			if len(details.Process) > 0 {
				a.Process = details.Process
			}
		}
	}
	if created.Valid {
		a.CreatedAt = created.Time
	}
	if updated.Valid {
		a.UpdatedAt = updated.Time
	}
	return a, nil
}

func (s *Service) adminAnalysis(w http.ResponseWriter, r *http.Request) {
	u, ok := users.UserFromContext(r.Context())
	if !ok || u.Role != "owner" {
		httputil.WriteError(w, 403, "owner_required", "owner permission required")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	status := r.URL.Query().Get("status")
	where := ""
	args := []any{}
	if status != "" {
		if status != "queued" && status != "running" && status != "succeeded" && status != "failed" {
			httputil.WriteError(w, 400, "invalid_status", "status must be queued, running, succeeded, or failed")
			return
		}
		where = " WHERE j.status=?"
		args = append(args, status)
	}
	countQuery := "SELECT COUNT(*) FROM analysis_jobs j" + where
	var total int
	if err := s.db.QueryRowContext(r.Context(), s.q(countQuery), args...).Scan(&total); err != nil {
		httputil.WriteError(w, 500, "internal_error", "could not count analyses")
		return
	}
	query := "SELECT j.id,j.plan_id,j.status,j.error,j.summary,j.result,j.overall_score,j.verdict,j.created_at,j.updated_at,p.user_id,p.title FROM analysis_jobs j JOIN business_plans p ON p.id=j.plan_id" + where + " ORDER BY j.updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(r.Context(), s.q(query), args...)
	if err != nil {
		httputil.WriteError(w, 500, "internal_error", "could not list analyses")
		return
	}
	defer rows.Close()
	items := []AdminAnalysis{}
	for rows.Next() {
		var a AdminAnalysis
		var raw []byte
		var errText, summary sql.NullString
		var score sql.NullFloat64
		var verdict sql.NullString
		var created, updated sql.NullTime
		if rows.Scan(&a.ID, &a.PlanID, &a.Status, &errText, &summary, &raw, &score, &verdict, &created, &updated, &a.UserID, &a.PlanTitle) == nil {
			a.Error = errText.String
			a.Summary = summary.String
			if score.Valid {
				a.OverallScore = &score.Float64
			}
			a.Verdict = verdict.String
			if len(raw) > 0 {
				a.Result = json.RawMessage(raw)
			}
			if created.Valid {
				a.CreatedAt = created.Time
			}
			if updated.Valid {
				a.UpdatedAt = updated.Time
			}
			items = append(items, a)
		}
	}
	httputil.WriteJSON(w, 200, map[string]any{"data": items, "meta": map[string]int{"page": page, "page_size": pageSize, "total": total}})
}
func (s *Service) adminList(w http.ResponseWriter, r *http.Request) {
	u, ok := users.UserFromContext(r.Context())
	if !ok || u.Role != "owner" {
		httputil.WriteError(w, 403, "owner_required", "owner permission required")
		return
	}
	rows, e := s.db.QueryContext(r.Context(), "SELECT id,user_id,title,filename,mime_type,size_bytes,version,status,visibility,created_at,updated_at FROM business_plans ORDER BY created_at DESC LIMIT 200")
	if e != nil {
		httputil.WriteError(w, 500, "internal_error", "could not list plans")
		return
	}
	defer rows.Close()
	items := []Plan{}
	for rows.Next() {
		var p Plan
		if rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Filename, &p.MimeType, &p.Size, &p.Version, &p.Status, &p.Visibility, &p.CreatedAt, &p.UpdatedAt) == nil {
			items = append(items, p)
		}
	}
	s.enrichAssets(r.Context(), items)
	httputil.WriteJSON(w, 200, map[string]any{"data": items})
}

// enrichAssets loads the latest storage asset for every plan in the slice in
// a single query instead of one query per plan (N+1). It mutates the slice
// elements in-place via a pointer map keyed by plan ID.
func (s *Service) enrichAssets(ctx context.Context, plans []Plan) {
	if len(plans) == 0 {
		return
	}

	// Build the IN clause and collect a pointer map so we can write back.
	ids := make([]int64, len(plans))
	byID := make(map[int64]*Plan, len(plans))
	for i := range plans {
		ids[i] = plans[i].ID
		byID[plans[i].ID] = &plans[i]
	}

	// Build a driver-appropriate IN (?,?,…) query. We use a manual approach
	// here because database.Placeholder only rewrites ? → $N positionally,
	// which is exactly what we need.
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(
		"SELECT DISTINCT ON (plan_id) id, plan_id, object_key FROM storage_assets WHERE plan_id IN (%s) ORDER BY plan_id, id DESC",
		strings.Join(placeholders, ","),
	)
	if s.driver != "postgres" {
		// MySQL does not support DISTINCT ON; use a subquery instead.
		query = fmt.Sprintf(
			"SELECT sa.id, sa.plan_id, sa.object_key FROM storage_assets sa INNER JOIN (SELECT plan_id, MAX(id) AS max_id FROM storage_assets WHERE plan_id IN (%s) GROUP BY plan_id) latest ON sa.id = latest.max_id",
			strings.Join(placeholders, ","),
		)
	}

	rows, err := s.db.QueryContext(ctx, database.Placeholder(s.driver, query), args...)
	if err != nil {
		// Non-fatal: plans just won't have download URLs.
		return
	}
	defer rows.Close()

	for rows.Next() {
		var assetID, planID int64
		var key string
		if rows.Scan(&assetID, &planID, &key) != nil {
			continue
		}
		p, ok := byID[planID]
		if !ok {
			continue
		}
		p.AssetID = &assetID
		p.DownloadURL = fmt.Sprintf("/api/v1/assets/%d/download", assetID)
		if s.store != nil {
			if url, err := s.store.URL(ctx, key, 15*time.Minute); err == nil && url != "" {
				p.DownloadURL = url
			}
		}
	}
}
func (s *Service) q(q string) string {
	return database.Placeholder(s.driver, q)
}
