package assets

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gloscai/template-go-vue3-docker/server/storage"
	"github.com/gloscai/template-go-vue3-docker/server/users"
)

var validSources = map[string]bool{"upload": true, "ai_generated": true, "fetched": true}

type Service struct {
	db     *sql.DB
	driver string
	store  *storage.Service
	max    int64
}

type Asset struct {
	ID          int64          `json:"id"`
	UserID      int64          `json:"user_id"`
	PlanID      *int64         `json:"plan_id,omitempty"`
	Source      string         `json:"source"`
	Name        string         `json:"name"`
	MimeType    string         `json:"mime_type"`
	Size        int64          `json:"size_bytes"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	DownloadURL string         `json:"download_url"`
	CreatedAt   time.Time      `json:"created_at"`
}

func New(db *sql.DB, driver string, store *storage.Service, maxUploadBytes int64) *Service {
	return &Service{db: db, driver: driver, store: store, max: maxUploadBytes}
}

func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/assets", s.list)
	mux.HandleFunc("POST /api/v1/assets", s.create)
	mux.HandleFunc("GET /api/v1/assets/{id}", s.get)
	mux.HandleFunc("GET /api/v1/assets/{id}/download", s.download)
	mux.HandleFunc("DELETE /api/v1/assets/{id}", s.delete)
	mux.HandleFunc("GET /api/v1/admin/assets", s.adminList)
	mux.HandleFunc("DELETE /api/v1/admin/assets/{id}", s.adminDelete)
}

func (s *Service) list(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	source := r.URL.Query().Get("source")
	if source != "" && !validSources[source] {
		errJSON(w, 400, "invalid_source", "source must be upload, ai_generated, or fetched")
		return
	}
	where := " WHERE user_id=?"
	args := []any{u.ID}
	if source != "" {
		where += " AND source=?"
		args = append(args, source)
	}
	query := "SELECT id,user_id,plan_id,source,name,object_key,mime_type,size_bytes,metadata,created_at FROM storage_assets" + where + " ORDER BY created_at DESC LIMIT 200"
	s.writeList(w, r, query, args)
}

func (s *Service) create(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, s.max+1024*1024)
	if err := r.ParseMultipartForm(s.max); err != nil {
		errJSON(w, 413, "file_too_large", "file is too large")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		errJSON(w, 400, "file_required", "file is required")
		return
	}
	defer file.Close()
	if header.Size > s.max {
		errJSON(w, 413, "file_too_large", "file is too large")
		return
	}
	source := strings.TrimSpace(r.FormValue("source"))
	if source == "" {
		source = "upload"
	}
	if !validSources[source] {
		errJSON(w, 400, "invalid_source", "source must be upload, ai_generated, or fetched")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = filepath.Base(header.Filename)
	}
	planID, err := optionalPlanID(r.FormValue("plan_id"))
	if err != nil {
		errJSON(w, 400, "invalid_plan_id", "plan_id must be a positive integer")
		return
	}
	if planID != nil {
		var ownerID int64
		if err := s.db.QueryRowContext(r.Context(), s.q("SELECT user_id FROM business_plans WHERE id=?"), *planID).Scan(&ownerID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
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
	}
	metadata := strings.TrimSpace(r.FormValue("metadata"))
	if metadata == "" {
		metadata = "{}"
	}
	var metadataObject map[string]any
	if json.Unmarshal([]byte(metadata), &metadataObject) != nil || metadataObject == nil {
		errJSON(w, 400, "invalid_metadata", "metadata must be a JSON object")
		return
	}
	key := fmt.Sprintf("users/%d/%s/%d-%s", u.ID, source, time.Now().UnixNano(), safeFilename(header.Filename))
	if err := s.store.Put(r.Context(), key, file, header.Size, header.Header.Get("Content-Type")); err != nil {
		errJSON(w, 500, storage.HTTPErrorCode(err), "could not save asset")
		return
	}
	asset, err := s.insert(r.Context(), u.ID, planID, source, name, key, header.Header.Get("Content-Type"), header.Size, metadata)
	if err != nil {
		_ = s.store.Delete(context.Background(), key)
		errJSON(w, 500, "internal_error", "could not record asset")
		return
	}
	asset.DownloadURL = s.downloadURL(r, asset.ID, key)
	jsonOut(w, 201, map[string]any{"data": asset})
}

// Save is the server-side counterpart to the multipart endpoint. AI and
// fetching workers can call it with source=ai_generated or source=fetched to
// persist binary output without going through an HTTP round trip.
func (s *Service) Save(ctx context.Context, userID int64, planID *int64, source, name, mimeType string, size int64, metadata map[string]any, body io.Reader) (Asset, error) {
	if userID <= 0 || !validSources[source] || strings.TrimSpace(name) == "" {
		return Asset{}, fmt.Errorf("invalid asset metadata")
	}
	if size < 0 || size > s.max {
		return Asset{}, fmt.Errorf("asset exceeds maximum size")
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return Asset{}, err
	}
	key := fmt.Sprintf("users/%d/%s/%d-%s", userID, source, time.Now().UnixNano(), safeFilename(name))
	if err := s.store.Put(ctx, key, body, size, mimeType); err != nil {
		return Asset{}, err
	}
	asset, err := s.RecordExisting(ctx, userID, planID, source, name, key, mimeType, size, string(encoded))
	if err != nil {
		_ = s.store.Delete(context.Background(), key)
		return Asset{}, err
	}
	asset.DownloadURL = fmt.Sprintf("/api/v1/assets/%d/download", asset.ID)
	if target, err := s.store.URL(ctx, key, 15*time.Minute); err == nil && target != "" {
		asset.DownloadURL = target
	}
	return asset, nil
}

// RecordExisting adds database metadata for an object that has already been
// written by the storage service (for example a business plan upload).
func (s *Service) RecordExisting(ctx context.Context, userID int64, planID *int64, source, name, key, mimeType string, size int64, metadata string) (Asset, error) {
	if !validSources[source] {
		return Asset{}, fmt.Errorf("invalid asset source")
	}
	if strings.TrimSpace(metadata) == "" {
		metadata = "{}"
	}
	if _, err := decodeMetadata(metadata); err != nil {
		return Asset{}, err
	}
	return s.insert(ctx, userID, planID, source, name, key, mimeType, size, metadata)
}

func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := positiveID(r.PathValue("id"))
	if err != nil {
		errJSON(w, 400, "invalid_id", "asset id must be a positive integer")
		return
	}
	asset, key, err := s.load(r.Context(), id, u.ID, u.Role == "owner")
	if errors.Is(err, sql.ErrNoRows) {
		errJSON(w, 404, "not_found", "asset not found")
		return
	}
	if err != nil {
		errJSON(w, 500, "internal_error", "could not load asset")
		return
	}
	asset.DownloadURL = s.downloadURL(r, id, key)
	jsonOut(w, 200, map[string]any{"data": asset})
}

func (s *Service) download(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	id, err := positiveID(r.PathValue("id"))
	if err != nil {
		errJSON(w, 400, "invalid_id", "asset id must be a positive integer")
		return
	}
	asset, key, err := s.load(r.Context(), id, u.ID, u.Role == "owner")
	if errors.Is(err, sql.ErrNoRows) {
		errJSON(w, 404, "not_found", "asset not found")
		return
	}
	if err != nil {
		errJSON(w, 500, "internal_error", "could not load asset")
		return
	}
	if target, err := s.store.URL(r.Context(), key, 15*time.Minute); err != nil {
		errJSON(w, 500, storage.HTTPErrorCode(err), "could not create download URL")
		return
	} else if target != "" {
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	body, _, err := s.store.Open(r.Context(), key)
	if err != nil {
		errJSON(w, 404, "not_found", "asset content not found")
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", asset.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeHeaderFilename(asset.Name)))
	content, err := io.ReadAll(body)
	if err != nil {
		errJSON(w, 500, "storage_error", "could not read asset content")
		return
	}
	http.ServeContent(w, r, asset.Name, asset.CreatedAt, bytes.NewReader(content))
}

func (s *Service) delete(w http.ResponseWriter, r *http.Request) {
	u, _ := users.UserFromContext(r.Context())
	s.remove(w, r, u.ID, u.Role == "owner")
}

func (s *Service) adminDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := users.UserFromContext(r.Context())
	if !ok || u.Role != "owner" {
		errJSON(w, 403, "owner_required", "owner permission required")
		return
	}
	s.remove(w, r, 0, true)
}

func (s *Service) remove(w http.ResponseWriter, r *http.Request, userID int64, all bool) {
	id, err := positiveID(r.PathValue("id"))
	if err != nil {
		errJSON(w, 400, "invalid_id", "asset id must be a positive integer")
		return
	}
	asset, key, err := s.load(r.Context(), id, userID, all)
	if errors.Is(err, sql.ErrNoRows) {
		errJSON(w, 404, "not_found", "asset not found")
		return
	}
	if err != nil {
		errJSON(w, 500, "internal_error", "could not load asset")
		return
	}
	if err := s.store.Delete(r.Context(), key); err != nil {
		errJSON(w, 500, storage.HTTPErrorCode(err), "could not delete asset content")
		return
	}
	query := "DELETE FROM storage_assets WHERE id=?"
	args := []any{id}
	if !all {
		query += " AND user_id=?"
		args = append(args, userID)
	}
	result, err := s.db.ExecContext(r.Context(), s.q(query), args...)
	if err != nil {
		errJSON(w, 500, "internal_error", "could not delete asset")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		errJSON(w, 404, "not_found", "asset not found")
		return
	}
	_ = asset
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) adminList(w http.ResponseWriter, r *http.Request) {
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
	source := r.URL.Query().Get("source")
	if source != "" && !validSources[source] {
		errJSON(w, 400, "invalid_source", "source must be upload, ai_generated, or fetched")
		return
	}
	where := ""
	args := []any{}
	if source != "" {
		where = " WHERE source=?"
		args = append(args, source)
	}
	var total int
	if err := s.db.QueryRowContext(r.Context(), s.q("SELECT COUNT(*) FROM storage_assets"+where), args...).Scan(&total); err != nil {
		errJSON(w, 500, "internal_error", "could not count assets")
		return
	}
	query := "SELECT id,user_id,plan_id,source,name,object_key,mime_type,size_bytes,metadata,created_at FROM storage_assets" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	s.writeListWithMeta(w, r, query, args, map[string]int{"page": page, "page_size": pageSize, "total": total})
}

func (s *Service) writeList(w http.ResponseWriter, r *http.Request, query string, args []any) {
	s.writeListWithMeta(w, r, query, args, nil)
}

func (s *Service) writeListWithMeta(w http.ResponseWriter, r *http.Request, query string, args []any, meta map[string]int) {
	rows, err := s.db.QueryContext(r.Context(), s.q(query), args...)
	if err != nil {
		errJSON(w, 500, "internal_error", "could not list assets")
		return
	}
	defer rows.Close()
	items := []Asset{}
	for rows.Next() {
		asset, key, err := scanAsset(rows)
		if err != nil {
			continue
		}
		asset.DownloadURL = s.downloadURL(r, asset.ID, key)
		items = append(items, asset)
	}
	response := map[string]any{"data": items}
	if meta != nil {
		response["meta"] = meta
	}
	jsonOut(w, 200, response)
}

func (s *Service) insert(ctx context.Context, userID int64, planID *int64, source, name, key, mimeType string, size int64, metadata string) (Asset, error) {
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	var asset Asset
	asset.UserID, asset.PlanID, asset.Source, asset.Name, asset.MimeType, asset.Size = userID, planID, source, name, mimeType, size
	if s.driver == "postgres" {
		err := s.db.QueryRowContext(ctx, "INSERT INTO storage_assets (user_id,plan_id,source,name,object_key,mime_type,size_bytes,metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,created_at", userID, planID, source, name, key, mimeType, size, metadata).Scan(&asset.ID, &asset.CreatedAt)
		if err != nil {
			return Asset{}, err
		}
	} else {
		result, err := s.db.ExecContext(ctx, "INSERT INTO storage_assets (user_id,plan_id,source,name,object_key,mime_type,size_bytes,metadata) VALUES (?,?,?,?,?,?,?,?)", userID, planID, source, name, key, mimeType, size, metadata)
		if err != nil {
			return Asset{}, err
		}
		asset.ID, _ = result.LastInsertId()
		asset.CreatedAt = time.Now()
	}
	if asset.Metadata, _ = decodeMetadata(metadata); asset.Metadata == nil {
		asset.Metadata = map[string]any{}
	}
	return asset, nil
}

func (s *Service) load(ctx context.Context, id, userID int64, all bool) (Asset, string, error) {
	query := "SELECT id,user_id,plan_id,source,name,object_key,mime_type,size_bytes,metadata,created_at FROM storage_assets WHERE id=?"
	args := []any{id}
	if !all {
		query += " AND user_id=?"
		args = append(args, userID)
	}
	row := s.db.QueryRowContext(ctx, s.q(query), args...)
	return scanAssetRow(row)
}

func (s *Service) downloadURL(r *http.Request, id int64, key string) string {
	if target, err := s.store.URL(r.Context(), key, 15*time.Minute); err == nil && target != "" {
		return target
	}
	return fmt.Sprintf("/api/v1/assets/%d/download", id)
}

func (s *Service) q(query string) string {
	if s.driver != "postgres" {
		return query
	}
	var b strings.Builder
	n := 0
	for _, part := range strings.Split(query, "?") {
		if n > 0 {
			fmt.Fprintf(&b, "$%d", n)
		}
		b.WriteString(part)
		n++
	}
	return b.String()
}

type scanner interface{ Scan(...any) error }

func scanAsset(rows *sql.Rows) (Asset, string, error) {
	return scanAssetRow(rows)
}

func scanAssetRow(row scanner) (Asset, string, error) {
	var asset Asset
	var planID sql.NullInt64
	var metadata, key string
	if err := row.Scan(&asset.ID, &asset.UserID, &planID, &asset.Source, &asset.Name, &key, &asset.MimeType, &asset.Size, &metadata, &asset.CreatedAt); err != nil {
		return Asset{}, "", err
	}
	if planID.Valid {
		asset.PlanID = &planID.Int64
	}
	asset.Metadata, _ = decodeMetadata(metadata)
	if asset.Metadata == nil {
		asset.Metadata = map[string]any{}
	}
	return asset, key, nil
}

func decodeMetadata(value string) (map[string]any, error) {
	var out map[string]any
	err := json.Unmarshal([]byte(value), &out)
	return out, err
}

func optionalPlanID(value string) (*int64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := positiveID(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func positiveID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}

func safeFilename(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	if value == "." || value == "" {
		return "asset"
	}
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r) {
			return r
		}
		return '-'
	}, value)
	return value
}

func safeHeaderFilename(value string) string {
	value = filepath.Base(value)
	value = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(value, "\"", "'"), "\\", "-"), "\r", "")
	return strings.ReplaceAll(value, "\n", "")
}

func jsonOut(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func errJSON(w http.ResponseWriter, status int, code, message string) {
	jsonOut(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
