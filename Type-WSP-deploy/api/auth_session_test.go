package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthSessionRouteRequiresAuthentication(t *testing.T) {
	loaderCalled := false
	mux := newMuxWithSessionLoader(func(context.Context, string) (*User, error) {
		loaderCalled = true
		return nil, errors.New("loader should not be called without a cookie")
	})
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/auth/session", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if loaderCalled {
		t.Fatal("session loader was called without a session cookie")
	}
}

func TestAuthSessionRouteReturnsAuthenticatedUser(t *testing.T) {
	user := &User{ID: 7, Username: "demo", Email: "demo@example.com"}
	mux := newMuxWithSessionLoader(func(_ context.Context, signed string) (*User, error) {
		if signed != "signed-session" {
			t.Fatalf("signed session = %q, want %q", signed, "signed-session")
		}
		return user, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "signed-session"})
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body struct {
		User User `json:"user"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.User != *user {
		t.Fatalf("user = %#v, want %#v", body.User, *user)
	}
}

func TestAuthSessionRouteRejectsInvalidSession(t *testing.T) {
	mux := newMuxWithSessionLoader(func(context.Context, string) (*User, error) {
		return nil, fmt.Errorf("%w: expired", errInvalidSession)
	})
	response := serveSessionRequest(mux)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestAuthSessionRouteReportsSessionStoreFailure(t *testing.T) {
	mux := newMuxWithSessionLoader(func(context.Context, string) (*User, error) {
		return nil, errors.New("redis unavailable")
	})
	response := serveSessionRequest(mux)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestVerificationCodeResponseRequiresExplicitDevelopmentGate(t *testing.T) {
	previous := debugVerificationCode
	t.Cleanup(func() {
		debugVerificationCode = previous
	})

	debugVerificationCode = false
	if _, exposed := verificationCodeResponse("123456")["debug_code"]; exposed {
		t.Fatal("debug_code was exposed with the gate disabled")
	}

	debugVerificationCode = true
	if code := verificationCodeResponse("123456")["debug_code"]; code != "123456" {
		t.Fatalf("debug_code = %#v, want %q", code, "123456")
	}
}

func TestLoadConfigDebugVerificationCodeRequiresDevelopmentAndOptIn(t *testing.T) {
	setRequiredProductionEnv(t)
	t.Setenv("EXPOSE_VERIFICATION_CODE", "true")
	productionConfig, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig production: %v", err)
	}
	if productionConfig.DebugVerificationCode {
		t.Fatal("production config exposed verification codes")
	}

	t.Setenv("APP_ENV", "development")
	t.Setenv("EXPOSE_VERIFICATION_CODE", "false")
	developmentConfig, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig development: %v", err)
	}
	if developmentConfig.DebugVerificationCode {
		t.Fatal("development config exposed verification codes without opt-in")
	}

	t.Setenv("EXPOSE_VERIFICATION_CODE", "true")
	developmentConfig, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig development opt-in: %v", err)
	}
	if !developmentConfig.DebugVerificationCode {
		t.Fatal("development config did not honor explicit verification-code opt-in")
	}
}

func TestGenerateVerificationCodeFormat(t *testing.T) {
	for range 32 {
		code, err := generateVerificationCode()
		if err != nil {
			t.Fatalf("generate verification code: %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("code length = %d, want 6", len(code))
		}
		for _, digit := range code {
			if digit < '0' || digit > '9' {
				t.Fatalf("code %q contains non-digit %q", code, digit)
			}
		}
	}
}

func serveSessionRequest(mux http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "signed-session"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	return response
}
