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

// sessionPrefix 是 Redis 中 session 鍵的前綴
const sessionPrefix = "sess:"

// 全域 session 設定，在 main 中初始化
var (
	sessionSecret []byte // HMAC 簽名金鑰
	sessionTTL    time.Duration
)

// InitSession 設定 session 的加密金鑰與存活時間
func InitSession(cfg *Config) {
	sessionSecret = []byte(cfg.SecretKey)
	sessionTTL = time.Duration(cfg.SessionTTL) * time.Second
}

// User 代表已登入使用者的基本資訊，儲存在 Redis session 中
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// contextKey 自訂型別，避免 context key 衝突
type contextKey string

const userCtxKey contextKey = "user"

// signSID 使用 HMAC-SHA256 對 session ID 簽名
// 回傳格式為 "sid.signature"，防止使用者偽造 session ID
func signSID(sid string) string {
	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(sid))
	sig := hex.EncodeToString(mac.Sum(nil))
	return sid + "." + sig
}

// verifySID 驗證已簽名的 session ID
// 回傳原始 sid 或錯誤（簽名不符或格式錯誤）
func verifySID(signed string) (string, error) {
	parts := strings.SplitN(signed, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("session 格式無效")
	}
	sid, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(sid))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", errors.New("session 簽名驗證失敗")
	}
	return sid, nil
}

// CreateSession 在 Redis 建立一筆 session 並回傳已簽名的 session ID
// user 資料會被序列化為 JSON 存入 Redis，設定 TTL 自動過期
func CreateSession(ctx context.Context, user *User) (string, error) {
	sid := uuid.New().String()
	data, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("序列化 session 失敗: %w", err)
	}
	if err := rdb.Set(ctx, sessionPrefix+sid, data, sessionTTL).Err(); err != nil {
		return "", fmt.Errorf("寫入 Redis session 失敗: %w", err)
	}
	return signSID(sid), nil
}

// LoadSession 根據已簽名的 session ID 從 Redis 載入使用者資料
// 若簽名無效或 session 已過期，回傳 error
func LoadSession(ctx context.Context, signed string) (*User, error) {
	sid, err := verifySID(signed)
	if err != nil {
		return nil, err
	}
	data, err := rdb.Get(ctx, sessionPrefix+sid).Bytes()
	if err != nil {
		return nil, errors.New("session 已過期或不存在")
	}
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("反序列化 session 失敗: %w", err)
	}
	return &user, nil
}

// DestroySession 從 Redis 刪除指定的 session
func DestroySession(ctx context.Context, signed string) {
	sid, err := verifySID(signed)
	if err != nil {
		return
	}
	rdb.Del(ctx, sessionPrefix+sid)
}

// requireAuth 是 HTTP 中介層，檢查請求是否帶有有效的 session cookie
// 通過驗證後，將使用者資料存入 request context，後續 handler 可透過 currentUser 取得
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, M{"error": "未登入"})
			return
		}
		user, err := LoadSession(r.Context(), cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, M{"error": "session 已過期"})
			return
		}
		// 將使用者資料放入 context，傳遞給下游 handler
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next(w, r.WithContext(ctx))
	}
}

// currentUser 從 request context 取出已驗證的使用者資料
// 僅限在 requireAuth 保護的 handler 中使用
func currentUser(r *http.Request) *User {
	return r.Context().Value(userCtxKey).(*User)
}
