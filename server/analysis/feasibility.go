package analysis

import (
	"archive/zip"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/gloscai/template-go-vue3-docker/server/storage"
)

// The prompt is kept as an embedded skill so every worker uses the same
// rubric as the project documentation and it can be reviewed like any other
// product requirement.
//
//go:embed feasibility_skill.md
var feasibilitySkill string

type DimensionScore struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Score      float64  `json:"score"`
	Weight     float64  `json:"weight"`
	Confidence float64  `json:"confidence"`
	Reasoning  string   `json:"reasoning"`
	Evidence   []string `json:"evidence"`
	Gaps       []string `json:"gaps"`
}

type AnalysisStep struct {
	Step      string   `json:"step"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Questions []string `json:"questions"`
}

type FeasibilityResult struct {
	PlanID          int64            `json:"plan_id"`
	OverallScore    float64          `json:"overall_score"`
	Verdict         string           `json:"verdict"`
	Summary         string           `json:"summary"`
	Dimensions      []DimensionScore `json:"dimensions"`
	AnalysisProcess []AnalysisStep   `json:"analysis_process"`
	NextActions     []string         `json:"next_actions"`
	Source          SourceInfo       `json:"source"`
}

type SourceInfo struct {
	Filename      string `json:"filename"`
	MimeType      string `json:"mime_type"`
	Characters    int    `json:"characters"`
	ImageAnalyzed bool   `json:"image_analyzed"`
}

type documentInput struct {
	Text         string
	ImageDataURL string
	MimeType     string
	Filename     string
	Characters   int
}

type dimensionDefinition struct {
	Key    string
	Name   string
	Weight float64
}

var dimensionDefinitions = []dimensionDefinition{
	{Key: "problem", Name: "问题与需求", Weight: 15},
	{Key: "market", Name: "市场空间", Weight: 15},
	{Key: "solution", Name: "解决方案与技术", Weight: 12},
	{Key: "competition", Name: "竞争与差异化", Weight: 12},
	{Key: "business_model", Name: "商业模式与财务", Weight: 15},
	{Key: "go_to_market", Name: "获客与运营", Weight: 10},
	{Key: "legal", Name: "法规与合规", Weight: 8},
	{Key: "team", Name: "团队与执行", Weight: 8},
	{Key: "risk", Name: "风险与验证", Weight: 5},
}

var dimensionAliases = map[string][]string{
	"problem":        {"问题", "需求", "问题与需求"},
	"market":         {"市场", "市场空间", "市场需求"},
	"solution":       {"方案", "技术", "解决方案", "解决方案与技术"},
	"competition":    {"竞争", "竞品", "竞争与差异化"},
	"business_model": {"商业模式", "财务", "商业模式与财务"},
	"go_to_market":   {"获客", "运营", "获客与运营"},
	"legal":          {"法规", "合规", "法规与合规"},
	"team":           {"团队", "执行", "团队与执行"},
	"risk":           {"风险", "验证", "风险与验证"},
}

func skillPrompt() string { return feasibilitySkill }

func readDocument(ctx context.Context, store *storage.Service, key, filename, mimeType string, max int64) (documentInput, error) {
	if store == nil || strings.TrimSpace(key) == "" {
		return documentInput{}, fmt.Errorf("uploaded source is unavailable")
	}
	body, _, err := store.Open(ctx, key)
	if err != nil {
		return documentInput{}, fmt.Errorf("opening uploaded source: %w", err)
	}
	defer body.Close()
	limit := max
	if limit <= 0 || limit > 20*1024*1024 {
		limit = 20 * 1024 * 1024
	}
	raw, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return documentInput{}, fmt.Errorf("reading uploaded source: %w", err)
	}
	if int64(len(raw)) > limit {
		return documentInput{}, fmt.Errorf("uploaded source exceeds analysis limit")
	}

	in := documentInput{MimeType: mimeType, Filename: filename}
	if isImage(filename, mimeType) {
		if mimeType == "" || mimeType == "application/octet-stream" {
			mimeType = sniffImageType(raw)
		}
		in.MimeType = mimeType
		in.ImageDataURL = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw)
		return in, nil
	}
	in.Text = extractText(raw, filename, mimeType)
	in.Characters = len([]rune(in.Text))
	if strings.TrimSpace(in.Text) == "" {
		in.Text = "文件未能在本地提取出可搜索文本。请根据文件元数据和附件内容标记信息缺口，并将需要 OCR/人工核验的字段列入 gaps。"
	}
	return in, nil
}

func isImage(filename, mimeType string) bool {
	return strings.HasPrefix(strings.ToLower(mimeType), "image/") || regexp.MustCompile(`(?i)\.(png|jpe?g|webp|gif|bmp|tiff?)$`).MatchString(filename)
}

func sniffImageType(raw []byte) string {
	if len(raw) >= 8 && bytes.Equal(raw[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return "image/png"
	}
	if len(raw) >= 3 && bytes.Equal(raw[:3], []byte{255, 216, 255}) {
		return "image/jpeg"
	}
	if len(raw) >= 12 && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "WEBP" {
		return "image/webp"
	}
	return "image/jpeg"
}

func extractText(raw []byte, filename, mimeType string) string {
	text := strings.ToLower(filename + " " + mimeType)
	switch {
	case strings.HasSuffix(text, ".docx") || strings.Contains(text, "wordprocessingml"):
		return extractDocx(raw)
	case strings.HasSuffix(text, ".pdf") || strings.Contains(text, "application/pdf"):
		return extractPDF(raw)
	case strings.HasSuffix(text, ".doc"):
		return extractBinaryText(raw)
	default:
		return strings.ToValidUTF8(string(raw), "")
	}
}

func extractBinaryText(raw []byte) string {
	var chunks []string
	for _, part := range regexp.MustCompile(`[\x20-\x7e\p{Han}]{4,}`).FindAll(raw, -1) {
		value := strings.TrimSpace(string(part))
		if value != "" && !strings.Contains(value, "Microsoft") {
			chunks = append(chunks, value)
		}
	}
	return strings.Join(chunks, " ")
}

func extractDocx(raw []byte) string {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return ""
	}
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ""
		}
		data, err := io.ReadAll(io.LimitReader(rc, 5*1024*1024+1))
		_ = rc.Close()
		if err != nil {
			return ""
		}
		if len(data) > 5*1024*1024 {
			return "文档文本超过单次分析的 5 MB 提取上限。"
		}
		return xmlText(data)
	}
	return ""
}

func xmlText(data []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var b strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return html.UnescapeString(stripTags(string(data)))
		}
		switch t := token.(type) {
		case xml.CharData:
			value := strings.TrimSpace(string(t))
			if value != "" {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(value)
			}
		case xml.EndElement:
			if t.Name.Local == "p" || t.Name.Local == "br" {
				b.WriteByte('\n')
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func stripTags(value string) string {
	value = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func extractPDF(raw []byte) string {
	// This intentionally avoids pretending to OCR binary PDF streams. It
	// recovers uncompressed text operators and leaves a clear evidence gap for
	// scanned PDFs, which can still be sent to an image-capable provider later.
	var chunks []string
	for _, part := range regexp.MustCompile(`[\x20-\x7e\p{Han}]{4,}`).FindAll(raw, -1) {
		value := strings.TrimSpace(string(part))
		value = strings.Trim(value, "()[]<>{}")
		if strings.ContainsAny(value, "\\") {
			value = strings.ReplaceAll(value, "\\n", " ")
		}
		if len(value) > 3 && !strings.HasPrefix(value, "/") {
			chunks = append(chunks, value)
		}
	}
	return strings.Join(chunks, " ")
}

func normalizeResult(raw string, planID int64, source documentInput) (FeasibilityResult, error) {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		if idx := strings.IndexByte(cleaned, '\n'); idx >= 0 {
			cleaned = cleaned[idx+1:]
		}
		cleaned = strings.TrimSuffix(strings.TrimSpace(cleaned), "```")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		// Some compatible providers add a short preamble despite the JSON-only
		// instruction. Recover the first complete object without accepting an
		// arbitrary prose response as a structured report.
		start, end := strings.IndexByte(cleaned, '{'), strings.LastIndexByte(cleaned, '}')
		if start < 0 || end <= start || json.Unmarshal([]byte(cleaned[start:end+1]), &payload) != nil {
			return FeasibilityResult{}, fmt.Errorf("AI response was not valid JSON: %w", err)
		}
	}
	out := FeasibilityResult{PlanID: planID, Source: SourceInfo{Filename: source.Filename, MimeType: source.MimeType, Characters: source.Characters, ImageAnalyzed: source.ImageDataURL != ""}}
	out.Summary = firstString(payload, "summary", "conclusion", "结论")
	out.Verdict = firstString(payload, "verdict", "feasibility", "可行性")
	out.NextActions = stringSlice(firstValue(payload, "next_actions", "next_steps", "recommendations", "建议"))

	items := payload["dimensions"]
	if items == nil {
		items = payload["scores"]
	}
	if list, ok := items.([]any); ok {
		for _, rawItem := range list {
			if item, ok := rawItem.(map[string]any); ok {
				if dimension, ok := normalizeDimension(item); ok {
					out.Dimensions = append(out.Dimensions, dimension)
				}
			}
		}
	} else if byName, ok := items.(map[string]any); ok {
		// Some models serialize the nine dimensions as an object keyed by
		// dimension name instead of the array required by the prompt. Normalize
		// that shape too; otherwise the result looks successful but all scores
		// silently become zero.
		for key, rawItem := range byName {
			if item, ok := rawItem.(map[string]any); ok {
				if _, exists := item["key"]; !exists {
					item["key"] = key
				}
				if dimension, ok := normalizeDimension(item); ok {
					out.Dimensions = append(out.Dimensions, dimension)
				}
			}
		}
	}
	byKey := make(map[string]DimensionScore, len(out.Dimensions))
	for _, dimension := range out.Dimensions {
		byKey[dimension.Key] = dimension
	}
	for _, definition := range dimensionDefinitions {
		if dimension, ok := byKey[definition.Key]; ok {
			dimension.Weight = definition.Weight
			byKey[definition.Key] = dimension
		} else {
			byKey[definition.Key] = DimensionScore{Key: definition.Key, Name: definition.Name, Weight: definition.Weight, Evidence: []string{}, Gaps: []string{}, Reasoning: "AI 未返回该维度的可核验结论。"}
		}
	}
	out.Dimensions = out.Dimensions[:0]
	for _, definition := range dimensionDefinitions {
		out.Dimensions = append(out.Dimensions, byKey[definition.Key])
	}
	// The server owns the final score so providers cannot bypass the published
	// weights or produce a non-reproducible result with different arithmetic.
	out.OverallScore = weightedScore(out.Dimensions)
	if out.Verdict == "" {
		out.Verdict = verdictFor(out.OverallScore)
	}
	if out.Summary == "" {
		out.Summary = fmt.Sprintf("项目综合可行性得分 %.1f/100，%s。", out.OverallScore, out.Verdict)
	}
	out.AnalysisProcess = normalizeProcess(payload["analysis_process"], out.Dimensions)
	if len(out.AnalysisProcess) == 0 {
		out.AnalysisProcess = normalizeProcess(payload["process"], out.Dimensions)
	}
	return out, nil
}

func normalizeDimension(item map[string]any) (DimensionScore, bool) {
	key := strings.ToLower(strings.TrimSpace(firstString(item, "key", "dimension", "id")))
	nameValue := strings.ToLower(strings.TrimSpace(firstString(item, "name", "title", "维度")))
	for _, definition := range dimensionDefinitions {
		if key == definition.Key || nameValue == strings.ToLower(definition.Name) || aliasMatch(key, definition.Key) || aliasMatch(nameValue, definition.Key) {
			key = definition.Key
			break
		}
	}
	if key == "" {
		return DimensionScore{}, false
	}
	name, weight := key, 0.0
	for _, definition := range dimensionDefinitions {
		if definition.Key == key {
			name, weight = definition.Name, definition.Weight
			break
		}
	}
	score, _ := number(firstValue(item, "score", "分数"))
	confidence, _ := number(firstValue(item, "confidence", "置信度"))
	return DimensionScore{Key: key, Name: name, Score: clamp(score), Weight: weight, Confidence: clamp(confidence), Reasoning: firstString(item, "reasoning", "analysis", "评估", "结论"), Evidence: stringSlice(firstValue(item, "evidence", "证据")), Gaps: stringSlice(firstValue(item, "gaps", "missing", "待验证", "缺口"))}, true
}

func aliasMatch(value, key string) bool {
	if value == "" {
		return false
	}
	for _, alias := range dimensionAliases[key] {
		if value == strings.ToLower(alias) {
			return true
		}
	}
	return false
}

func normalizeProcess(value any, dimensions []DimensionScore) []AnalysisStep {
	var out []AnalysisStep
	if list, ok := value.([]any); ok {
		for _, rawItem := range list {
			if item, ok := rawItem.(map[string]any); ok {
				out = append(out, AnalysisStep{Step: firstString(item, "step", "key"), Title: firstString(item, "title", "name"), Status: firstString(item, "status"), Summary: firstString(item, "summary", "content", "reasoning"), Questions: stringSlice(firstValue(item, "questions", "gaps"))})
			}
		}
	}
	if len(out) == 0 {
		for _, dimension := range dimensions {
			out = append(out, AnalysisStep{Step: dimension.Key, Title: dimension.Name, Status: "completed", Summary: dimension.Reasoning, Questions: dimension.Gaps})
		}
	}
	return out
}

func firstValue(payload map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}
func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
			if value != nil {
				return strings.TrimSpace(fmt.Sprint(value))
			}
		}
	}
	return ""
}
func stringSlice(value any) []string {
	var out []string
	if list, ok := value.([]any); ok {
		for _, item := range list {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
	} else if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
		out = []string{strings.TrimSpace(text)}
	}
	return out
}
func number(value any) (float64, bool) {
	switch n := value.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case json.Number:
		f, e := n.Float64()
		return f, e == nil
	case string:
		f, e := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, e == nil
	}
	return 0, false
}
func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
func weightedScore(dimensions []DimensionScore) float64 {
	var total, weights float64
	for _, dimension := range dimensions {
		total += clamp(dimension.Score) * dimension.Weight
		weights += dimension.Weight
	}
	if weights == 0 {
		return 0
	}
	return float64(int(total/weights*10+0.5)) / 10
}
func verdictFor(score float64) string {
	switch {
	case score >= 80:
		return "可行"
	case score >= 60:
		return "有条件可行"
	case score >= 40:
		return "风险较高"
	default:
		return "当前不可行"
	}
}
