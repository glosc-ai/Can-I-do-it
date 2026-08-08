package analysis

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gloscai/template-go-vue3-docker/server/assets"
	"github.com/gloscai/template-go-vue3-docker/server/storage"
)

type Worker struct {
	db     *sql.DB
	driver string
	key    []byte
	assets *assets.Service
	store  *storage.Service
	max    int64
}

func NewWorker(db *sql.DB, driver, encryptionKey string, assetServices ...*assets.Service) *Worker {
	var assetService *assets.Service
	if len(assetServices) > 0 {
		assetService = assetServices[0]
	}
	return &Worker{db: db, driver: driver, key: []byte(encryptionKey), assets: assetService}
}

// NewWorkerWithStorage keeps the worker testable and gives it the same object
// storage boundary as the upload and download handlers.
func NewWorkerWithStorage(db *sql.DB, driver, encryptionKey string, store *storage.Service, max int64, assetService *assets.Service) *Worker {
	return &Worker{db: db, driver: driver, key: []byte(encryptionKey), assets: assetService, store: store, max: max}
}

func (w *Worker) Run(ctx context.Context) {
	// Drain any jobs that were queued before the worker started immediately.
	// Waiting for the first ticker tick makes a freshly started instance look
	// stuck even though there is work ready to be claimed.
	w.process(ctx)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	var id, plan, userID int64
	var filename, mimeType, objectKey string
	// business_plans.object_key is the canonical source object recorded at
	// upload time. Keep the claim query free of outer joins: PostgreSQL rejects
	// FOR UPDATE on the nullable side of a LEFT JOIN, which otherwise causes
	// every queued job to be skipped silently.
	query := "SELECT j.id,j.plan_id,p.user_id,p.filename,p.mime_type,p.object_key FROM analysis_jobs j JOIN business_plans p ON p.id=j.plan_id WHERE j.status='queued' ORDER BY j.id LIMIT 1"
	if w.driver == "postgres" {
		query += " FOR UPDATE OF j SKIP LOCKED"
	}
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, query).Scan(&id, &plan, &userID, &filename, &mimeType, &objectKey); err != nil {
		return
	}
	if _, err = tx.ExecContext(ctx, w.q("UPDATE analysis_jobs SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='queued'"), "running", id); err != nil {
		return
	}
	if err = tx.Commit(); err != nil {
		return
	}
	_, _ = w.db.ExecContext(ctx, w.q("UPDATE business_plans SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), "processing", plan)

	jobStatus, failure, summary := "succeeded", "", ""
	result := FeasibilityResult{PlanID: plan}
	source, sourceErr := w.loadSource(ctx, objectKey, filename, mimeType)
	if sourceErr != nil {
		jobStatus, failure, summary = "failed", sourceErr.Error(), "无法读取上传文件，分析未完成。"
	} else if endpoint, model, key := w.aiSettings(ctx); endpoint == "" || key == "" {
		jobStatus, failure, summary = "failed", "AI provider is not configured", "分析未完成，请先在管理员设置中配置 AI 服务。"
	} else {
		content := userContent(source)
		requestBody := map[string]any{
			"model": model,
			"messages": []map[string]any{
				{"role": "system", "content": skillPrompt()},
				{"role": "user", "content": content},
			},
			"temperature": 0.2,
		}
		body, _ := json.Marshal(requestBody)
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/chat/completions", bytes.NewReader(body))
		if reqErr != nil {
			jobStatus, failure, summary = "failed", reqErr.Error(), "分析请求创建失败。"
		} else {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+key)
			client := &http.Client{Timeout: 4 * time.Minute}
			resp, callErr := client.Do(req)
			if callErr != nil {
				jobStatus, failure, summary = "failed", "could not reach AI provider: "+callErr.Error(), "无法连接 AI 服务。"
			} else {
				defer resp.Body.Close()
				if resp.StatusCode/100 != 2 {
					jobStatus, failure, summary = "failed", fmt.Sprintf("AI provider returned HTTP %d", resp.StatusCode), "AI 服务返回错误。"
				} else {
					var ai struct {
						Choices []struct {
							Message struct {
								Content string `json:"content"`
							} `json:"message"`
						} `json:"choices"`
					}
					if err := json.NewDecoder(io.LimitReader(resp.Body, 2*1024*1024)).Decode(&ai); err != nil || len(ai.Choices) == 0 {
						jobStatus, failure, summary = "failed", "AI provider returned an invalid response", "AI 服务返回了无法解析的结果。"
					} else if normalized, normalizeErr := normalizeResult(ai.Choices[0].Message.Content, plan, source); normalizeErr != nil {
						jobStatus, failure, summary = "failed", normalizeErr.Error(), "AI 结果不是合法的结构化评分。"
					} else {
						result, summary = normalized, normalized.Summary
					}
				}
			}
		}
	}

	payload, _ := json.Marshal(result)
	if jobStatus == "failed" {
		payload = nil
	}
	updateQuery := "UPDATE analysis_jobs SET status=?,error=?,result=NULL,summary=?,overall_score=NULL,verdict='',updated_at=CURRENT_TIMESTAMP WHERE id=?"
	args := []any{jobStatus, failure, summary, id}
	if jobStatus == "succeeded" {
		updateQuery = "UPDATE analysis_jobs SET status=?,error=?,result=?,summary=?,overall_score=?,verdict=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"
		if w.driver == "postgres" {
			updateQuery = "UPDATE analysis_jobs SET status=?,error=?,result=?::jsonb,summary=?,overall_score=?,verdict=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"
		}
		args = []any{jobStatus, failure, string(payload), summary, result.OverallScore, result.Verdict, id}
	}
	if _, err := w.db.ExecContext(ctx, w.q(updateQuery), args...); err != nil {
		return
	}
	if jobStatus == "succeeded" {
		_ = w.saveDimensionScores(ctx, id, result.Dimensions)
		if w.assets != nil {
			planID := plan
			_, _ = w.assets.Save(ctx, userID, &planID, "ai_generated", fmt.Sprintf("analysis-%d.json", id), "application/json", int64(len(payload)), map[string]any{"analysis_job_id": id, "filename": filename}, bytes.NewReader(payload))
		}
	}
	planStatus := "completed"
	if jobStatus == "failed" {
		planStatus = "failed"
	}
	_, _ = w.db.ExecContext(ctx, w.q("UPDATE business_plans SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"), planStatus, plan)
}

func (w *Worker) loadSource(ctx context.Context, key, filename, mimeType string) (documentInput, error) {
	if w.store == nil {
		return documentInput{Filename: filename, MimeType: mimeType, Text: "未配置对象存储，无法读取上传内容。"}, nil
	}
	return readDocument(ctx, w.store, key, filename, mimeType, w.max)
}

func userContent(source documentInput) any {
	instruction := "请分析这份项目材料。所有数字都必须能在材料中找到依据；不确定的内容写入 gaps。保留每一步调查的可复核摘要。\n文件名：" + source.Filename
	if source.ImageDataURL != "" {
		return []map[string]any{{"type": "text", "text": instruction}, {"type": "image_url", "image_url": map[string]string{"url": source.ImageDataURL}}}
	}
	text := source.Text
	if len([]rune(text)) > 120000 {
		text = string([]rune(text)[:120000]) + "\n[文本已截断]"
	}
	return instruction + "\n\n文件内容：\n" + text
}

func (w *Worker) saveDimensionScores(ctx context.Context, jobID int64, dimensions []DimensionScore) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, w.q("DELETE FROM analysis_dimension_scores WHERE analysis_job_id=?"), jobID); err != nil {
		return err
	}
	for _, dimension := range dimensions {
		evidence, _ := json.Marshal(dimension.Evidence)
		gaps, _ := json.Marshal(dimension.Gaps)
		query := "INSERT INTO analysis_dimension_scores (analysis_job_id,dimension_key,dimension_name,score,weight,confidence,reasoning,evidence,gaps) VALUES (?,?,?,?,?,?,?,?,?)"
		if w.driver == "postgres" {
			query = "INSERT INTO analysis_dimension_scores (analysis_job_id,dimension_key,dimension_name,score,weight,confidence,reasoning,evidence,gaps) VALUES (?,?,?,?,?,?,?,?::jsonb,?::jsonb)"
		}
		if _, err := tx.ExecContext(ctx, w.q(query), jobID, dimension.Key, dimension.Name, dimension.Score, dimension.Weight, dimension.Confidence, dimension.Reasoning, string(evidence), string(gaps)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (w *Worker) aiSettings(ctx context.Context) (string, string, string) {
	q := "SELECT `key`,value FROM app_settings"
	if w.driver == "postgres" {
		q = "SELECT key,value FROM app_settings"
	}
	rows, err := w.db.QueryContext(ctx, q)
	if err != nil {
		return "", "", ""
	}
	defer rows.Close()
	var endpoint, model, key string
	for rows.Next() {
		var k, v string
		_ = rows.Scan(&k, &v)
		switch k {
		case "ai_endpoint":
			endpoint = v
		case "ai_model":
			model = v
		case "ai_api_key":
			key, _ = decrypt(w.key, v)
		}
	}
	return endpoint, model, key
}

func decrypt(key []byte, encoded string) (string, error) {
	b, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(b)
	if err != nil {
		return "", err
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	n := g.NonceSize()
	if len(raw) < n {
		return "", fmt.Errorf("invalid encrypted value")
	}
	plain, err := g.Open(nil, raw[:n], raw[n:], nil)
	return string(plain), err
}

func (w *Worker) q(q string) string {
	if w.driver != "postgres" {
		return q
	}
	var b strings.Builder
	n := 0
	for _, part := range strings.Split(q, "?") {
		if n > 0 {
			fmt.Fprintf(&b, "$%d", n)
		}
		b.WriteString(part)
		n++
	}
	return b.String()
}
