package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestStoredSessionPayloadOmitsEmail(t *testing.T) {
	data, err := marshalStoredSession(&User{
		ID:       7,
		Username: "owner",
		Email:    "owner@example.com",
	}, false)
	if err != nil {
		t.Fatalf("marshal stored session: %v", err)
	}
	if strings.Contains(string(data), "owner@example.com") || strings.Contains(string(data), "email") {
		t.Fatalf("stored session exposed email: %s", data)
	}
}

func TestStoredSessionTracksRememberChoice(t *testing.T) {
	data, err := marshalStoredSession(&User{ID: 7, Username: "owner"}, true)
	if err != nil {
		t.Fatalf("marshal stored session: %v", err)
	}

	var session storedSession
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatalf("decode stored session: %v", err)
	}
	if !session.Persistent {
		t.Fatal("remembered session was not marked persistent")
	}
}

func TestLoadSessionRejectsAnOlderUserGeneration(t *testing.T) {
	previousSecret := sessionSecret
	sessionSecret = []byte("test-session-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })

	data, err := json.Marshal(storedSession{
		User:       User{ID: 7, Username: "owner", Email: "owner@example.com"},
		Generation: 2,
	})
	if err != nil {
		t.Fatalf("marshal stored session: %v", err)
	}

	_, err = loadSessionWithStores(
		context.Background(),
		signSID("session-id"),
		func(context.Context, string) ([]byte, error) { return data, nil },
		func(context.Context, string) (int64, error) { return 3, nil },
	)
	if !errors.Is(err, errInvalidSession) {
		t.Fatalf("LoadSession error = %v, want invalid session", err)
	}
}

func TestLoadSessionAcceptsTheCurrentUserGeneration(t *testing.T) {
	previousSecret := sessionSecret
	sessionSecret = []byte("test-session-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })

	data, err := json.Marshal(storedSession{
		User:       User{ID: 7, Username: "owner", Email: "owner@example.com"},
		Generation: 3,
	})
	if err != nil {
		t.Fatalf("marshal stored session: %v", err)
	}

	user, err := loadSessionWithStores(
		context.Background(),
		signSID("session-id"),
		func(context.Context, string) ([]byte, error) { return data, nil },
		func(context.Context, string) (int64, error) { return 3, nil },
	)
	if err != nil || user.ID != 7 {
		t.Fatalf("LoadSession user = %#v, error = %v", user, err)
	}
}

func TestLoadSessionRestoresRememberChoice(t *testing.T) {
	previousSecret := sessionSecret
	sessionSecret = []byte("test-session-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })

	data, err := json.Marshal(storedSession{
		User:       User{ID: 7, Username: "owner"},
		Generation: 0,
		Persistent: true,
	})
	if err != nil {
		t.Fatalf("marshal stored session: %v", err)
	}

	user, err := loadSessionWithStores(
		context.Background(),
		signSID("session-id"),
		func(context.Context, string) ([]byte, error) { return data, nil },
		func(context.Context, string) (int64, error) { return 0, nil },
	)
	if err != nil {
		t.Fatalf("LoadSession error = %v", err)
	}
	if !user.sessionPersistent {
		t.Fatal("remember choice was not restored from the stored session")
	}
}

func TestRememberedSessionRefreshesThirtyDayServerIdleDeadline(t *testing.T) {
	previousSecret := sessionSecret
	sessionSecret = []byte("test-session-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })

	called := false
	err := refreshPersistentSessionWithStore(
		context.Background(),
		signSID("session-id"),
		&User{ID: 7, sessionPersistent: true},
		func(_ context.Context, key string, ttl time.Duration) (bool, error) {
			called = true
			if key != sessionPrefix+"session-id" {
				t.Fatalf("session key = %q", key)
			}
			if ttl != rememberSessionTTL {
				t.Fatalf("idle TTL = %s, want %s", ttl, rememberSessionTTL)
			}
			return true, nil
		},
	)
	if err != nil {
		t.Fatalf("refresh remembered session: %v", err)
	}
	if !called {
		t.Fatal("remembered session idle deadline was not refreshed")
	}
}

func TestBrowserSessionDoesNotRefreshServerIdleDeadline(t *testing.T) {
	called := false
	err := refreshPersistentSessionWithStore(
		context.Background(),
		"signed-session",
		&User{ID: 7},
		func(context.Context, string, time.Duration) (bool, error) {
			called = true
			return true, nil
		},
	)
	if err != nil {
		t.Fatalf("refresh standard session: %v", err)
	}
	if called {
		t.Fatal("standard browser session unexpectedly refreshed its idle deadline")
	}
}

func TestRememberedSessionRejectsRefreshAfterExpiryRace(t *testing.T) {
	previousSecret := sessionSecret
	sessionSecret = []byte("test-session-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })

	err := refreshPersistentSessionWithStore(
		context.Background(),
		signSID("session-id"),
		&User{ID: 7, sessionPersistent: true},
		func(context.Context, string, time.Duration) (bool, error) {
			return false, nil
		},
	)
	if !errors.Is(err, errInvalidSession) {
		t.Fatalf("refresh error = %v, want invalid session", err)
	}
}

func TestRememberedSessionReportsRefreshStoreFailure(t *testing.T) {
	previousSecret := sessionSecret
	sessionSecret = []byte("test-session-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })
	wantErr := errors.New("redis unavailable")

	err := refreshPersistentSessionWithStore(
		context.Background(),
		signSID("session-id"),
		&User{ID: 7, sessionPersistent: true},
		func(context.Context, string, time.Duration) (bool, error) {
			return false, wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("refresh error = %v, want %v", err, wantErr)
	}
}
