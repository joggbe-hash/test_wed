package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessEndpoint(t *testing.T) {
	mux := newMuxWithDependencies(func(context.Context, string) (*User, error) { return nil, nil }, func(context.Context) error { return nil })
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestReadinessEndpointReportsDependencyFailure(t *testing.T) {
	mux := newMuxWithDependencies(func(context.Context, string) (*User, error) { return nil, nil }, func(context.Context) error { return errors.New("redis unavailable") })
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if body := response.Body.String(); body != "{\"status\":\"not_ready\"}\n" {
		t.Fatalf("body = %q", body)
	}
}
