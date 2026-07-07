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

const (
	codePrefix         = "vcode:"
	codeTTL            = 5 * time.Minute
	rememberSessionTTL = 30 * 24 * time.Hour
)

type sendCodeRequest struct {
	Email string `json:"email"`
}

// handleSendCode 產生 6 位數驗證碼，先放 Redis，再推送 email job 給 worker。
// 第一版為了團隊 demo 會回傳 debug_code；正式環境要移除。
func handleSendCode(w http.ResponseWriter, r *http.Request) {
	var req sendCodeRequest
	if err := readJSON(r, &req); err != nil || req.Email == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "email is required"})
		return
	}

	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	ctx := r.Context()
	ops := NewAtomicRollback()

	codeKey := codePrefix + req.Email
	if err := rdb.Set(ctx, codeKey, code, codeTTL).Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "save verification code failed"})
		return
	}
	ops.Add("delete verification code "+codeKey, func() error {
		return rdb.Del(ctx, codeKey).Err()
	})

	job, _ := json.Marshal(M{
		"type": "send_verification_email",
		"payload": M{
			"email": req.Email,
			"code":  code,
		},
	})
	if err := rdb.RPush(ctx, "task_queue", job).Err(); err != nil {
		ops.Execute()
		writeJSON(w, http.StatusInternalServerError, M{"error": "enqueue email job failed"})
		return
	}

	writeJSON(w, http.StatusOK, M{"message": "verification code sent", "debug_code": code})
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

// handleRegister 驗證 Redis 內的驗證碼，通過後把使用者寫入 user_db。
func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "username, email, password and code are required"})
		return
	}

	ctx := r.Context()

	storedCode, err := rdb.Get(ctx, codePrefix+req.Email).Result()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "verification code expired or not found"})
		return
	}
	if storedCode != req.Code {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid verification code"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "hash password failed"})
		return
	}

	var userID int
	err = WithTx(ctx, userPool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
			req.Username, req.Email, string(hash),
		).Scan(&userID)
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeJSON(w, http.StatusConflict, M{"error": "email already registered"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "register failed"})
		return
	}

	rdb.Del(ctx, codePrefix+req.Email)
	writeJSON(w, http.StatusCreated, M{"message": "registered", "user_id": userID})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

// handleLogin 檢查帳密後建立 Redis session，並用 HttpOnly cookie 回傳 session id。
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "email and password are required"})
		return
	}

	ctx := r.Context()

	var user User
	var passwordHash string
	err := userPool.QueryRow(ctx,
		"SELECT id, username, email, password_hash FROM users WHERE email = $1",
		req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &passwordHash)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, M{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, M{"error": "invalid email or password"})
		return
	}

	sessionDuration := sessionTTL
	if req.Remember {
		sessionDuration = rememberSessionTTL
	}

	signed, err := CreateSessionWithTTL(ctx, &user, sessionDuration)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "create session failed"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})

	writeJSON(w, http.StatusOK, M{"user": user})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		DestroySession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	writeJSON(w, http.StatusOK, M{"message": "logged out"})
}

func isUniqueViolation(err error) bool {
	return err != nil && contains(err.Error(), "23505")
}

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
