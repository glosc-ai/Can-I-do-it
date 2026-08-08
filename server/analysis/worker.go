package analysis

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
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
			// The provider sits behind a gateway with a short first-byte timeout.
			// Streaming keeps that connection active while the model generates the
			// (comparatively large) nine-dimension JSON result.
			"stream": true,
			"messages": []map[string]any{
				{"role": "system", "content": skillPrompt()},
				{"role": "user", "content": content},
			},
			"temperature": 0.2,
		}
		body, _ := json.Marshal(requestBody)
		client := &http.Client{Timeout: 4 * time.Minute}
		var resp *http.Response
		var callErr error
		for attempt := 0; attempt < 2; attempt++ {
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/chat/completions", bytes.NewReader(body))
			if reqErr != nil {
				callErr = reqErr
				break
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+key)
			req.Header.Set("Idempotency-Key", fmt.Sprintf("analysis-%d", id))
			// A failed keep-alive connection can surface as EOF before headers.
			// Do not reuse it for the retry.
			req.Close = true
			resp, callErr = client.Do(req)
			if resp != nil || callErr == nil || !retryableProviderRequestError(callErr) || attempt == 1 {
				break
			}
			select {
			case <-ctx.Done():
				attempt = 2
			case <-time.After(250 * time.Millisecond):
			}
		}
		if callErr != nil && resp == nil {
			failure, summary = providerRequestFailure(callErr)
			jobStatus = "failed"
		} else {
			defer resp.Body.Close()
			if resp.StatusCode/100 != 2 {
				failure, summary = providerHTTPFailure(resp)
				jobStatus = "failed"
			} else {
				responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
				if len(responseBody) > 2*1024*1024 {
					jobStatus, failure, summary = "failed", "AI 服务响应过大。", "AI 服务返回内容超过 2 MB 限制，请缩短输出或检查模型配置。"
				} else if len(bytes.TrimSpace(responseBody)) == 0 {
					jobStatus = "failed"
					if readErr != nil {
						failure, summary = "无法读取 AI 服务响应。", "AI 服务响应读取失败，请检查网关连接和上游服务日志。"
					} else {
						failure, summary = "AI 服务返回了空响应，可能在生成结果时断开了连接。", "AI 服务没有返回可用内容，请检查网关超时和上游服务日志。"
					}
				} else if content, err := providerResponseContent(responseBody); err != nil {
					jobStatus = "failed"
					if errors.Is(err, io.EOF) {
						failure, summary = "AI 服务返回了空响应，可能在生成结果时断开了连接。", "AI 服务没有返回可用内容，请检查网关超时和上游服务日志。"
					} else {
						failure, summary = "AI 服务返回格式不完整："+err.Error(), "AI 服务响应格式异常，请检查模型网关的 OpenAI 兼容配置。"
					}
				} else if normalized, normalizeErr := normalizeResult(content, plan, source); normalizeErr != nil {
					jobStatus, failure, summary = "failed", "AI 返回内容无法转换为结构化评分："+normalizeErr.Error(), "AI 结果格式不符合分析要求，请重试或检查模型配置。"
				} else {
					result, summary = normalized, normalized.Summary
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

// providerRequestFailure turns transport errors into messages that tell the
// operator what can actually be fixed. In particular, EOF means the peer
// closed the connection before sending HTTP response headers; it is different
// from a provider HTTP error or an invalid AI response body.
func providerRequestFailure(err error) (string, string) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "AI 服务响应超时。", "AI 服务在规定时间内没有返回结果，请检查模型耗时或网关超时配置。"
	}
	if errors.Is(err, context.Canceled) {
		return "AI 分析请求已取消。", "任务可能因服务重启或关闭而中断，请重新提交分析。"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "无法解析 AI 服务域名。", "请检查 AI Endpoint 的域名、服务器 DNS 和网络配置。"
	}
	var certErr x509.CertificateInvalidError
	var hostnameErr x509.HostnameError
	var authorityErr x509.UnknownAuthorityError
	if errors.As(err, &certErr) || errors.As(err, &hostnameErr) || errors.As(err, &authorityErr) {
		return "AI 服务 TLS 证书验证失败。", "请检查 Endpoint 域名、证书有效期和证书链配置。"
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return "AI 服务拒绝了连接。", "请确认 AI 服务正在运行，并检查 Endpoint 端口及防火墙配置。"
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return "AI 服务连接被重置。", "连接被 AI 服务或中间网关中断，请检查上游服务和网关日志。"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "AI 服务连接超时。", "无法及时连接或读取 AI 服务，请检查网络和网关超时配置。"
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "AI 服务在返回结果前关闭了连接（EOF）。", "AI 服务可能已经接受请求，但结果没有成功传回，请检查网关连接和上游服务日志。"
	}
	return "无法连接 AI 服务。", "请检查 AI Endpoint、DNS 和服务器出网网络。"
}

func retryableProviderRequestError(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET)
}

// providerResponseContent extracts the assistant text from an
// OpenAI-compatible response. A few gateways return content as an array of
// text parts, so accepting both forms avoids treating an otherwise valid
// result as an empty choices response.
func providerResponseContent(raw []byte) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return "", io.EOF
	}
	if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("data:")) {
		return providerStreamContent(bytes.NewReader(raw))
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return string(raw), nil
	}
	if len(envelope.Choices) == 0 {
		return "", fmt.Errorf("AI 服务返回成功状态但没有 choices 内容")
	}
	content := envelope.Choices[0].Message.Content
	if len(content) == 0 || string(content) == "null" {
		return "", fmt.Errorf("AI 服务返回的 message.content 为空")
	}
	var text string
	if json.Unmarshal(content, &text) == nil {
		if strings.TrimSpace(text) == "" {
			return "", fmt.Errorf("AI 服务返回的 message.content 为空")
		}
		return text, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &parts) == nil {
		var b strings.Builder
		for _, part := range parts {
			if part.Text != "" {
				b.WriteString(part.Text)
			}
		}
		if b.Len() > 0 {
			return b.String(), nil
		}
	}
	return "", fmt.Errorf("AI 服务返回的 message.content 不是文本")
}

// providerStreamContent converts OpenAI-compatible SSE chunks into the same
// assistant text returned by a non-streaming response. A gateway may emit a
// final regular JSON envelope even when stream=true, so that shape is handled
// by providerResponseContent before this helper is called.
func providerStreamContent(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	var out strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var envelope struct {
			Choices []struct {
				Delta struct {
					Content json.RawMessage `json:"content"`
				} `json:"delta"`
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
			return "", fmt.Errorf("invalid streaming event: %w", err)
		}
		for _, choice := range envelope.Choices {
			content := choice.Delta.Content
			if len(content) == 0 {
				content = choice.Message.Content
			}
			if text, ok := contentText(content); ok {
				out.WriteString(text)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading streaming response: %w", err)
	}
	if strings.TrimSpace(out.String()) == "" {
		return "", fmt.Errorf("AI 服务流式响应没有内容")
	}
	return out.String(), nil
}

func contentText(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text, text != ""
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var out strings.Builder
		for _, part := range parts {
			out.WriteString(part.Text)
		}
		return out.String(), out.Len() > 0
	}
	return "", false
}

func providerHTTPFailure(resp *http.Response) (string, string) {
	detail := readProviderError(resp.Body)
	if detail != "" {
		detail = "（" + detail + "）"
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "AI 服务认证失败" + detail + "。", "请检查 AI API Key 是否有效、是否已过期，以及该 Key 是否有模型调用权限。"
	case http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity:
		return fmt.Sprintf("AI 请求参数无效（HTTP %d）", resp.StatusCode) + detail + "。", "请检查 Endpoint 路径、模型名称和 OpenAI 兼容参数。"
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return fmt.Sprintf("AI 服务处理超时（HTTP %d）", resp.StatusCode) + detail + "。", "AI 服务或中间网关处理超时，请稍后重试或调高网关超时时间。"
	case http.StatusTooManyRequests:
		return "AI 服务请求频率受限（HTTP 429）" + detail + "。", "当前调用额度或并发数已达到限制，请稍后重试。"
	default:
		if resp.StatusCode >= 500 {
			return fmt.Sprintf("AI 服务暂时不可用（HTTP %d）", resp.StatusCode) + detail + "。", "AI 服务或其上游发生故障，请稍后重试并检查服务日志。"
		}
		return fmt.Sprintf("AI 服务返回错误（HTTP %d）", resp.StatusCode) + detail + "。", "请检查 AI 服务配置并稍后重试。"
	}
}

func readProviderError(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, 16*1024))
	if err != nil || len(raw) == 0 {
		return ""
	}
	var payload struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &payload) != nil || len(payload.Error) == 0 {
		return strings.TrimSpace(string(raw))
	}
	var message string
	if json.Unmarshal(payload.Error, &message) == nil {
		return strings.TrimSpace(message)
	}
	var detail struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(payload.Error, &detail) == nil {
		if detail.Message != "" {
			return strings.TrimSpace(detail.Message)
		}
		return strings.TrimSpace(detail.Code)
	}
	return strings.TrimSpace(string(payload.Error))
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
