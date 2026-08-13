// Package httputil provides shared HTTP response helpers used across all
// handler packages. Centralising JSON responses ensures a consistent error
// envelope format throughout the API.
package httputil

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes v as JSON and writes it with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a standard API error envelope:
//
//	{"error": {"code": "<code>", "message": "<message>"}}
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
