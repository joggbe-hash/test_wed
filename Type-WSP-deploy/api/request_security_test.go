package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireBrowserRequestRejectsSimpleCrossSiteRequest(t *testing.T) {
	called := false
	handler := requireBrowserRequest(func(http.ResponseWriter, *http.Request) {
		called = true
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	handler(response, req)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if called {
		t.Fatal("cross-site request reached the protected handler")
	}
}

func TestRequireBrowserRequestAllowsFrontendHeader(t *testing.T) {
	called := false
	handler := requireBrowserRequest(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set(browserRequestHeader, "1")
	response := httptest.NewRecorder()

	handler(response, req)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("trusted frontend request did not reach the protected handler")
	}
}

func TestMutationRoutesRejectRequestsWithoutBrowserHeader(t *testing.T) {
	mux := newMuxWithSessionLoader(func(_ context.Context, _ string) (*User, error) {
		t.Fatal("session loader must not run before browser-request validation")
		return nil, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/api/posts", strings.NewReader("content=forged"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session", Value: "signed-session"})
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestLoginVerificationRouteRejectsRequestsWithoutBrowserHeader(t *testing.T) {
	mux := newMuxWithSessionLoader(func(_ context.Context, _ string) (*User, error) {
		t.Fatal("session loader must not run for login verification")
		return nil, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login/verify", strings.NewReader(`{"email":"user@example.com","code":"123456","challenge_id":"2cd53940-fc0d-4972-921b-086061dde6e5"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestLoginOwnershipVerificationRouteRejectsRequestsWithoutBrowserHeader(t *testing.T) {
	mux := newMuxWithSessionLoader(func(_ context.Context, _ string) (*User, error) {
		t.Fatal("session loader must not run for login ownership verification")
		return nil, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login/ownership/verify", strings.NewReader(`{"email":"user@example.com","code":"ABCDEFGHJKLMNPQ2","challenge_id":"2cd53940-fc0d-4972-921b-086061dde6e5"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestReadJSONRequiresApplicationJSONAndSingleValue(t *testing.T) {
	for _, test := range []struct {
		name        string
		contentType string
		body        string
		wantError   bool
	}{
		{name: "valid JSON", contentType: "application/json", body: `{"value":1}`},
		{name: "text plain", contentType: "text/plain", body: `{"value":1}`, wantError: true},
		{name: "trailing JSON", contentType: "application/json", body: `{"value":1}{"value":2}`, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(test.body))
			req.Header.Set("Content-Type", test.contentType)
			var payload struct {
				Value int `json:"value"`
			}
			err := readJSON(req, &payload)
			if (err != nil) != test.wantError {
				t.Fatalf("readJSON error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}

func TestRequestClientIdentityUsesProxyAddressAndSafeFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "172.18.0.2:43210"
	req.Header.Set("X-Real-IP", "203.0.113.7")
	if got := requestClientIdentity(req); got != "203.0.113.7" {
		t.Fatalf("requestClientIdentity = %q, want proxy-provided client", got)
	}

	req.Header.Set("X-Real-IP", "not-an-ip")
	if got := requestClientIdentity(req); got != "172.18.0.2" {
		t.Fatalf("requestClientIdentity fallback = %q, want remote address", got)
	}
}
