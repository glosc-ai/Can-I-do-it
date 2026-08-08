package analysis

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestProviderRequestFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: context.DeadlineExceeded, want: "响应超时"},
		{name: "closed before headers", err: io.EOF, want: "EOF"},
		{name: "network", err: errors.New("network unavailable"), want: "无法连接"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, summary := providerRequestFailure(tt.err)
			if !strings.Contains(message, tt.want) {
				t.Fatalf("message = %q, want it to contain %q", message, tt.want)
			}
			if summary == "" {
				t.Fatal("summary should explain the next diagnostic step")
			}
		})
	}
}

func TestProviderHTTPFailure(t *testing.T) {
	tests := []struct {
		status int
		body   string
		want   string
	}{
		{status: http.StatusUnauthorized, body: `{"error":{"message":"invalid token"}}`, want: "认证失败"},
		{status: http.StatusTooManyRequests, body: `{"error":{"message":"quota exhausted"}}`, want: "频率受限"},
		{status: http.StatusBadGateway, body: `{"error":"upstream unavailable"}`, want: "暂时不可用"},
		{status: http.StatusBadRequest, body: `{"error":{"message":"unknown model"}}`, want: "请求参数无效"},
	}
	for _, tt := range tests {
		resp := &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body))}
		message, summary := providerHTTPFailure(resp)
		if !strings.Contains(message, tt.want) {
			t.Errorf("status %d message = %q, want it to contain %q", tt.status, message, tt.want)
		}
		if summary == "" {
			t.Errorf("status %d summary should not be empty", tt.status)
		}
	}
}
