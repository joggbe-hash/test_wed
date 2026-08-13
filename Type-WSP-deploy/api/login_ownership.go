package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"typewsp/shared/authstate"
	"typewsp/shared/contracts"
)

const (
	loginOwnershipVerifyAttemptsPrefix  = "login:ownership:verify-attempts:"
	loginOwnershipGrantPrefix           = "login:ownership:grant:"
	loginOwnershipClientHourlyPrefix    = "login:ownership:send-client-hourly:"
	loginOwnershipClientDailyPrefix     = "login:ownership:send-client-daily:"
	loginOwnershipCodeFormat            = "base32-16-v1"
	loginOwnershipRequiredCode          = "LOGIN_EMAIL_OWNERSHIP_REQUIRED"
	loginOwnershipChallengeTTL          = 24 * time.Hour
	loginOwnershipVerificationLimit     = 5
	loginOwnershipGrantTTL              = 5 * time.Minute
	loginOwnershipGrantMaxAttempts      = 3
	loginOwnershipSendCooldown          = time.Minute
	loginOwnershipHourlySendLimit       = 5
	loginOwnershipDailySendLimit        = 10
	loginOwnershipClientHourlySendLimit = 20
	loginOwnershipClientDailySendLimit  = 50
	passwordVerificationGrantBytes      = 32
	passwordVerificationGrantLength     = 43
)

var (
	errLoginOwnershipSendLimited = errors.New("login ownership email send limit reached")
	ensureLoginOwnershipScript   = redis.NewScript(`
local function keyType(key)
  local result = redis.call('TYPE', key)
  if type(result) == 'table' then
    return result['ok']
  end
  return result
end

local function requireType(key, expected)
  local actual = keyType(key)
  if actual ~= 'none' and actual ~= expected then
    return redis.error_reply('unexpected key type for ' .. key)
  end
end

local function canonicalNonnegativeInteger(raw)
  if not raw or not string.match(raw, '^%d+$') then
    return nil
  end
  local parsed = tonumber(raw)
  if not parsed or parsed < 0 or parsed ~= math.floor(parsed) or tostring(parsed) ~= raw then
    return nil
  end
  return parsed
end

requireType(KEYS[1], 'stream')
requireType(KEYS[2], 'hash')
for index = 3, 7 do
  requireType(KEYS[index], 'string')
end

local streamLength = redis.call('XLEN', KEYS[1])
local activeExists = redis.call('EXISTS', KEYS[2]) == 1
local challengeID = ARGV[3]
local code = ARGV[4]
local payload = ARGV[5]
local activeTTL = tonumber(ARGV[6])
local previousDeliveryID = ''
local previousDeliveryState = ''
local previousPayload = ''

if activeExists then
  challengeID = redis.call('HGET', KEYS[2], 'challenge_id')
  code = redis.call('HGET', KEYS[2], 'code')
  payload = redis.call('HGET', KEYS[2], 'payload')
  activeTTL = redis.call('PTTL', KEYS[2])
  if not challengeID or not code or not payload or activeTTL <= 0 then
    return redis.error_reply('invalid active login ownership challenge')
  end
  previousDeliveryID = redis.call('HGET', KEYS[2], 'delivery_id') or ''
  previousDeliveryState = redis.call('HGET', KEYS[2], 'delivery_state') or ''
  if previousDeliveryID == '' or (previousDeliveryState ~= ARGV[15] and previousDeliveryState ~= ARGV[16]) then
    return redis.error_reply('invalid active login ownership delivery state')
  end
  previousPayload = payload
  local decodedPayload = cjson.decode(payload)
  decodedPayload['delivery_id'] = ARGV[14]
  payload = cjson.encode(decodedPayload)
end

local counters = {}
for index = 4, 7 do
  local raw = redis.call('GET', KEYS[index])
  if raw then
    local parsed = canonicalNonnegativeInteger(raw)
    if not parsed or redis.call('PTTL', KEYS[index]) <= 0 then
      return redis.error_reply('invalid login ownership send counter')
    end
    counters[index] = parsed
  else
    counters[index] = 0
  end
end

if redis.call('EXISTS', KEYS[3]) == 1 then
  if redis.call('PTTL', KEYS[3]) <= 0 then
    return redis.error_reply('invalid login ownership send cooldown')
  end
  if activeExists then
    return {2, challengeID, code, activeTTL}
  end
  return {-1, '', '', 0}
end

local limited = counters[4] >= tonumber(ARGV[8])
  or counters[5] >= tonumber(ARGV[9])
  or counters[6] >= tonumber(ARGV[10])
  or counters[7] >= tonumber(ARGV[11])
if limited then
  if activeExists then
    return {2, challengeID, code, activeTTL}
  end
  return {-1, '', '', 0}
end

if streamLength >= tonumber(ARGV[1]) then
  return {-2, '', '', 0}
end

-- Every command after this point is type-safe and cannot fail for valid Redis
-- scalar inputs. XADD is deliberately last and uses pcall so a queue failure can
-- compensate every state mutation made by this invocation.
if not activeExists then
  redis.call('HSET', KEYS[2],
    'challenge_id', challengeID,
    'code', code,
    'payload', payload,
    'delivery_id', ARGV[14],
    'delivery_state', ARGV[15])
  redis.call('PEXPIRE', KEYS[2], ARGV[6])
else
  redis.call('HSET', KEYS[2],
    'payload', payload,
    'delivery_id', ARGV[14],
    'delivery_state', ARGV[15])
end
redis.call('SET', KEYS[3], '1', 'PX', ARGV[7])
for index = 4, 7 do
  local value = redis.call('INCR', KEYS[index])
  if value == 1 then
    if index == 4 or index == 6 then
      redis.call('PEXPIRE', KEYS[index], ARGV[12])
    else
      redis.call('PEXPIRE', KEYS[index], ARGV[13])
    end
  end
end
local added = redis.pcall('XADD', KEYS[1], '*', 'type', ARGV[2], 'payload', payload, 'attempts', '0')
if type(added) == 'table' and added['err'] then
  if not activeExists then
    redis.call('DEL', KEYS[2])
  else
    redis.call('HSET', KEYS[2],
      'payload', previousPayload,
      'delivery_id', previousDeliveryID,
      'delivery_state', previousDeliveryState)
  end
  redis.call('DEL', KEYS[3])
  for index = 4, 7 do
    local value = redis.call('DECR', KEYS[index])
    if value <= 0 then
      redis.call('DEL', KEYS[index])
    end
  end
  return redis.error_reply(added['err'])
end
return {1, challengeID, code, activeTTL}
`)
	exchangeLoginOwnershipCodeScript = redis.NewScript(`
local function keyType(key)
  local result = redis.call('TYPE', key)
  if type(result) == 'table' then return result['ok'] end
  return result
end
local activeType = keyType(KEYS[1])
local attemptType = keyType(KEYS[2])
local grantType = keyType(KEYS[3])
if activeType ~= 'none' and activeType ~= 'hash' then
  return redis.error_reply('unexpected login ownership challenge type')
end
if attemptType ~= 'none' and attemptType ~= 'string' then
  return redis.error_reply('unexpected login ownership attempt type')
end
if grantType ~= 'none' and grantType ~= 'string' then
  return redis.error_reply('unexpected password verification grant type')
end

local storedChallengeID = redis.call('HGET', KEYS[1], 'challenge_id')
local storedCode = redis.call('HGET', KEYS[1], 'code')
local activeTTL = redis.call('PTTL', KEYS[1])
if not storedChallengeID or not storedCode or activeTTL <= 0 or storedChallengeID ~= ARGV[1] then
  return -1
end

-- A correct 80-bit code always wins over the source-scoped wrong-guess budget.
if storedCode == ARGV[2] then
  if redis.call('EXISTS', KEYS[3]) == 1 then
    return redis.error_reply('password verification grant collision')
  end
  redis.call('SET', KEYS[3], ARGV[3], 'PX', ARGV[4])
  redis.call('DEL', KEYS[1], KEYS[2])
  return 1
end

local attempts = redis.call('GET', KEYS[2])
if attempts then
  attempts = tonumber(attempts)
  if not attempts or attempts < 0 or attempts ~= math.floor(attempts) then
    return redis.error_reply('invalid login ownership attempt counter')
  end
else
  attempts = 0
end
if attempts >= tonumber(ARGV[5]) then
  return 2
end
attempts = redis.call('INCR', KEYS[2])
if attempts == 1 then redis.call('PEXPIRE', KEYS[2], activeTTL) end
if attempts >= tonumber(ARGV[5]) then return 2 end
return 0
`)
	consumePasswordVerificationGrantScript = redis.NewScript(`
local function keyType(key)
  local result = redis.call('TYPE', key)
  if type(result) == 'table' then return result['ok'] end
  return result
end
local function canonicalNonnegativeInteger(raw)
  if not raw or not string.match(raw, '^%d+$') then
    return nil
  end
  local parsed = tonumber(raw)
  if not parsed or parsed < 0 or parsed ~= math.floor(parsed) or tostring(parsed) ~= raw then
    return nil
  end
  return parsed
end
for index = 1, 3 do
  local actual = keyType(KEYS[index])
  if actual ~= 'none' and actual ~= 'string' then
    return redis.error_reply('unexpected password verification admission key type')
  end
end

local clientAttempts = canonicalNonnegativeInteger(redis.call('GET', KEYS[2]) or '0')
local accountAttempts = canonicalNonnegativeInteger(redis.call('GET', KEYS[3]) or '0')
if not clientAttempts or not accountAttempts then
  return redis.error_reply('invalid login attempt counter')
end

local remainingUses = redis.call('GET', KEYS[1])
if not remainingUses or redis.call('PTTL', KEYS[1]) <= 0 then
  return {0, clientAttempts, accountAttempts, 0}
end
remainingUses = canonicalNonnegativeInteger(remainingUses)
if not remainingUses or remainingUses <= 0 then
  return redis.error_reply('invalid password verification grant')
end

remainingUses = remainingUses - 1
if remainingUses == 0 then
  redis.call('DEL', KEYS[1])
else
  redis.call('DECR', KEYS[1])
end
clientAttempts = redis.call('INCR', KEYS[2])
accountAttempts = redis.call('INCR', KEYS[3])
if clientAttempts == 1 or redis.call('PTTL', KEYS[2]) < 0 then
  redis.call('PEXPIRE', KEYS[2], ARGV[1])
end
if accountAttempts == 1 or redis.call('PTTL', KEYS[3]) < 0 then
  redis.call('PEXPIRE', KEYS[3], ARGV[2])
end
return {1, clientAttempts, accountAttempts, remainingUses}
`)
	revokePasswordVerificationGrantScript = redis.NewScript(`
local result = redis.call('TYPE', KEYS[1])
local actual = result
if type(result) == 'table' then actual = result['ok'] end
if actual ~= 'none' and actual ~= 'string' then
  return redis.error_reply('unexpected password verification grant type')
end
return redis.call('DEL', KEYS[1])
`)
)

type loginAdmission int64

const (
	loginAdmissionBlocked           loginAdmission = 0
	loginAdmissionAllowed           loginAdmission = 1
	loginAdmissionOwnershipRequired loginAdmission = 2
	loginAdmissionGrantCandidate    loginAdmission = 3
)

type ownershipVerificationResult int64

const (
	ownershipVerificationExpired  ownershipVerificationResult = -1
	ownershipVerificationRejected ownershipVerificationResult = 0
	ownershipVerificationAccepted ownershipVerificationResult = 1
	ownershipVerificationLocked   ownershipVerificationResult = 2
)

type loginOwnershipChallenge struct {
	ChallengeID      string `json:"challenge_id"`
	CodeFormat       string `json:"code_format"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

func generateLoginOwnershipCode() (string, error) {
	return generateRegistrationVerificationCode()
}

func validLoginOwnershipCode(code string) bool {
	return validRegistrationVerificationCode(code)
}

func generatePasswordVerificationGrant() (string, error) {
	raw := make([]byte, passwordVerificationGrantBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate password verification grant failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func validPasswordVerificationGrant(grant string) bool {
	if len(grant) != passwordVerificationGrantLength || strings.IndexFunc(grant, func(char rune) bool {
		return (char < 'A' || char > 'Z') &&
			(char < 'a' || char > 'z') &&
			(char < '0' || char > '9') && char != '-' && char != '_'
	}) >= 0 {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(grant)
	return err == nil && len(raw) == passwordVerificationGrantBytes && base64.RawURLEncoding.EncodeToString(raw) == grant
}

func loginOwnershipActiveKey(email string) string {
	return authstate.LoginOwnershipActiveKey(email)
}

func loginOwnershipVerificationAttemptKey(challengeID, clientIdentity string) string {
	return hashedValueKey(loginOwnershipVerifyAttemptsPrefix, challengeID+"\x00"+clientIdentity)
}

func loginOwnershipGrantKey(email, clientIdentity, grant string) string {
	return hashedValueKey(loginOwnershipGrantPrefix, strings.ToLower(strings.TrimSpace(email))+"\x00"+clientIdentity+"\x00"+grant)
}

func loginOwnershipSendCooldownKey(email string) string {
	return authstate.LoginOwnershipSendCooldownKey(email)
}

func loginOwnershipSendHourlyKey(email string) string {
	return authstate.LoginOwnershipSendHourlyKey(email)
}

func loginOwnershipSendDailyKey(email string) string {
	return authstate.LoginOwnershipSendDailyKey(email)
}

func loginOwnershipClientSendHourlyKey(clientIdentity string) string {
	return hashedValueKey(loginOwnershipClientHourlyPrefix, clientIdentity)
}

func loginOwnershipClientSendDailyKey(clientIdentity string) string {
	return hashedValueKey(loginOwnershipClientDailyPrefix, clientIdentity)
}

func ensureLoginOwnershipChallenge(ctx context.Context, email, clientIdentity string) (loginOwnershipChallenge, error) {
	challengeID := uuid.NewString()
	deliveryID := uuid.NewString()
	code, err := generateLoginOwnershipCode()
	if err != nil {
		return loginOwnershipChallenge{}, err
	}
	expiresAt := time.Now().Add(loginOwnershipChallengeTTL)
	payload, err := json.Marshal(contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              email,
		Code:               code,
		ChallengeID:        challengeID,
		DeliveryID:         deliveryID,
		ExpiresAtUnixMilli: expiresAt.UnixMilli(),
	})
	if err != nil {
		return loginOwnershipChallenge{}, fmt.Errorf("encode login ownership email payload failed: %w", err)
	}

	values, err := ensureLoginOwnershipScript.Run(
		ctx,
		rdb,
		[]string{
			contracts.TaskStreamKey,
			loginOwnershipActiveKey(email),
			loginOwnershipSendCooldownKey(email),
			loginOwnershipSendHourlyKey(email),
			loginOwnershipSendDailyKey(email),
			loginOwnershipClientSendHourlyKey(clientIdentity),
			loginOwnershipClientSendDailyKey(clientIdentity),
		},
		maxTaskStreamLength,
		contracts.TaskSendVerificationEmail,
		challengeID,
		code,
		string(payload),
		loginOwnershipChallengeTTL.Milliseconds(),
		loginOwnershipSendCooldown.Milliseconds(),
		loginOwnershipHourlySendLimit,
		loginOwnershipDailySendLimit,
		loginOwnershipClientHourlySendLimit,
		loginOwnershipClientDailySendLimit,
		time.Hour.Milliseconds(),
		(24 * time.Hour).Milliseconds(),
		deliveryID,
		contracts.LoginOwnershipDeliveryStateQueued,
		contracts.LoginOwnershipDeliveryStateDelivered,
	).Slice()
	if err != nil {
		return loginOwnershipChallenge{}, fmt.Errorf("ensure login ownership challenge failed: %w", err)
	}
	if len(values) != 4 {
		return loginOwnershipChallenge{}, fmt.Errorf("ensure login ownership challenge returned %d values", len(values))
	}
	status, err := redisCounter(values[0])
	if err != nil {
		return loginOwnershipChallenge{}, fmt.Errorf("decode login ownership challenge status: %w", err)
	}
	if status == -1 {
		return loginOwnershipChallenge{}, errLoginOwnershipSendLimited
	}
	if status == -2 {
		return loginOwnershipChallenge{}, errTaskQueueFull
	}
	if status != 1 && status != 2 {
		return loginOwnershipChallenge{}, fmt.Errorf("invalid login ownership challenge status %d", status)
	}
	resolvedChallengeID, ok := values[1].(string)
	if !ok || resolvedChallengeID == "" {
		return loginOwnershipChallenge{}, fmt.Errorf("invalid login ownership challenge id")
	}
	ttlMillis, err := redisCounter(values[3])
	if err != nil || ttlMillis <= 0 {
		return loginOwnershipChallenge{}, fmt.Errorf("invalid login ownership challenge TTL")
	}
	return loginOwnershipChallenge{
		ChallengeID:      resolvedChallengeID,
		CodeFormat:       loginOwnershipCodeFormat,
		ExpiresInSeconds: int((ttlMillis + 999) / 1000),
	}, nil
}

func exchangeLoginOwnershipCode(
	ctx context.Context,
	email, challengeID, clientIdentity, code, grant string,
) (ownershipVerificationResult, error) {
	result, err := exchangeLoginOwnershipCodeScript.Run(
		ctx,
		rdb,
		[]string{
			loginOwnershipActiveKey(email),
			loginOwnershipVerificationAttemptKey(challengeID, clientIdentity),
			loginOwnershipGrantKey(email, clientIdentity, grant),
		},
		challengeID,
		code,
		loginOwnershipGrantMaxAttempts,
		loginOwnershipGrantTTL.Milliseconds(),
		loginOwnershipVerificationLimit,
	).Int64()
	if err != nil {
		return ownershipVerificationExpired, fmt.Errorf("exchange login ownership code failed: %w", err)
	}
	verificationResult := ownershipVerificationResult(result)
	switch verificationResult {
	case ownershipVerificationExpired,
		ownershipVerificationRejected,
		ownershipVerificationAccepted,
		ownershipVerificationLocked:
		return verificationResult, nil
	default:
		return ownershipVerificationExpired, fmt.Errorf("invalid login ownership verification result %d", result)
	}
}

func consumePasswordVerificationGrantAndReserveAttempt(
	ctx context.Context,
	email, clientIdentity, grant string,
) (bool, int64, int64, int64, error) {
	if !validPasswordVerificationGrant(grant) {
		return false, 0, 0, 0, nil
	}
	values, err := consumePasswordVerificationGrantScript.Run(
		ctx,
		rdb,
		[]string{
			loginOwnershipGrantKey(email, clientIdentity, grant),
			loginAttemptKey(email, clientIdentity),
			accountLoginAttemptKey(email),
		},
		loginAttemptWindow.Milliseconds(),
		accountLoginAttemptWindow.Milliseconds(),
	).Slice()
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("consume password verification grant failed: %w", err)
	}
	if len(values) != 4 {
		return false, 0, 0, 0, fmt.Errorf("consume password verification grant returned %d values", len(values))
	}
	decoded := make([]int64, len(values))
	for index, value := range values {
		decoded[index], err = redisCounter(value)
		if err != nil {
			return false, 0, 0, 0, fmt.Errorf("decode password verification grant result %d: %w", index, err)
		}
	}
	if decoded[0] != 0 && decoded[0] != 1 {
		return false, 0, 0, 0, fmt.Errorf("invalid password verification grant status %d", decoded[0])
	}
	return decoded[0] == 1, decoded[1], decoded[2], decoded[3], nil
}

func revokePasswordVerificationGrant(ctx context.Context, email, clientIdentity, grant string) error {
	if !validPasswordVerificationGrant(grant) {
		return nil
	}
	if err := revokePasswordVerificationGrantScript.Run(
		ctx,
		rdb,
		[]string{loginOwnershipGrantKey(email, clientIdentity, grant)},
	).Err(); err != nil {
		return fmt.Errorf("revoke password verification grant failed: %w", err)
	}
	return nil
}

func writeLoginOwnershipRequired(w http.ResponseWriter, ctx context.Context, email, clientIdentity string) {
	w.Header().Set("Cache-Control", "no-store")
	challenge, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "authentication service unavailable"})
		return
	}
	writeJSON(w, http.StatusTooManyRequests, M{
		"error":               "email ownership verification required",
		"code":                loginOwnershipRequiredCode,
		"ownership_challenge": challenge,
	})
}

type loginOwnershipVerificationRequest struct {
	Email       string `json:"email"`
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

func handleLoginOwnershipVerification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	limitAuthRequestBody(w, r)
	var req loginOwnershipVerificationRequest
	if err := readJSON(r, &req); err != nil || req.Email == "" || req.ChallengeID == "" || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "email, challenge_id and code are required"})
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil || !validLoginOwnershipCode(req.Code) {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid email ownership challenge"})
		return
	}
	parsedChallengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil || parsedChallengeID.String() != req.ChallengeID {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid email ownership challenge"})
		return
	}

	grant, err := generatePasswordVerificationGrant()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "generate password verification grant failed"})
		return
	}
	result, err := exchangeLoginOwnershipCode(
		r.Context(),
		email,
		req.ChallengeID,
		requestClientIdentity(r),
		req.Code,
		grant,
	)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "authentication service unavailable"})
		return
	}
	switch result {
	case ownershipVerificationExpired:
		writeJSON(w, http.StatusBadRequest, M{"error": "email ownership challenge expired or not found"})
	case ownershipVerificationRejected:
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid email ownership code"})
	case ownershipVerificationLocked:
		writeJSON(w, http.StatusTooManyRequests, M{"error": "too many email ownership verification attempts"})
	case ownershipVerificationAccepted:
		writeJSON(w, http.StatusOK, M{
			"password_verification_grant": grant,
			"expires_in_seconds":          int(loginOwnershipGrantTTL.Seconds()),
			"max_attempts":                loginOwnershipGrantMaxAttempts,
		})
	default:
		writeJSON(w, http.StatusServiceUnavailable, M{"error": "authentication service unavailable"})
	}
}
