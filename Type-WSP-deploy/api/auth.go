package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"typewsp/shared/contracts"
	"typewsp/shared/rollback"
)

const (
	codePrefix                              = "vcode:"
	verificationActivePrefix                = "vcode:active:"
	verificationAttemptsPrefix              = "vcode:attempts:challenge:"
	verificationClientAttemptsPrefix        = "vcode:attempts:client:"
	verificationSendCooldownPrefix          = "vcode:send-cooldown:"
	verificationSendHourlyPrefix            = "vcode:send-hourly:"
	verificationSendDailyPrefix             = "vcode:send-daily:"
	verificationClientSendHourlyPrefix      = "vcode:send-client-hourly:"
	verificationClientSendDailyPrefix       = "vcode:send-client-daily:"
	codeTTL                                 = 5 * time.Minute
	registrationCodeTTL                     = 24 * time.Hour
	registrationVerificationCodeLength      = 16
	registrationVerificationCodeAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	verificationCodeAttemptLimit            = 5
	verificationChallengeAttemptLimit       = 20
	verificationCodeSendCooldown            = time.Minute
	verificationCodeHourlySendLimit         = 5
	verificationCodeDailySendLimit          = 10
	verificationClientHourlySendLimit       = 20
	verificationClientDailySendLimit        = 50
	loginAttemptPrefix                      = "login:attempts:"
	accountLoginAttemptPrefix               = "login:account-attempts:"
	loginVerificationCodePrefix             = "login:vcode:"
	loginVerificationActivePrefix           = "login:vcode:active:"
	loginVerificationAttemptsPrefix         = "login:vcode:attempts:challenge:"
	loginVerificationClientPrefix           = "login:vcode:attempts:client:"
	loginVerificationSendCooldownPrefix     = "login:vcode:send-cooldown:"
	loginVerificationSendHourlyPrefix       = "login:vcode:send-hourly:"
	loginVerificationSendDailyPrefix        = "login:vcode:send-daily:"
	loginVerificationClientHourlyPrefix     = "login:vcode:send-client-hourly:"
	loginVerificationClientDailyPrefix      = "login:vcode:send-client-daily:"
	loginAttemptLimit                       = 10
	loginAttemptWindow                      = 5 * time.Minute
	accountLoginAttemptLimit                = 25
	accountLoginAttemptWindow               = 15 * time.Minute
	maxConcurrentLoginPasswordVerifications = 4
	rememberSessionTTL                      = 30 * 24 * time.Hour
	maxAuthRequestBytes                     = 64 << 10
	minUsernameRunes                        = 2
	maxUsernameRunes                        = 20
	minPasswordRunes                        = 8
	maxPasswordBytes                        = 72
)

var (
	debugVerificationCode                bool
	sessionCookieSecure                  = true
	dummyLoginPasswordHash               = []byte("$2a$12$hVErFRzichm5h6hlc/CBx.97BpxRbUaDSfkqS5dBrr06P0Yd48E0q")
	loginPasswordVerificationConcurrency = newConcurrencyLimiter(maxConcurrentLoginPasswordVerifications)
)

var (
	reserveVerificationSendScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then
  return -1
end
local hourly = tonumber(redis.call('GET', KEYS[2]) or '0')
local daily = tonumber(redis.call('GET', KEYS[3]) or '0')
local clientHourly = tonumber(redis.call('GET', KEYS[4]) or '0')
local clientDaily = tonumber(redis.call('GET', KEYS[5]) or '0')
if hourly >= tonumber(ARGV[1]) then
  return -2
end
if daily >= tonumber(ARGV[2]) then
  return -3
end
if clientHourly >= tonumber(ARGV[3]) then
  return -4
end
if clientDaily >= tonumber(ARGV[4]) then
  return -5
end
redis.call('SET', KEYS[1], '1', 'PX', ARGV[5])
hourly = redis.call('INCR', KEYS[2])
if hourly == 1 then
  redis.call('PEXPIRE', KEYS[2], ARGV[6])
end
daily = redis.call('INCR', KEYS[3])
if daily == 1 then
  redis.call('PEXPIRE', KEYS[3], ARGV[7])
end
clientHourly = redis.call('INCR', KEYS[4])
if clientHourly == 1 then
  redis.call('PEXPIRE', KEYS[4], ARGV[6])
end
clientDaily = redis.call('INCR', KEYS[5])
if clientDaily == 1 then
  redis.call('PEXPIRE', KEYS[5], ARGV[7])
end
return 1
`)
	rollbackVerificationSendScript = redis.NewScript(`
redis.call('DEL', KEYS[1])
for i = 2, 5 do
  local current = tonumber(redis.call('GET', KEYS[i]) or '0')
  if current > 1 then
    redis.call('DECR', KEYS[i])
  elseif current == 1 then
    redis.call('DEL', KEYS[i])
  end
end
return 1
`)
	activateVerificationChallengeScript = redis.NewScript(`
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[3])
redis.call('SET', KEYS[2], ARGV[2], 'PX', ARGV[3])
return 1
`)
	rollbackVerificationChallengeScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) == ARGV[1] then
  redis.call('DEL', KEYS[1], KEYS[2])
end
return 1
`)
	rollbackRegistrationChallengeScript = redis.NewScript(`
redis.call('DEL', KEYS[1])
if redis.call('GET', KEYS[2]) == ARGV[1] then
  redis.call('DEL', KEYS[2])
end
return 1
`)
	consumeVerificationCodeScript = redis.NewScript(`
local storedCode = redis.call('GET', KEYS[1])
local activeChallenge = redis.call('GET', KEYS[2])
if not storedCode or activeChallenge ~= ARGV[1] then
  return -1
end
local clientAttempts = tonumber(redis.call('GET', KEYS[3]) or '0')
local challengeAttempts = tonumber(redis.call('GET', KEYS[4]) or '0')
if challengeAttempts >= tonumber(ARGV[4]) then
  redis.call('DEL', KEYS[1], KEYS[2], KEYS[3], KEYS[4])
  return 2
end
if clientAttempts >= tonumber(ARGV[3]) then
  return 2
end
if storedCode == ARGV[2] then
  redis.call('DEL', KEYS[1], KEYS[2], KEYS[3], KEYS[4])
  return 1
end
local ttl = redis.call('PTTL', KEYS[1])
clientAttempts = redis.call('INCR', KEYS[3])
challengeAttempts = redis.call('INCR', KEYS[4])
if ttl > 0 then
  if clientAttempts == 1 then redis.call('PEXPIRE', KEYS[3], ttl) end
  if challengeAttempts == 1 then redis.call('PEXPIRE', KEYS[4], ttl) end
end
if challengeAttempts >= tonumber(ARGV[4]) then
  redis.call('DEL', KEYS[1], KEYS[2], KEYS[3], KEYS[4])
  return 2
end
if clientAttempts >= tonumber(ARGV[3]) then
  return 2
end
return 0
`)
	consumeRegistrationCodeScript = redis.NewScript(`
local storedCode = redis.call('GET', KEYS[1])
if not storedCode then
  return -1
end
if storedCode == ARGV[2] then
  redis.call('DEL', KEYS[1], KEYS[3], KEYS[4])
  if redis.call('GET', KEYS[2]) == ARGV[1] then
    redis.call('DEL', KEYS[2])
  end
  return 1
end
local clientAttempts = tonumber(redis.call('GET', KEYS[3]) or '0')
local challengeAttempts = tonumber(redis.call('GET', KEYS[4]) or '0')
if clientAttempts >= tonumber(ARGV[3]) or challengeAttempts >= tonumber(ARGV[4]) then
  return 2
end
local ttl = redis.call('PTTL', KEYS[1])
clientAttempts = redis.call('INCR', KEYS[3])
challengeAttempts = redis.call('INCR', KEYS[4])
if ttl > 0 then
  if clientAttempts == 1 then redis.call('PEXPIRE', KEYS[3], ttl) end
  if challengeAttempts == 1 then redis.call('PEXPIRE', KEYS[4], ttl) end
end
if clientAttempts >= tonumber(ARGV[3]) or challengeAttempts >= tonumber(ARGV[4]) then
  return 2
end
return 0
`)
	reserveLoginAttemptScript = redis.NewScript(`
local clientAttempts = tonumber(redis.call('GET', KEYS[1]) or '0')
local accountAttempts = tonumber(redis.call('GET', KEYS[2]) or '0')
if clientAttempts >= tonumber(ARGV[3]) or accountAttempts >= tonumber(ARGV[4]) then
  return {0, clientAttempts, accountAttempts}
end
clientAttempts = redis.call('INCR', KEYS[1])
accountAttempts = redis.call('INCR', KEYS[2])
if clientAttempts == 1 or redis.call('PTTL', KEYS[1]) < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if accountAttempts == 1 or redis.call('PTTL', KEYS[2]) < 0 then
  redis.call('PEXPIRE', KEYS[2], ARGV[2])
end
return {1, clientAttempts, accountAttempts}
`)
)

func InitAuth(cfg *Config) {
	debugVerificationCode = cfg.DebugVerificationCode
	sessionCookieSecure = cfg.AppEnv != "development" && cfg.AppEnv != "test"
}

func generateVerificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate verification code failed: %w", err)
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func generateRegistrationVerificationCode() (string, error) {
	raw := make([]byte, 10)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate registration verification code failed: %w", err)
	}
	encoding := base32.NewEncoding(registrationVerificationCodeAlphabet).WithPadding(base32.NoPadding)
	return encoding.EncodeToString(raw), nil
}

func validRegistrationVerificationCode(code string) bool {
	if len(code) != registrationVerificationCodeLength {
		return false
	}
	return strings.IndexFunc(code, func(char rune) bool {
		return !strings.ContainsRune(registrationVerificationCodeAlphabet, char)
	}) < 0
}

func verificationCodeResponse(challengeID, code string) M {
	response := M{"message": "verification code sent", "challenge_id": challengeID}
	if debugVerificationCode {
		response["debug_code"] = code
	}
	return response
}

func loginVerificationCodeResponse(challengeID, code string) M {
	response := M{
		"message":               "login verification code sent",
		"challenge_id":          challengeID,
		"requires_verification": true,
		"expires_in_seconds":    int(codeTTL.Seconds()),
	}
	if debugVerificationCode {
		response["debug_code"] = code
	}
	return response
}

func hashedEmailKey(prefix, email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	digest := sha256.Sum256([]byte(normalized))
	return prefix + hex.EncodeToString(digest[:])
}

func verificationCodeKey(email, challengeID string) string {
	return hashedValueKey(codePrefix, strings.ToLower(strings.TrimSpace(email))+"\x00"+challengeID)
}

func verificationLatestChallengeKey(email string) string {
	return hashedEmailKey(verificationActivePrefix, email)
}

func hashedValueKey(prefix, value string) string {
	digest := sha256.Sum256([]byte(value))
	return prefix + hex.EncodeToString(digest[:])
}

func verificationChallengeAttemptKey(challengeID string) string {
	return hashedValueKey(verificationAttemptsPrefix, challengeID)
}

func verificationClientAttemptKey(challengeID, clientIdentity string) string {
	return hashedValueKey(verificationClientAttemptsPrefix, challengeID+"\x00"+clientIdentity)
}

func loginVerificationCodeKey(email string) string {
	return hashedEmailKey(loginVerificationCodePrefix, email)
}

func loginVerificationActiveChallengeKey(email string) string {
	return hashedEmailKey(loginVerificationActivePrefix, email)
}

func loginVerificationChallengeAttemptKey(challengeID string) string {
	return hashedValueKey(loginVerificationAttemptsPrefix, challengeID)
}

func loginVerificationClientAttemptKey(challengeID, clientIdentity string) string {
	return hashedValueKey(loginVerificationClientPrefix, challengeID+"\x00"+clientIdentity)
}

func loginVerificationSendCooldownKey(email string) string {
	return hashedEmailKey(loginVerificationSendCooldownPrefix, email)
}

func loginVerificationSendHourlyKey(email string) string {
	return hashedEmailKey(loginVerificationSendHourlyPrefix, email)
}

func loginVerificationSendDailyKey(email string) string {
	return hashedEmailKey(loginVerificationSendDailyPrefix, email)
}

func loginVerificationClientSendHourlyKey(clientIdentity string) string {
	return hashedValueKey(loginVerificationClientHourlyPrefix, clientIdentity)
}

func loginVerificationClientSendDailyKey(clientIdentity string) string {
	return hashedValueKey(loginVerificationClientDailyPrefix, clientIdentity)
}

func verificationSendCooldownKey(email string) string {
	return hashedEmailKey(verificationSendCooldownPrefix, email)
}

func verificationSendHourlyKey(email string) string {
	return hashedEmailKey(verificationSendHourlyPrefix, email)
}

func verificationSendDailyKey(email string) string {
	return hashedEmailKey(verificationSendDailyPrefix, email)
}

func verificationClientSendHourlyKey(clientIdentity string) string {
	return hashedValueKey(verificationClientSendHourlyPrefix, clientIdentity)
}

func verificationClientSendDailyKey(clientIdentity string) string {
	return hashedValueKey(verificationClientSendDailyPrefix, clientIdentity)
}

func loginAttemptKey(email, clientIdentity string) string {
	normalized := strings.ToLower(strings.TrimSpace(email)) + "\x00" + clientIdentity
	digest := sha256.Sum256([]byte(normalized))
	return loginAttemptPrefix + hex.EncodeToString(digest[:])
}

func accountLoginAttemptKey(email string) string {
	return hashedEmailKey(accountLoginAttemptPrefix, email)
}

func loginAttemptAllowed(attempts int64) bool {
	return attempts > 0 && attempts <= loginAttemptLimit
}

func redisCounter(value any) (int64, error) {
	if value == nil {
		return 0, nil
	}
	switch typed := value.(type) {
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	case int64:
		return typed, nil
	default:
		return 0, fmt.Errorf("unexpected Redis counter type %T", value)
	}
}

func reserveLoginAttempt(ctx context.Context, email, clientIdentity string) (bool, int64, int64, error) {
	values, err := reserveLoginAttemptScript.Run(
		ctx,
		rdb,
		[]string{loginAttemptKey(email, clientIdentity), accountLoginAttemptKey(email)},
		loginAttemptWindow.Milliseconds(),
		accountLoginAttemptWindow.Milliseconds(),
		loginAttemptLimit,
		accountLoginAttemptLimit,
	).Slice()
	if err != nil {
		return false, 0, 0, fmt.Errorf("reserve login attempt failed: %w", err)
	}
	if len(values) != 3 {
		return false, 0, 0, fmt.Errorf("reserve login attempt returned %d values", len(values))
	}
	reserved, err := redisCounter(values[0])
	if err != nil {
		return false, 0, 0, fmt.Errorf("decode login reservation result: %w", err)
	}
	clientAttempts, err := redisCounter(values[1])
	if err != nil {
		return false, 0, 0, fmt.Errorf("decode client login attempts: %w", err)
	}
	accountAttempts, err := redisCounter(values[2])
	if err != nil {
		return false, 0, 0, fmt.Errorf("decode account login attempts: %w", err)
	}
	if reserved == 0 && !loginPreAuthenticationShouldBlock(clientAttempts, accountAttempts) {
		return false, 0, 0, fmt.Errorf("login reservation rejected below configured limits")
	}
	return reserved == 1, clientAttempts, accountAttempts, nil
}

func loginAttemptShouldBlock(attempts int64) bool {
	return attempts >= loginAttemptLimit
}

func accountLoginAttemptShouldBlock(attempts int64) bool {
	return attempts >= accountLoginAttemptLimit
}

func loginPreAuthenticationShouldBlock(clientAttempts, accountAttempts int64) bool {
	return loginAttemptShouldBlock(clientAttempts) || accountLoginAttemptShouldBlock(accountAttempts)
}

func resetLoginAttempts(ctx context.Context, email, clientIdentity string) error {
	if err := rdb.Del(ctx, loginAttemptKey(email, clientIdentity), accountLoginAttemptKey(email)).Err(); err != nil {
		return fmt.Errorf("reset login attempts failed: %w", err)
	}
	return nil
}

func verificationAttemptAllowed(attempts int64) bool {
	return attempts >= 0 && attempts < verificationCodeAttemptLimit
}

func verificationSendAllowed(hourly, daily int64) bool {
	return hourly >= 0 && daily >= 0 && hourly < verificationCodeHourlySendLimit && daily < verificationCodeDailySendLimit
}

func reserveVerificationSend(ctx context.Context, email, clientIdentity string) (int64, error) {
	result, err := reserveVerificationSendScript.Run(
		ctx,
		rdb,
		[]string{
			verificationSendCooldownKey(email),
			verificationSendHourlyKey(email),
			verificationSendDailyKey(email),
			verificationClientSendHourlyKey(clientIdentity),
			verificationClientSendDailyKey(clientIdentity),
		},
		verificationCodeHourlySendLimit,
		verificationCodeDailySendLimit,
		verificationClientHourlySendLimit,
		verificationClientDailySendLimit,
		verificationCodeSendCooldown.Milliseconds(),
		time.Hour.Milliseconds(),
		(24 * time.Hour).Milliseconds(),
	).Int64()
	if err != nil {
		return 0, fmt.Errorf("reserve verification send failed: %w", err)
	}
	return result, nil
}

func rollbackVerificationSend(ctx context.Context, email, clientIdentity string) error {
	return rollbackVerificationSendScript.Run(
		ctx,
		rdb,
		[]string{
			verificationSendCooldownKey(email),
			verificationSendHourlyKey(email),
			verificationSendDailyKey(email),
			verificationClientSendHourlyKey(clientIdentity),
			verificationClientSendDailyKey(clientIdentity),
		},
	).Err()
}

func reserveLoginVerificationSend(ctx context.Context, email, clientIdentity string) (int64, error) {
	result, err := reserveVerificationSendScript.Run(
		ctx,
		rdb,
		[]string{
			loginVerificationSendCooldownKey(email),
			loginVerificationSendHourlyKey(email),
			loginVerificationSendDailyKey(email),
			loginVerificationClientSendHourlyKey(clientIdentity),
			loginVerificationClientSendDailyKey(clientIdentity),
		},
		verificationCodeHourlySendLimit,
		verificationCodeDailySendLimit,
		verificationClientHourlySendLimit,
		verificationClientDailySendLimit,
		verificationCodeSendCooldown.Milliseconds(),
		time.Hour.Milliseconds(),
		(24 * time.Hour).Milliseconds(),
	).Int64()
	if err != nil {
		return 0, fmt.Errorf("reserve login verification send failed: %w", err)
	}
	return result, nil
}

func rollbackLoginVerificationSend(ctx context.Context, email, clientIdentity string) error {
	return rollbackVerificationSendScript.Run(
		ctx,
		rdb,
		[]string{
			loginVerificationSendCooldownKey(email),
			loginVerificationSendHourlyKey(email),
			loginVerificationSendDailyKey(email),
			loginVerificationClientSendHourlyKey(clientIdentity),
			loginVerificationClientSendDailyKey(clientIdentity),
		},
	).Err()
}

type verificationResult int

const (
	verificationExpired  verificationResult = -1
	verificationRejected verificationResult = 0
	verificationAccepted verificationResult = 1
	verificationLocked   verificationResult = 2
)

func activateVerificationChallenge(ctx context.Context, email, challengeID, code string) error {
	return activateVerificationChallengeScript.Run(
		ctx,
		rdb,
		[]string{verificationCodeKey(email, challengeID), verificationLatestChallengeKey(email)},
		code,
		challengeID,
		registrationCodeTTL.Milliseconds(),
	).Err()
}

func activateLoginVerificationChallenge(ctx context.Context, email, challengeID, code string) error {
	return activateVerificationChallengeScript.Run(
		ctx,
		rdb,
		[]string{loginVerificationCodeKey(email), loginVerificationActiveChallengeKey(email)},
		code,
		challengeID,
		codeTTL.Milliseconds(),
	).Err()
}

func rollbackLoginVerificationChallenge(ctx context.Context, email, challengeID string) error {
	return rollbackVerificationChallengeScript.Run(
		ctx,
		rdb,
		[]string{loginVerificationCodeKey(email), loginVerificationActiveChallengeKey(email)},
		challengeID,
	).Err()
}

func rollbackVerificationChallenge(ctx context.Context, email, challengeID string) error {
	return rollbackRegistrationChallengeScript.Run(
		ctx,
		rdb,
		[]string{verificationCodeKey(email, challengeID), verificationLatestChallengeKey(email)},
		challengeID,
	).Err()
}

func consumeVerificationCode(ctx context.Context, email, challengeID, clientIdentity, candidate string) (verificationResult, error) {
	result, err := consumeRegistrationCodeScript.Run(
		ctx,
		rdb,
		[]string{
			verificationCodeKey(email, challengeID),
			verificationLatestChallengeKey(email),
			verificationClientAttemptKey(challengeID, clientIdentity),
			verificationChallengeAttemptKey(challengeID),
		},
		challengeID,
		candidate,
		verificationCodeAttemptLimit,
		verificationChallengeAttemptLimit,
	).Int64()
	if err != nil {
		return verificationExpired, fmt.Errorf("consume verification code failed: %w", err)
	}
	return verificationResult(result), nil
}

func consumeLoginVerificationCode(ctx context.Context, email, challengeID, clientIdentity, candidate string) (verificationResult, error) {
	result, err := consumeVerificationCodeScript.Run(
		ctx,
		rdb,
		[]string{
			loginVerificationCodeKey(email),
			loginVerificationActiveChallengeKey(email),
			loginVerificationClientAttemptKey(challengeID, clientIdentity),
			loginVerificationChallengeAttemptKey(challengeID),
		},
		challengeID,
		candidate,
		verificationCodeAttemptLimit,
		verificationChallengeAttemptLimit,
	).Int64()
	if err != nil {
		return verificationExpired, fmt.Errorf("consume login verification code failed: %w", err)
	}
	return verificationResult(result), nil
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
	if !utf8.ValidString(password) {
		return fmt.Errorf("密碼格式不正確")
	}
	if utf8.RuneCountInString(password) < minPasswordRunes {
		return fmt.Errorf("密碼至少需要 8 個字元")
	}
	if len(password) > maxPasswordBytes {
		return fmt.Errorf("密碼過長，請縮短後再試")
	}

	var hasLetter, hasDigit bool
	for _, char := range password {
		hasLetter = hasLetter || unicode.IsLetter(char)
		hasDigit = hasDigit || unicode.IsDigit(char)
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("密碼須包含至少一個字母與一個數字")
	}

	return nil
}

func limitAuthRequestBody(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBytes)
}

// handleSendCode 產生高熵註冊驗證碼，先放 Redis，再推送 email job 給 worker。
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
	clientIdentity := requestClientIdentity(r)
	reservation, err := reserveVerificationSend(ctx, email, clientIdentity)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "start verification request failed"})
		return
	}
	if reservation < 0 {
		challengeID, latestErr := rdb.Get(ctx, verificationLatestChallengeKey(email)).Result()
		if latestErr == nil {
			code, _ := rdb.Get(ctx, verificationCodeKey(email, challengeID)).Result()
			writeJSON(w, http.StatusOK, verificationCodeResponse(challengeID, code))
			return
		}
		writeJSON(w, http.StatusTooManyRequests, M{"error": "please wait before requesting another verification code"})
		return
	}

	code, err := generateRegistrationVerificationCode()
	if err != nil {
		_ = rollbackVerificationSend(ctx, email, clientIdentity)
		writeJSON(w, http.StatusInternalServerError, M{"error": "generate verification code failed"})
		return
	}
	ops := rollback.New()
	ops.Add("release verification send reservation", func(cleanupCtx context.Context) error {
		return rollbackVerificationSend(cleanupCtx, email, clientIdentity)
	})

	challengeID := uuid.NewString()
	if err := activateVerificationChallenge(ctx, email, challengeID, code); err != nil {
		_ = rollbackVerificationSend(ctx, email, clientIdentity)
		writeJSON(w, http.StatusInternalServerError, M{"error": "save verification code failed"})
		return
	}
	ops.Add("delete verification challenge", func(cleanupCtx context.Context) error {
		return rollbackVerificationChallenge(cleanupCtx, email, challengeID)
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

	writeJSON(w, http.StatusOK, verificationCodeResponse(challengeID, code))
}

type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Code        string `json:"code"`
	ChallengeID string `json:"challenge_id"`
}

// handleRegister 驗證 Redis 內的驗證碼，通過後把使用者寫入 user_db。
func handleRegister(w http.ResponseWriter, r *http.Request) {
	limitAuthRequestBody(w, r)
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" || req.Code == "" || req.ChallengeID == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "username, email, password, code and challenge_id are required"})
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
		writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if !validRegistrationVerificationCode(req.Code) {
		writeJSON(w, http.StatusBadRequest, M{"error": "註冊驗證碼必須為 16 位大寫 Base32 字元"})
		return
	}
	if _, err := uuid.Parse(req.ChallengeID); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid verification challenge"})
		return
	}

	ctx := r.Context()

	result, err := consumeVerificationCode(ctx, email, req.ChallengeID, requestClientIdentity(r), req.Code)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service unavailable"})
		return
	}
	switch result {
	case verificationExpired:
		writeJSON(w, http.StatusBadRequest, M{"error": "verification code expired or not found"})
		return
	case verificationLocked:
		writeJSON(w, http.StatusTooManyRequests, M{"error": "too many verification attempts; please try again later"})
		return
	case verificationRejected:
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid verification code"})
		return
	case verificationAccepted:
		// Continue. The code is consumed atomically before account creation.
	default:
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service unavailable"})
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

	writeJSON(w, http.StatusCreated, M{"message": "registered", "user_id": userID})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type clientUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func clientUserFrom(user *User) clientUser {
	return clientUser{ID: user.ID, Username: user.Username}
}

type passwordHashComparer func(hashedPassword, password []byte) error

func verifyLoginPasswordWithCompare(passwordHash, password string, compare passwordHashComparer) bool {
	accountExists := passwordHash != ""
	hash := dummyLoginPasswordHash
	if accountExists {
		hash = []byte(passwordHash)
	}
	return compare(hash, []byte(password)) == nil && accountExists
}

func verifyLoginPassword(passwordHash, password string) bool {
	return verifyLoginPasswordWithCompare(passwordHash, password, bcrypt.CompareHashAndPassword)
}

// handleLogin 檢查帳密後寄送短效登入驗證碼；此階段不建立 Session。
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
	clientIdentity := requestClientIdentity(r)
	reserved, clientAttempts, accountAttempts, err := reserveLoginAttempt(ctx, email, clientIdentity)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "authentication service unavailable"})
		return
	}
	if !reserved {
		writeJSON(w, http.StatusTooManyRequests, M{"error": "too many login attempts; please try again later"})
		return
	}
	if !loginPasswordVerificationConcurrency.tryAcquire() {
		writeJSON(w, http.StatusTooManyRequests, M{"error": "authentication service is busy; please try again"})
		return
	}
	defer loginPasswordVerificationConcurrency.release()

	var user User
	var passwordHash string
	err = userPool.QueryRow(ctx,
		"SELECT id, username, email, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &passwordHash)
	if err != nil {
		passwordHash = ""
	}

	passwordMatches := verifyLoginPassword(passwordHash, req.Password)
	if !passwordMatches {
		if !loginAttemptAllowed(clientAttempts) || accountAttempts >= accountLoginAttemptLimit {
			writeJSON(w, http.StatusTooManyRequests, M{"error": "too many login attempts; please try again later"})
			return
		}
		writeJSON(w, http.StatusUnauthorized, M{"error": "invalid email or password"})
		return
	}
	reservation, err := reserveLoginVerificationSend(ctx, email, clientIdentity)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service unavailable"})
		return
	}
	if reservation < 0 {
		writeJSON(w, http.StatusTooManyRequests, M{"error": "please wait before requesting another login verification code"})
		return
	}

	code, err := generateVerificationCode()
	if err != nil {
		_ = rollbackLoginVerificationSend(ctx, email, clientIdentity)
		writeJSON(w, http.StatusInternalServerError, M{"error": "generate login verification code failed"})
		return
	}

	ops := rollback.New()
	ops.Add("release login verification send reservation", func(cleanupCtx context.Context) error {
		return rollbackLoginVerificationSend(cleanupCtx, email, clientIdentity)
	})

	challengeID := uuid.NewString()
	if err := activateLoginVerificationChallenge(ctx, email, challengeID, code); err != nil {
		_ = rollbackLoginVerificationSend(ctx, email, clientIdentity)
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service unavailable"})
		return
	}
	ops.Add("delete login verification challenge", func(cleanupCtx context.Context) error {
		return rollbackLoginVerificationChallenge(cleanupCtx, email, challengeID)
	})

	if err := enqueueTask(ctx, contracts.TaskSendVerificationEmail, M{
		"email": email,
		"code":  code,
	}); err != nil {
		if rollbackErr := ops.Execute(ctx); rollbackErr != nil {
			log.Printf("rollback login verification request failed: %v", rollbackErr)
		}
		if errors.Is(err, errTaskQueueFull) {
			writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service is busy"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "enqueue login verification email failed"})
		return
	}

	writeJSON(w, http.StatusAccepted, loginVerificationCodeResponse(challengeID, code))
}

type loginVerificationRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	ChallengeID string `json:"challenge_id"`
	Remember    bool   `json:"remember"`
}

// handleLoginVerification atomically consumes the email code before creating
// the Redis session. A password-only request can never reach this endpoint's
// session creation path because login challenges are issued only by handleLogin.
func handleLoginVerification(w http.ResponseWriter, r *http.Request) {
	limitAuthRequestBody(w, r)
	var req loginVerificationRequest
	if err := readJSON(r, &req); err != nil || req.Email == "" || req.Code == "" || req.ChallengeID == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "email, code and challenge_id are required"})
		return
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid login verification challenge"})
		return
	}
	if len(req.Code) != 6 || strings.IndexFunc(req.Code, func(char rune) bool { return !unicode.IsDigit(char) }) >= 0 {
		writeJSON(w, http.StatusBadRequest, M{"error": "verification code must contain 6 digits"})
		return
	}
	if _, err := uuid.Parse(req.ChallengeID); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid login verification challenge"})
		return
	}

	ctx := r.Context()
	var user User
	if err := userPool.QueryRow(ctx,
		"SELECT id, username, email FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Username, &user.Email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusBadRequest, M{"error": "login verification code expired or not found"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "authentication service unavailable"})
		return
	}

	result, err := consumeLoginVerificationCode(ctx, email, req.ChallengeID, requestClientIdentity(r), req.Code)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service unavailable"})
		return
	}
	switch result {
	case verificationExpired:
		writeJSON(w, http.StatusBadRequest, M{"error": "login verification code expired or not found"})
		return
	case verificationLocked:
		writeJSON(w, http.StatusTooManyRequests, M{"error": "too many login verification attempts; please start again"})
		return
	case verificationRejected:
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid login verification code"})
		return
	case verificationAccepted:
		// Continue. The single-use challenge was consumed atomically.
	default:
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "verification service unavailable"})
		return
	}

	clientIdentity := requestClientIdentity(r)
	if err := resetLoginAttempts(ctx, email, clientIdentity); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "authentication service unavailable"})
		return
	}

	sessionDuration := sessionTTL
	if req.Remember {
		sessionDuration = rememberSessionTTL
	}
	signed, err := CreateSessionWithTTL(ctx, &user, sessionDuration, req.Remember)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "create session failed"})
		return
	}

	http.SetCookie(w, sessionCookieForLogin(signed, sessionDuration, req.Remember))
	writeJSON(w, http.StatusOK, M{"user": clientUserFrom(&user)})
}

func sessionCookieForLogin(signed string, duration time.Duration, persistent bool) *http.Cookie {
	maxAge := 0
	if persistent {
		maxAge = int(duration.Seconds())
	}
	return &http.Cookie{
		Name:     "session",
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

// handleCurrentSession 回傳已驗證 session 對應的最小使用者資料，供前端恢復登入狀態。
func handleCurrentSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, M{"user": clientUserFrom(currentUser(r))})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	handleLogoutWithDestroyer(w, r, DestroySession)
}

type sessionDestroyer func(context.Context, string) error

func handleLogoutWithDestroyer(w http.ResponseWriter, r *http.Request, destroy sessionDestroyer) {
	var destroyErr error
	if cookie, err := r.Cookie("session"); err == nil {
		destroyErr = destroy(r.Context(), cookie.Value)
	}

	if destroyErr != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "session service unavailable"})
		return
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, M{"message": "logged out"})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
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
