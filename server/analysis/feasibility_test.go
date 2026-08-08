package analysis

import "testing"

func TestNormalizeResultCalculatesWeightedScoreAndProcess(t *testing.T) {
	result, err := normalizeResult(`{"summary":"材料基本完整","dimensions":[{"key":"market","score":80,"confidence":70,"reasoning":"有访谈数据","evidence":["20份访谈"],"gaps":["还缺复购率"]}],"next_actions":["做付费测试"]}`, 7, documentInput{Filename: "plan.txt", MimeType: "text/plain", Text: "内容", Characters: 2})
	if err != nil {
		t.Fatalf("normalizeResult() error = %v", err)
	}
	if len(result.Dimensions) != len(dimensionDefinitions) {
		t.Fatalf("got %d dimensions, want %d", len(result.Dimensions), len(dimensionDefinitions))
	}
	if result.OverallScore <= 0 || result.OverallScore >= 100 {
		t.Fatalf("unexpected weighted score %.1f", result.OverallScore)
	}
	if len(result.AnalysisProcess) != len(dimensionDefinitions) {
		t.Fatalf("got %d process steps, want %d", len(result.AnalysisProcess), len(dimensionDefinitions))
	}
	if result.Dimensions[1].Key != "market" || result.Dimensions[1].Score != 80 {
		t.Fatalf("market dimension was not normalized: %#v", result.Dimensions[1])
	}
	if len(result.NextActions) != 1 || result.NextActions[0] != "做付费测试" {
		t.Fatalf("next actions were not preserved: %#v", result.NextActions)
	}
}

func TestNormalizeResultAcceptsCodeFenceAndClampsScores(t *testing.T) {
	result, err := normalizeResult("```json\n{\"overall_score\":120,\"verdict\":\"可行\",\"dimensions\":[{\"key\":\"risk\",\"score\":-4,\"confidence\":101}]}\n```", 2, documentInput{})
	if err != nil {
		t.Fatalf("normalizeResult() error = %v", err)
	}
	if result.OverallScore != 0 || result.Dimensions[len(result.Dimensions)-1].Score != 0 || result.Dimensions[len(result.Dimensions)-1].Confidence != 100 {
		t.Fatalf("scores were not clamped: overall=%v risk=%#v", result.OverallScore, result.Dimensions[len(result.Dimensions)-1])
	}
}

func TestNormalizeDimensionAcceptsChineseNames(t *testing.T) {
	dimension, ok := normalizeDimension(map[string]any{"name": "商业模式与财务", "score": 66})
	if !ok || dimension.Key != "business_model" || dimension.Weight != 15 {
		t.Fatalf("Chinese dimension name was not mapped: %#v, ok=%v", dimension, ok)
	}
}
