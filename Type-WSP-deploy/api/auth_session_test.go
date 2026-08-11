package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
		User struct {
			ID       int     `json:"id"`
			Username string  `json:"username"`
			Email    *string `json:"email"`
		} `json:"user"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.User.ID != user.ID || body.User.Username != user.Username {
		t.Fatalf("user = %#v, want id=%d username=%q", body.User, user.ID, user.Username)
	}
	if body.User.Email != nil || strings.Contains(response.Body.String(), user.Email) {
		t.Fatalf("session response exposed email: %s", response.Body.String())
	}
}

func TestSessionCookiePersistenceMatchesRememberChoice(t *testing.T) {
	standard := sessionCookieForLogin("signed-session", time.Hour, false)
	if standard.MaxAge != 0 {
		t.Fatalf("standard login MaxAge = %d, want session cookie", standard.MaxAge)
	}

	remembered := sessionCookieForLogin("signed-session", 30*24*time.Hour, true)
	if remembered.MaxAge != int((30 * 24 * time.Hour).Seconds()) {
		t.Fatalf("remembered login MaxAge = %d", remembered.MaxAge)
	}
}

func TestAuthenticatedRememberedSessionRefreshesBrowserIdleDeadline(t *testing.T) {
	user := &User{ID: 7, Username: "demo", sessionPersistent: true}
	mux := newMuxWithSessionLoader(func(context.Context, string) (*User, error) {
		return user, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "signed-session"})
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("refreshed cookies = %#v, want one session cookie", cookies)
	}
	if cookies[0].Name != "session" || cookies[0].Value != "signed-session" {
		t.Fatalf("refreshed cookie = %#v", cookies[0])
	}
	if cookies[0].MaxAge != int(rememberSessionTTL.Seconds()) {
		t.Fatalf("refreshed MaxAge = %d, want %d", cookies[0].MaxAge, int(rememberSessionTTL.Seconds()))
	}
}

func TestAuthenticatedBrowserSessionDoesNotBecomePersistent(t *testing.T) {
	mux := newMuxWithSessionLoader(func(context.Context, string) (*User, error) {
		return &User{ID: 7, Username: "demo"}, nil
	})
	response := serveSessionRequest(mux)

	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("standard session unexpectedly received a persistent cookie: %#v", cookies)
	}
}

func TestLogoutAllRouteIsNotExposed(t *testing.T) {
	mux := newMuxWithSessionLoader(func(context.Context, string) (*User, error) {
		t.Fatal("removed route should not load a session")
		return nil, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout-all", nil)
	req.Header.Set("X-Type-WSP-Request", "1")
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
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
	if _, exposed := verificationCodeResponse("challenge-id", "123456")["debug_code"]; exposed {
		t.Fatal("debug_code was exposed with the gate disabled")
	}

	debugVerificationCode = true
	if code := verificationCodeResponse("challenge-id", "123456")["debug_code"]; code != "123456" {
		t.Fatalf("debug_code = %#v, want %q", code, "123456")
	}
}

func TestInitAuthUsesSecureCookiesOutsideDevelopment(t *testing.T) {
	previous := sessionCookieSecure
	t.Cleanup(func() {
		sessionCookieSecure = previous
	})

	InitAuth(&Config{AppEnv: "development"})
	if sessionCookieSecure {
		t.Fatal("development session cookie was unexpectedly secure")
	}

	InitAuth(&Config{AppEnv: "production"})
	if !sessionCookieSecure {
		t.Fatal("production session cookie was not secure")
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

func TestDestroySessionPropagatesStoreFailure(t *testing.T) {
	previousSecret := sessionSecret
	sessionSecret = []byte("test-session-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })

	wantErr := errors.New("redis unavailable")
	called := false
	err := destroySessionWithStore(context.Background(), signSID("session-id"), func(_ context.Context, key, channel string) error {
		called = true
		if key != sessionPrefix+"session-id" {
			t.Fatalf("session key = %q", key)
		}
		if channel != sessionRevocationChannel("session-id") {
			t.Fatalf("revocation channel = %q", channel)
		}
		return wantErr
	})
	if !called {
		t.Fatal("session store was not called")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("DestroySession error = %v, want %v", err, wantErr)
	}
}

func TestLogoutReportsSessionStoreFailureAndPreservesCookieForRetry(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "signed-session"})
	response := httptest.NewRecorder()

	handleLogoutWithDestroyer(response, req, func(context.Context, string) error {
		return errors.New("redis unavailable")
	})

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("failed logout replaced the retry credential: %#v", cookies)
	}
}

func TestLogoutClearsCookieAfterSessionStoreSucceeds(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "signed-session"})
	response := httptest.NewRecorder()

	handleLogoutWithDestroyer(response, req, func(context.Context, string) error {
		return nil
	})

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].MaxAge >= 0 {
		t.Fatalf("successful logout did not clear the browser cookie: %#v", cookies)
	}
}

func serveSessionRequest(mux http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "signed-session"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	return response
}
