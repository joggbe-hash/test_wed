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
	"strings"
	"time"

	"github.com/google/uuid"
)

const sessionPrefix = "sess:"

var (
	sessionSecret []byte
	sessionTTL    time.Duration
)

func InitSession(cfg *Config) {
	sessionSecret = []byte(cfg.SecretKey)
	sessionTTL = time.Duration(cfg.SessionTTL) * time.Second
}

// User 是放進 Redis session 的最小使用者資料，handler 從 request context 取用。
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
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
		return "", errors.New("invalid session format")
	}
	sid, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(sid))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", errors.New("invalid session signature")
	}
	return sid, nil
}

// CreateSession 只把 signed sid 放 cookie，完整 user 資料存在 Redis，方便登出與 TTL 控制。
func CreateSession(ctx context.Context, user *User) (string, error) {
	sid := uuid.New().String()
	data, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("marshal session user failed: %w", err)
	}
	if err := rdb.Set(ctx, sessionPrefix+sid, data, sessionTTL).Err(); err != nil {
		return "", fmt.Errorf("save session failed: %w", err)
	}
	return signSID(sid), nil
}

func LoadSession(ctx context.Context, signed string) (*User, error) {
	sid, err := verifySID(signed)
	if err != nil {
		return nil, err
	}
	data, err := rdb.Get(ctx, sessionPrefix+sid).Bytes()
	if err != nil {
		return nil, errors.New("session expired or not found")
	}
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("decode session failed: %w", err)
	}
	return &user, nil
}

func DestroySession(ctx context.Context, signed string) {
	sid, err := verifySID(signed)
	if err != nil {
		return
	}
	rdb.Del(ctx, sessionPrefix+sid)
}

// requireAuth 是需要登入的 handler middleware，驗證成功後把 User 放進 context。
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, M{"error": "unauthorized"})
			return
		}
		user, err := LoadSession(r.Context(), cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, M{"error": "invalid session"})
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next(w, r.WithContext(ctx))
	}
}

func currentUser(r *http.Request) *User {
	return r.Context().Value(userCtxKey).(*User)
}
