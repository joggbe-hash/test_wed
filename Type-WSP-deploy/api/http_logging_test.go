package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerRecordsRouteStatusAndRequestID(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		if requestIDFromContext(r.Context()) == "" {
			t.Error("request context does not contain request ID")
		}
		writeJSON(w, http.StatusCreated, M{"ok": true})
	})
	response := httptest.NewRecorder()
	requestLogger(logger, mux).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items/42", nil))
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if response.Header().Get(requestIDHeader) == "" {
		t.Fatal("response does not contain X-Request-ID")
	}
	for _, expected := range []string{`"method":"GET"`, `"route":"GET /items/{id}"`, `"status":201`, `"request_id":"`} {
		if !strings.Contains(logs.String(), expected) {
			t.Fatalf("log %q does not contain %q", logs.String(), expected)
		}
	}
}
