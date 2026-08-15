// Package webui embeds the built Vue SPA and serves it alongside the API in
// a single binary. The dist/ directory is a placeholder checked into the
// repo so `go build`/`go run .` succeed outside Docker; the production image
// overwrites it with the real `web/dist` build output before compiling.
package webui

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var distFS embed.FS

// Handler serves the embedded SPA, falling back to index.html for any path
// that isn't a static asset so Vue Router's history mode works on refresh.
// Unmatched /api/ or /health/ paths get a JSON 404 instead of the SPA shell,
// since those never legitimately reach index.html.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isUnmatchedAPIPath(r.URL.Path) {
			writeNotFound(w)
			return
		}
		trimmed := strings.TrimPrefix(r.URL.Path, "/")
		if trimmed == "" {
			trimmed = "."
		}
		if _, err := fs.Stat(sub, trimmed); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

func isUnmatchedAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/health/")
}

func writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": "not_found", "message": "resource not found"},
	})
}
