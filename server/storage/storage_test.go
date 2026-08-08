package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocalStorageRoundTrip(t *testing.T) {
	store := New(nil, "postgres", "01234567890123456789012345678901", t.TempDir(), 1024, R2Config{})
	ctx := context.Background()
	content := []byte("asset content")
	if err := store.Put(ctx, "users/1/upload/example.txt", bytes.NewReader(content), int64(len(content)), "text/plain"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	body, size, err := store.Open(ctx, "users/1/upload/example.txt")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(content) != string(got) {
		t.Fatalf("content = %q, want %q", got, content)
	}
	if size != int64(len(content)) {
		t.Fatalf("size = %d, want %d", size, len(content))
	}
	if err := store.Delete(ctx, "users/1/upload/example.txt"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	url, err := store.URL(ctx, "users/1/upload/example.txt", 0)
	if err != nil || url != "" {
		t.Fatalf("URL() = %q, %v; want empty URL and nil error", url, err)
	}
}

func TestEnabledR2RequiresCompleteConfiguration(t *testing.T) {
	store := New(nil, "postgres", "01234567890123456789012345678901", t.TempDir(), 1024, R2Config{Enabled: true, Endpoint: "https://example.com", Bucket: "bucket"})
	err := store.Put(context.Background(), "key", bytes.NewReader([]byte("x")), 1, "text/plain")
	if err != ErrR2NotConfigured {
		t.Fatalf("Put() error = %v, want ErrR2NotConfigured", err)
	}
}
