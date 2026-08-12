package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	sessionPrefix           = "sess:"
	sessionRevocationPrefix = "sess:revoked:"
	sessionGenerationPrefix = "sess:generation:"
)

var errInvalidSession = errors.New("invalid session")

var (
	sessionSecret []byte
	sessionTTL    time.Duration
)

var destroySessionScript = redis.NewScript(`
redis.call('DEL', KEYS[1])
redis.call('PUBLISH', KEYS[2], 'revoked')
return 1
`)

var storeSessionScript = redis.NewScript(`
local generation = tonumber(redis.call('GET', KEYS[2]) or '0')
local session = cjson.decode(ARGV[1])
session.generation = generation
redis.call('SET', KEYS[1], cjson.encode(session), 'PX', ARGV[2])
return generation
`)

func InitSession(cfg *Config) {
	sessionSecret = []byte(cfg.SecretKey)
	sessionTTL = time.Duration(cfg.SessionTTL) * time.Second
}

// User 是放進 Redis session 的最小使用者資料，handler 從 request context 取用。
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`

	sessionPersistent bool
}

type storedSession struct {
	User       User  `json:"user"`
	Generation int64 `json:"generation"`
	Persistent bool  `json:"persistent,omitempty"`
}

type contextKey string

const userCtxKey contextKey = "user"

// signSID 用 HMAC 簽 session id，避免前端竄改 cookie 裡的 sid。
func signSID(sid string) string {
	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(sid))
	sig := hex.EncodeToString(mac.Sum(nil))
	return sid + "." + sig
}

func verifySID(signed string) (string, error) {
	parts := strings.SplitN(signed, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("%w: invalid format", errInvalidSession)
	}
	sid, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(sid))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", fmt.Errorf("%w: invalid signature", errInvalidSession)
	}
	return sid, nil
}

// CreateSession 只把 signed sid 放 cookie，完整 user 資料存在 Redis，方便登出與 TTL 控制。
func CreateSession(ctx context.Context, user *User) (string, error) {
	return CreateSessionWithTTL(ctx, user, sessionTTL, false)
}

func CreateSessionWithTTL(ctx context.Context, user *User, ttl time.Duration, persistent bool) (string, error) {
	sid := uuid.New().String()
	data, err := marshalStoredSession(user, persistent)
	if err != nil {
		return "", fmt.Errorf("marshal session user failed: %w", err)
	}
	if err := storeSessionScript.Run(
		ctx,
		rdb,
		[]string{sessionPrefix + sid, sessionGenerationKey(user.ID)},
		data,
		ttl.Milliseconds(),
	).Err(); err != nil {
		return "", fmt.Errorf("save session failed: %w", err)
	}
	return signSID(sid), nil
}

func marshalStoredSession(user *User, persistent bool) ([]byte, error) {
	return json.Marshal(storedSession{User: User{
		ID:       user.ID,
		Username: user.Username,
	}, Persistent: persistent})
}

type sessionValueLoader func(context.Context, string) ([]byte, error)
type sessionGenerationLoader func(context.Context, string) (int64, error)

func loadSessionWithStores(
	ctx context.Context,
	signed string,
	loadValue sessionValueLoader,
	loadGeneration sessionGenerationLoader,
) (*User, error) {
	sid, err := verifySID(signed)
	if err != nil {
		return nil, err
	}
	data, err := loadValue(ctx, sessionPrefix+sid)
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("%w: expired or not found", errInvalidSession)
	}
	if err != nil {
		return nil, fmt.Errorf("load session failed: %w", err)
	}
	var session storedSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("decode session failed: %w", err)
	}
	if session.User.ID == 0 {
		if err := json.Unmarshal(data, &session.User); err != nil || session.User.ID == 0 {
			return nil, fmt.Errorf("%w: invalid session payload", errInvalidSession)
		}
	}

	generation, err := loadGeneration(ctx, sessionGenerationKey(session.User.ID))
	if err != nil {
		return nil, fmt.Errorf("load session generation failed: %w", err)
	}
	if generation != session.Generation {
		return nil, fmt.Errorf("%w: revoked user session", errInvalidSession)
	}
	session.User.sessionPersistent = session.Persistent
	return &session.User, nil
}

func LoadSession(ctx context.Context, signed string) (*User, error) {
	return loadSessionWithStores(
		ctx,
		signed,
		func(ctx context.Context, key string) ([]byte, error) {
			return rdb.Get(ctx, key).Bytes()
		},
		func(ctx context.Context, key string) (int64, error) {
			generation, err := rdb.Get(ctx, key).Int64()
			if errors.Is(err, redis.Nil) {
				return 0, nil
			}
			return generation, err
		},
	)
}

type sessionIdleDeadlineRefresher func(context.Context, string, time.Duration) (bool, error)

func refreshPersistentSessionWithStore(
	ctx context.Context,
	signed string,
	user *User,
	refresh sessionIdleDeadlineRefresher,
) error {
	if user == nil || !user.sessionPersistent {
		return nil
	}

	sid, err := verifySID(signed)
	if err != nil {
		return err
	}
	refreshed, err := refresh(ctx, sessionPrefix+sid, rememberSessionTTL)
	if err != nil {
		return fmt.Errorf("refresh session failed: %w", err)
	}
	if !refreshed {
		return fmt.Errorf("%w: expired during refresh", errInvalidSession)
	}
	return nil
}

func LoadSessionAndRefresh(ctx context.Context, signed string) (*User, error) {
	user, err := LoadSession(ctx, signed)
	if err != nil {
		return nil, err
	}
	if err := refreshPersistentSessionWithStore(
		ctx,
		signed,
		user,
		func(ctx context.Context, key string, ttl time.Duration) (bool, error) {
			return rdb.Expire(ctx, key, ttl).Result()
		},
	); err != nil {
		return nil, err
	}
	return user, nil
}

type sessionLoader func(context.Context, string) (*User, error)

type sessionStateDestroyer func(context.Context, string, string) error

func sessionRevocationChannel(sid string) string {
	digest := sha256.Sum256([]byte(sid))
	return sessionRevocationPrefix + hex.EncodeToString(digest[:])
}

func sessionGenerationKey(userID int) string {
	return sessionGenerationPrefix + strconv.Itoa(userID)
}

func destroySessionWithStore(ctx context.Context, signed string, destroy sessionStateDestroyer) error {
	sid, err := verifySID(signed)
	if err != nil {
		return nil
	}
	if err := destroy(ctx, sessionPrefix+sid, sessionRevocationChannel(sid)); err != nil {
		return fmt.Errorf("destroy session failed: %w", err)
	}
	return nil
}

func DestroySession(ctx context.Context, signed string) error {
	return destroySessionWithStore(ctx, signed, func(ctx context.Context, key, channel string) error {
		return destroySessionScript.Run(ctx, rdb, []string{key, channel}).Err()
	})
}

// requireAuthWithLoaders 是需要登入的 handler middleware，驗證成功後把 User 放進 context。
func requireAuthWithLoaders(next http.HandlerFunc, loadSession, refreshSession sessionLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, M{"error": "unauthorized"})
			return
		}
		refreshAllowed := r.Header.Get(browserRequestHeader) == "1"
		selectedLoader := loadSession
		if refreshAllowed {
			selectedLoader = refreshSession
		}
		user, err := selectedLoader(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, errInvalidSession) {
				writeJSON(w, http.StatusUnauthorized, M{"error": "invalid session"})
				return
			}

			writeJSON(w, http.StatusServiceUnavailable, M{"error": "session service unavailable"})
			return
		}
		if user == nil {
			writeJSON(w, http.StatusServiceUnavailable, M{"error": "session service unavailable"})
			return
		}
		if refreshAllowed && user.sessionPersistent {
			http.SetCookie(w, sessionCookieForLogin(cookie.Value, rememberSessionTTL, true))
		}

		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next(w, r.WithContext(ctx))
	}
}

func currentUser(r *http.Request) *User {
	return r.Context().Value(userCtxKey).(*User)
}
