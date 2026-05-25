package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// Redis 中驗證碼的鍵前綴與存活時間
const (
	codePrefix = "vcode:" // 驗證碼在 Redis 中的 key 前綴
	codeTTL    = 5 * time.Minute
)

// sendCodeRequest 寄送驗證碼的請求格式
type sendCodeRequest struct {
	Email string `json:"email"`
}

// handleSendCode 處理 POST /api/auth/send-code
// 產生 6 位數驗證碼，存入 Redis 並透過 Worker 非同步寄信
func handleSendCode(w http.ResponseWriter, r *http.Request) {
	var req sendCodeRequest
	if err := readJSON(r, &req); err != nil || req.Email == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "email 為必填欄位"})
		return
	}

	// 產生 6 位數驗證碼
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	ctx := r.Context()
	ops := NewAtomicRollback()

	// 步驟 1：將驗證碼存入 Redis，設定 5 分鐘過期
	codeKey := codePrefix + req.Email
	if err := rdb.Set(ctx, codeKey, code, codeTTL).Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "驗證碼儲存失敗"})
		return
	}
	ops.Add("刪除 Redis 驗證碼", func() error {
		return rdb.Del(ctx, codeKey).Err()
	})

	// 步驟 2：將寄信任務放入 Redis 佇列，由 Worker 非同步處理
	job, _ := json.Marshal(M{
		"type": "send_verification_email",
		"payload": M{
			"email": req.Email,
			"code":  code,
		},
	})
	if err := rdb.RPush(ctx, "task_queue", job).Err(); err != nil {
		// 佇列推送失敗 → 回滾步驟 1（刪除已儲存的驗證碼）
		ops.Execute()
		writeJSON(w, http.StatusInternalServerError, M{"error": "任務排程失敗"})
		return
	}

	writeJSON(w, http.StatusOK, M{"message": "驗證碼已發送", "debug_code": code})
}

// registerRequest 註冊的請求格式
type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

// handleRegister 處理 POST /api/auth/register
// 驗證碼比對 → 密碼雜湊 → 寫入資料庫，使用 DB 交易確保原子性
func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "請求格式錯誤"})
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "所有欄位皆為必填"})
		return
	}

	ctx := r.Context()

	// 從 Redis 取出驗證碼並比對
	storedCode, err := rdb.Get(ctx, codePrefix+req.Email).Result()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "驗證碼已過期，請重新取得"})
		return
	}
	if storedCode != req.Code {
		writeJSON(w, http.StatusBadRequest, M{"error": "驗證碼錯誤"})
		return
	}

	// 使用 bcrypt 雜湊密碼（cost=12 提供足夠的暴力破解防護）
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "密碼處理失敗"})
		return
	}

	// 在 user_db 交易中插入使用者記錄
	var userID int
	err = WithTx(ctx, userPool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
			req.Username, req.Email, string(hash),
		).Scan(&userID)
	})
	if err != nil {
		// 若違反 email 唯一性約束，回傳 409 Conflict
		if isUniqueViolation(err) {
			writeJSON(w, http.StatusConflict, M{"error": "此 email 已被註冊"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "註冊失敗"})
		return
	}

	// 註冊成功後刪除驗證碼（失敗不影響結果，TTL 會自動清理）
	rdb.Del(ctx, codePrefix+req.Email)

	writeJSON(w, http.StatusCreated, M{"message": "註冊成功", "user_id": userID})
}

// loginRequest 登入的請求格式
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleLogin 處理 POST /api/auth/login
// 驗證帳號密碼 → 建立 Redis session → 回傳加密的 session cookie
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "email 和 password 為必填"})
		return
	}

	ctx := r.Context()

	// 查詢使用者記錄
	var user User
	var passwordHash string
	err := userPool.QueryRow(ctx,
		"SELECT id, username, email, password_hash FROM users WHERE email = $1",
		req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &passwordHash)

	if err != nil {
		// 不論是查無此人或其他錯誤，都回傳相同訊息以防止帳號列舉攻擊
		writeJSON(w, http.StatusUnauthorized, M{"error": "帳號或密碼錯誤"})
		return
	}

	// 比對 bcrypt 雜湊密碼
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, M{"error": "帳號或密碼錯誤"})
		return
	}

	// 建立 session 並回傳 cookie
	signed, err := CreateSession(ctx, &user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "session 建立失敗"})
		return
	}

	// 設定 HttpOnly + Secure cookie，防止 XSS 竊取 session
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    signed,
		Path:     "/",
		HttpOnly: true,           // 禁止 JavaScript 存取
		Secure:   true,           // 僅在 HTTPS 傳輸
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	writeJSON(w, http.StatusOK, M{"user": user})
}

// handleLogout 處理 POST /api/auth/logout
// 從 Redis 刪除 session 並清除客戶端 cookie
func handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		DestroySession(r.Context(), cookie.Value)
	}

	// 清除客戶端的 session cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1, // 立即過期
	})

	writeJSON(w, http.StatusOK, M{"message": "已登出"})
}

// isUniqueViolation 檢查 PostgreSQL 錯誤是否為唯一性約束違反（SQLSTATE 23505）
func isUniqueViolation(err error) bool {
	return err != nil && contains(err.Error(), "23505")
}

// contains 簡單的子字串搜尋
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
