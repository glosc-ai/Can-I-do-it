package analysis

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Worker struct {
	db     *sql.DB
	driver string
	key    []byte
}

func NewWorker(db *sql.DB, driver, encryptionKey string) *Worker {
	return &Worker{db: db, driver: driver, key: []byte(encryptionKey)}
}
func (w *Worker) Run(ctx context.Context) {
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
	var id, plan int64
	var status string
	query := "SELECT id,plan_id,status FROM analysis_jobs WHERE status='queued' ORDER BY id LIMIT 1"
	if w.driver == "postgres" {
		query += " FOR UPDATE SKIP LOCKED"
	}
	tx, e := w.db.BeginTx(ctx, nil)
	if e != nil {
		return
	}
	defer tx.Rollback()
	if e = tx.QueryRowContext(ctx, query).Scan(&id, &plan, &status); e != nil {
		return
	}
	if _, e = tx.ExecContext(ctx, w.q("UPDATE analysis_jobs SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='queued'"), "running", id); e != nil {
		return
	}
	if e = tx.Commit(); e != nil {
		return
	}
	_, _ = w.db.ExecContext(ctx, w.q("UPDATE business_plans SET status=? WHERE id=?"), "processing", plan)
	result := map[string]any{"feasibility": "analyzed", "plan_id": plan}
	jobStatus, failure, summary := "succeeded", "", "AI analysis completed."
	if endpoint, model, key := w.aiSettings(ctx); endpoint != "" && key != "" {
		requestBody := map[string]any{"model": model, "messages": []map[string]string{{"role": "system", "content": "Analyze this business plan for feasibility and return concise JSON."}, {"role": "user", "content": fmt.Sprintf("Business plan #%d", plan)}}, "temperature": 0.2}
		body, _ := json.Marshal(requestBody)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/chat/completions", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			if resp.StatusCode/100 == 2 {
				var ai struct {
					Choices []struct {
						Message struct {
							Content string `json:"content"`
						} `json:"message"`
					} `json:"choices"`
				}
				if json.NewDecoder(resp.Body).Decode(&ai) == nil && len(ai.Choices) > 0 {
					result["feedback"] = ai.Choices[0].Message.Content
				} else {
					jobStatus, failure, summary = "failed", "AI provider returned an invalid response", "AI analysis failed."
				}
			} else {
				jobStatus, failure, summary = "failed", fmt.Sprintf("AI provider returned HTTP %d", resp.StatusCode), "AI analysis failed."
			}
			resp.Body.Close()
		} else {
			jobStatus, failure, summary = "failed", "could not reach AI provider", "AI analysis failed."
		}
	} else {
		jobStatus, failure, summary = "failed", "AI provider is not configured", "AI analysis failed. Configure the AI provider in the owner console."
	}
	payload, _ := json.Marshal(result)
	updateQuery := "UPDATE analysis_jobs SET status=?,error=?,result=?,summary=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"
	if jobStatus == "failed" {
		updateQuery = "UPDATE analysis_jobs SET status=?,error=?,result=NULL,summary=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"
	} else if w.driver == "postgres" {
		updateQuery = "UPDATE analysis_jobs SET status=?,error=?,result=?::jsonb,summary=?,updated_at=CURRENT_TIMESTAMP WHERE id=?"
	}
	args := []any{jobStatus, failure, summary, id}
	if jobStatus != "failed" {
		args = []any{jobStatus, failure, string(payload), summary, id}
	}
	_, _ = w.db.ExecContext(ctx, w.q(updateQuery), args...)
	planStatus := "completed"
	if jobStatus == "failed" {
		planStatus = "failed"
	}
	_, _ = w.db.ExecContext(ctx, w.q("UPDATE business_plans SET status=? WHERE id=?"), planStatus, plan)
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
	out := ""
	n := 0
	for _, p := range split(q, "?") {
		if n > 0 {
			out += fmt.Sprintf("$%d", n)
		}
		out += p
		n++
	}
	return out
}
func split(s, sep string) []string {
	var out []string
	for {
		idx := -1
		for i := 0; i+len(sep) <= len(s); i++ {
			if s[i:i+len(sep)] == sep {
				idx = i
				break
			}
		}
		if idx < 0 {
			return append(out, s)
		}
		out = append(out, s[:idx])
		s = s[idx+len(sep):]
	}
}
