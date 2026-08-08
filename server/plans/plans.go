package plans

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gloscai/template-go-vue3-docker/server/users"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	db          *sql.DB
	driver, dir string
	max         int64
}
type Plan struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Status    string    `json:"status"`
	Size      int64     `json:"size_bytes"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Analysis struct {
	ID        int64           `json:"id"`
	PlanID    int64           `json:"plan_id"`
	Status    string          `json:"status"`
	Error     string          `json:"error,omitempty"`
	Summary   string          `json:"summary,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type AdminAnalysis struct {
	Analysis
	UserID    int64  `json:"user_id"`
	PlanTitle string `json:"plan_title"`
}

func New(db *sql.DB, driver, dir string, max int64) *Service {
	return &Service{db: db, driver: driver, dir: dir, max: max}
}
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/plans", s.list)
	mux.HandleFunc("POST /api/v1/plans", s.create)
	mux.HandleFunc("POST /api/v1/plans/{id}/analyze", s.analyze)
	mux.HandleFunc("GET /api/v1/plans/{id}/analysis", s.analysis)
	mux.HandleFunc("POST /api/v1/plans/{id}/analysis/retry", s.retryAnalysis)
	mux.HandleFunc("GET /api/v1/plans/{id}", s.get)
	mux.HandleFunc("GET /api/v1/admin/plans", s.adminList)
	mux.HandleFunc("GET /api/v1/admin/analysis", s.adminAnalysis)
}
func (s *Service) list(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	rows, e := s.db.QueryContext(r.Context(), s.q("SELECT id,user_id,title,filename,mime_type,size_bytes,version,status,created_at,updated_at FROM business_plans WHERE user_id=? ORDER BY created_at DESC"), u.ID)
	if e != nil {
		errJSON(w, 500, "internal_error", "could not list plans")
		return
	}
	defer rows.Close()
	items := []Plan{}
	for rows.Next() {
		var p Plan
		if rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Filename, &p.MimeType, &p.Size, &p.Version, &p.Status, &p.CreatedAt, &p.UpdatedAt) == nil {
			items = append(items, p)
		}
	}
	jsonOut(w, 200, map[string]any{"data": items})
}
func (s *Service) create(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	if err := r.ParseMultipartForm(s.max); err != nil {
		errJSON(w, 413, "file_too_large", "file is too large")
		return
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		errJSON(w, 400, "file_required", "file is required")
		return
	}
	defer file.Close()
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = header.Filename
	}
	if header.Size > s.max {
		errJSON(w, 413, "file_too_large", "file is too large")
		return
	}
	if e = os.MkdirAll(s.dir, 0750); e != nil {
		errJSON(w, 500, "storage_error", "could not prepare storage")
		return
	}
	key := fmt.Sprintf("%d-%d-%s", u.ID, time.Now().UnixNano(), filepath.Base(header.Filename))
	path := filepath.Join(s.dir, key)
	out, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if e != nil {
		errJSON(w, 500, "storage_error", "could not save file")
		return
	}
	_, e = io.Copy(out, file)
	out.Close()
	if e != nil {
		errJSON(w, 500, "storage_error", "could not save file")
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
	if s.driver == "postgres" {
		e = s.db.QueryRowContext(r.Context(), "INSERT INTO business_plans (user_id,title,filename,mime_type,object_key,size_bytes) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id,created_at", u.ID, title, p.Filename, p.MimeType, key, p.Size).Scan(&p.ID, &p.CreatedAt)
	} else {
		res, x := s.db.ExecContext(r.Context(), "INSERT INTO business_plans (user_id,title,filename,mime_type,object_key,size_bytes) VALUES (?,?,?,?,?,?)", u.ID, title, p.Filename, p.MimeType, key, p.Size)
		e = x
		if e == nil {
			p.ID, _ = res.LastInsertId()
			p.CreatedAt = time.Now()
		}
	}
	if e != nil {
		_ = os.Remove(path)
		errJSON(w, 500, "internal_error", "could not create plan")
		return
	}
	jsonOut(w, 201, map[string]any{"data": p})
}
func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var p Plan
	e := s.db.QueryRowContext(r.Context(), s.q("SELECT id,user_id,title,filename,mime_type,size_bytes,version,status,created_at,updated_at FROM business_plans WHERE id=? AND user_id=?"), id, u.ID).Scan(&p.ID, &p.UserID, &p.Title, &p.Filename, &p.MimeType, &p.Size, &p.Version, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if e == sql.ErrNoRows {
		errJSON(w, 404, "not_found", "plan not found")
		return
	}
	if e != nil {
		errJSON(w, 500, "internal_error", "could not load plan")
		return
	}
	jsonOut(w, 200, map[string]any{"data": p})
}
func (s *Service) analyze(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var owner int64
	if e := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=?"), id).Scan(&owner); e == sql.ErrNoRows {
		errJSON(w, 404, "not_found", "plan not found")
		return
	} else if e != nil || owner != u.ID {
		errJSON(w, 403, "forbidden", "not your plan")
		return
	}
	var jobID int64
	if s.driver == "postgres" {
		e := s.db.QueryRowContext(r.Context(), "INSERT INTO analysis_jobs (plan_id) VALUES ($1) RETURNING id", id).Scan(&jobID)
		if e != nil {
			errJSON(w, 500, "internal_error", "could not queue analysis")
			return
		}
	} else {
		res, e := s.db.ExecContext(r.Context(), "INSERT INTO analysis_jobs (plan_id) VALUES (?)", id)
		if e != nil {
			errJSON(w, 500, "internal_error", "could not queue analysis")
			return
		}
		jobID, _ = res.LastInsertId()
	}
	_, _ = s.db.ExecContext(r.Context(), s.q("UPDATE business_plans SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), "queued", id)
	jsonOut(w, 202, map[string]any{"data": map[string]any{"id": jobID, "status": "queued"}})
}

func (s *Service) analysis(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		errJSON(w, 400, "invalid_id", "plan id must be a positive integer")
		return
	}
	var ownerID int64
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=? AND user_id=?"), id, u.ID).Scan(&ownerID); err != nil {
		if err == sql.ErrNoRows {
			errJSON(w, 404, "not_found", "plan not found")
		} else {
			errJSON(w, 500, "internal_error", "could not load plan")
		}
		return
	}
	job, err := s.latestAnalysis(r.Context(), id)
	if err == sql.ErrNoRows {
		jsonOut(w, 200, map[string]any{"data": nil})
		return
	}
	if err != nil {
		errJSON(w, 500, "internal_error", "could not load analysis")
		return
	}
	jsonOut(w, 200, map[string]any{"data": job})
}

func (s *Service) retryAnalysis(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		errJSON(w, 400, "invalid_id", "plan id must be a positive integer")
		return
	}
	var ownerID int64
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=?"), id).Scan(&ownerID); err != nil {
		if err == sql.ErrNoRows {
			errJSON(w, 404, "not_found", "plan not found")
		} else {
			errJSON(w, 500, "internal_error", "could not load plan")
		}
		return
	}
	if ownerID != u.ID && u.Role != "owner" {
		errJSON(w, 403, "forbidden", "not your plan")
		return
	}
	var jobID int64
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT id FROM analysis_jobs WHERE plan_id=? AND status='failed' ORDER BY id DESC LIMIT 1"), id).Scan(&jobID); err != nil {
		if err == sql.ErrNoRows {
			errJSON(w, 409, "not_retryable", "plan has no failed analysis")
		} else {
			errJSON(w, 500, "internal_error", "could not find failed analysis")
		}
		return
	}
	if _, err := s.db.ExecContext(r.Context(), s.q("UPDATE analysis_jobs SET status=?,error='',summary='',result=NULL,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='failed'"), "queued", jobID); err != nil {
		errJSON(w, 500, "internal_error", "could not retry analysis")
		return
	}
	_, _ = s.db.ExecContext(r.Context(), s.q("UPDATE business_plans SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), "queued", id)
	jsonOut(w, 202, map[string]any{"data": map[string]any{"id": jobID, "status": "queued"}})
}

func (s *Service) latestAnalysis(ctx context.Context, planID int64) (Analysis, error) {
	var a Analysis
	var raw []byte
	var errText, summary sql.NullString
	var created, updated sql.NullTime
	err := s.db.QueryRowContext(ctx, s.q("SELECT id,plan_id,status,error,summary,result,created_at,updated_at FROM analysis_jobs WHERE plan_id=? ORDER BY id DESC LIMIT 1"), planID).Scan(&a.ID, &a.PlanID, &a.Status, &errText, &summary, &raw, &created, &updated)
	if err != nil {
		return Analysis{}, err
	}
	a.Error, a.Summary = errText.String, summary.String
	if len(raw) > 0 {
		a.Result = json.RawMessage(raw)
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
		errJSON(w, 403, "owner_required", "owner permission required")
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
			errJSON(w, 400, "invalid_status", "status must be queued, running, succeeded, or failed")
			return
		}
		where = " WHERE j.status=?"
		args = append(args, status)
	}
	countQuery := "SELECT COUNT(*) FROM analysis_jobs j" + where
	var total int
	if err := s.db.QueryRowContext(r.Context(), s.q(countQuery), args...).Scan(&total); err != nil {
		errJSON(w, 500, "internal_error", "could not count analyses")
		return
	}
	query := "SELECT j.id,j.plan_id,j.status,j.error,j.summary,j.result,j.created_at,j.updated_at,p.user_id,p.title FROM analysis_jobs j JOIN business_plans p ON p.id=j.plan_id" + where + " ORDER BY j.updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(r.Context(), s.q(query), args...)
	if err != nil {
		errJSON(w, 500, "internal_error", "could not list analyses")
		return
	}
	defer rows.Close()
	items := []AdminAnalysis{}
	for rows.Next() {
		var a AdminAnalysis
		var raw []byte
		var errText, summary sql.NullString
		var created, updated sql.NullTime
		if rows.Scan(&a.ID, &a.PlanID, &a.Status, &errText, &summary, &raw, &created, &updated, &a.UserID, &a.PlanTitle) == nil {
			a.Error = errText.String
			a.Summary = summary.String
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
	jsonOut(w, 200, map[string]any{"data": items, "meta": map[string]int{"page": page, "page_size": pageSize, "total": total}})
}
func (s *Service) adminList(w http.ResponseWriter, r *http.Request) {
	u, ok := users.UserFromContext(r.Context())
	if !ok || u.Role != "owner" {
		errJSON(w, 403, "owner_required", "owner permission required")
		return
	}
	rows, e := s.db.QueryContext(r.Context(), "SELECT id,user_id,title,filename,mime_type,size_bytes,version,status,created_at,updated_at FROM business_plans ORDER BY created_at DESC LIMIT 200")
	if e != nil {
		errJSON(w, 500, "internal_error", "could not list plans")
		return
	}
	defer rows.Close()
	items := []Plan{}
	for rows.Next() {
		var p Plan
		if rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Filename, &p.MimeType, &p.Size, &p.Version, &p.Status, &p.CreatedAt, &p.UpdatedAt) == nil {
			items = append(items, p)
		}
	}
	jsonOut(w, 200, map[string]any{"data": items})
}
func (s *Service) q(q string) string {
	if s.driver != "postgres" {
		return q
	}
	var b strings.Builder
	n := 0
	for _, p := range strings.Split(q, "?") {
		if n > 0 {
			fmt.Fprintf(&b, "$%d", n)
		}
		b.WriteString(p)
		n++
	}
	return b.String()
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func errJSON(w http.ResponseWriter, status int, code, msg string) {
	jsonOut(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
