package httpx

import (
	"encoding/json"
	"net/http"
)

// Error writes the public error envelope without exposing internal failures.
func Error(w http.ResponseWriter, message string, status int) {
	code := map[int]string{400: "invalid_request", 401: "unauthorized", 403: "forbidden",
		404: "not_found", 405: "method_not_allowed", 408: "timeout", 409: "conflict",
		410: "gone", 413: "too_large", 415: "bad_type", 422: "pixel_limit",
		429: "rate_limited", 500: "internal_error", 503: "unavailable"}[status]
	if code == "" {
		code = "request_failed"
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{code, message})
}
