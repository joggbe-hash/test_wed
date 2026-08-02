package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"typewsp/shared/contracts"
	"typewsp/shared/rollback"
)

const (
	codePrefix                     = "vcode:"
	verificationAttemptsPrefix     = "vcode:attempts:"
	verificationSendCooldownPrefix = "vcode:send-cooldown:"
	codeTTL                        = 5 * time.Minute
	verificationCodeAttemptLimit   = 5
	verificationCodeSendCooldown   = time.Minute
	rememberSessionTTL             = 30 * 24 * time.Hour
	maxAuthRequestBytes            = 64 << 10
	minUsernameRunes               = 2
	maxUsernameRunes               = 20
	minPasswordBytes               = 8
	maxPasswordBytes               = 72
)

var debugVerificationCode bool

var (
	storeVerificationCodeScript = redis.NewScript(`
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
redis.call('DEL', KEYS[2])
return 1
`)
	recordFailedVerificationAttemptScript = redis.NewScript(`
local storedCode = redis.call('GET', KEYS[1])
if not storedCode or storedCode ~= ARGV[1] then
  return -1
end
local ttl = redis.call('PTTL', KEYS[1])
if ttl <= 0 then
  return -1
end
local attempts = redis.call('INCR', KEYS[2])
if attempts == 1 then
  redis.call('PEXPIRE', KEYS[2], ttl)
end
if attempts >= tonumber(ARGV[2]) then
  redis.call('DEL', KEYS[1])
  redis.call('DEL', KEYS[2])
end
return attempts
`)
)

func InitAuth(cfg *Config) {
	debugVerificationCode = cfg.DebugVerificationCode
}

func generateVerificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate verification code failed: %w", err)
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func verificationCodeResponse(code string) M {
	response := M{"message": "verification code sent"}
	if debugVerificationCode {
		response["debug_code"] = code
	}
	return response
}

func verificationAttemptKey(email string) string {
	return verificationAttemptsPrefix + email
}

func verificationSendCooldownKey(email string) string {
	return verificationSendCooldownPrefix + email
}

func recordFailedVerificationAttempt(ctx context.Context, codeKey, attemptKey, expectedCode string) (int, error) {
	attempts, err := recordFailedVerificationAttemptScript.Run(
		ctx,
		rdb,
		[]string{codeKey, attemptKey},
		expectedCode,
		verificationCodeAttemptLimit,
	).Int()
	if err != nil {
		return 0, fmt.Errorf("record verification attempt failed: %w", err)
	}
	return attempts, nil
}

type sendCodeRequest struct {
	Email string `json:"email"`
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > 254 || !utf8.ValidString(email) {
		return "", fmt.Errorf("invalid email")
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", fmt.Errorf("invalid email")
	}

	return email, nil
}

func normalizeUsername(raw string) (string, error) {
	username := strings.TrimSpace(raw)
	length := utf8.RuneCountInString(username)
	if !utf8.ValidString(username) || length < minUsernameRunes || length > maxUsernameRunes {
		return "", fmt.Errorf("username must contain 2-20 characters")
	}

	for _, char := range username {
		if unicode.IsSpace(char) || unicode.IsControl(char) {
			return "", fmt.Errorf("username must not contain spaces or control characters")
		}
	}

	return username, nil
}

func validatePassword(password string) error {
	if !utf8.ValidString(password) || len(password) < minPasswordBytes || len(password) > maxPasswordBytes {
		return fmt.Errorf("password must contain 8-72 bytes")
	}

	var hasLetter, hasDigit bool
	for _, char := range password {
		hasLetter = hasLetter || unicode.IsLetter(char)
		hasDigit = hasDigit || unicode.IsDigit(char)
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("password must include at least one letter and one number")
	}

	return nil
}

func limitAuthRequestBody(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBytes)
}

// handleSendCode 產生 6 位數驗證碼，先放 Redis，再推送 email job 給 worker。
// debug_code 僅能在明確的 development 設定下回傳，部署環境預設不暴露。
func handleSendCode(w http.ResponseWriter, r *http.Request) {
	limitAuthRequestBody(w, r)
	var req sendCodeRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "請輸入有效的電子信箱"})
		return
	}

	ctx := r.Context()
	cooldownKey := verificationSendCooldownKey(email)
	allowed, err := rdb.SetNX(ctx, cooldownKey, "1", verificationCodeSendCooldown).Result()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "start verification request failed"})
		return
	}
	if !allowed {
		writeJSON(w, http.StatusTooManyRequests, M{"error": "please wait before requesting another verification code"})
		return
	}

	code, err := generateVerificationCode()
	if err != nil {
		rdb.Del(ctx, cooldownKey)
		writeJSON(w, http.StatusInternalServerError, M{"error": "generate verification code failed"})
		return
	}
	ops := rollback.New()

	codeKey := codePrefix + email
	attemptKey := verificationAttemptKey(email)
	if err := storeVerificationCodeScript.Run(ctx, rdb, []string{codeKey, attemptKey}, code, codeTTL.Milliseconds()).Err(); err != nil {
		rdb.Del(ctx, cooldownKey)
		writeJSON(w, http.StatusInternalServerError, M{"error": "save verification code failed"})
		return
	}
	ops.Add("delete verification state", func(cleanupCtx context.Context) error {
		return rdb.Del(cleanupCtx, codeKey, attemptKey, cooldownKey).Err()
	})

	if err := enqueueTask(ctx, contracts.TaskSendVerificationEmail, M{
		"email": email,
		"code":  code,
	}); err != nil {
		if rollbackErr := ops.Execute(ctx); rollbackErr != nil {
			log.Printf("rollback send-code request failed: %v", rollbackErr)
		}
		if errors.Is(err, errTaskQueueFull) {
			writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service is busy"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "enqueue email job failed"})
		return
	}

	writeJSON(w, http.StatusOK, verificationCodeResponse(code))
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

// handleRegister 驗證 Redis 內的驗證碼，通過後把使用者寫入 user_db。
func handleRegister(w http.ResponseWriter, r *http.Request) {
	limitAuthRequestBody(w, r)
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "username, email, password and code are required"})
		return
	}

	username, err := normalizeUsername(req.Username)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "使用者名稱需為 2-20 個字，且不能包含空白"})
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "請輸入有效的電子信箱"})
		return
	}
	if err := validatePassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "密碼需為 8-72 bytes，且至少包含一個字母與一個數字"})
		return
	}
	if len(req.Code) != 6 || strings.IndexFunc(req.Code, func(char rune) bool { return !unicode.IsDigit(char) }) >= 0 {
		writeJSON(w, http.StatusBadRequest, M{"error": "驗證碼必須為 6 位數字"})
		return
	}

	ctx := r.Context()

	codeKey := codePrefix + email
	attemptKey := verificationAttemptKey(email)
	storedCode, err := rdb.Get(ctx, codeKey).Result()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "verification code expired or not found"})
		return
	}
	if storedCode != req.Code {
		attempts, attemptErr := recordFailedVerificationAttempt(ctx, codeKey, attemptKey, storedCode)
		if attemptErr != nil {
			writeJSON(w, http.StatusInternalServerError, M{"error": "verify registration code failed"})
			return
		}
		if attempts < 0 {
			writeJSON(w, http.StatusBadRequest, M{"error": "verification code expired or not found"})
			return
		}
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
			username, email, string(hash),
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

	rdb.Del(ctx, codeKey, attemptKey)
	writeJSON(w, http.StatusCreated, M{"message": "registered", "user_id": userID})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

// handleLogin 檢查帳密後建立 Redis session，並用 HttpOnly cookie 回傳 session id。
func handleLogin(w http.ResponseWriter, r *http.Request) {
	limitAuthRequestBody(w, r)
	var req loginRequest
	if err := readJSON(r, &req); err != nil || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "請輸入電子信箱與密碼"})
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil || len(req.Password) > maxPasswordBytes {
		writeJSON(w, http.StatusUnauthorized, M{"error": "電子信箱或密碼不正確"})
		return
	}

	ctx := r.Context()

	var user User
	var passwordHash string
	err = userPool.QueryRow(ctx,
		"SELECT id, username, email, password_hash FROM users WHERE email = $1",
		email,
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

// handleCurrentSession 回傳已驗證 session 對應的最小使用者資料，供前端恢復登入狀態。
func handleCurrentSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, M{"user": currentUser(r)})
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
