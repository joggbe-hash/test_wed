package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

func TestLoginOwnershipSecretsHaveRequiredEntropyAndFormat(t *testing.T) {
	for range 32 {
		code, err := generateLoginOwnershipCode()
		if err != nil {
			t.Fatalf("generate ownership code: %v", err)
		}
		if !validLoginOwnershipCode(code) {
			t.Fatalf("generated ownership code %q is invalid", code)
		}

		grant, err := generatePasswordVerificationGrant()
		if err != nil {
			t.Fatalf("generate password verification grant: %v", err)
		}
		raw, err := base64.RawURLEncoding.DecodeString(grant)
		if err != nil || len(raw) != 32 {
			t.Fatalf("grant decodes to %d bytes with error %v, want 32 bytes", len(raw), err)
		}
	}
}

func TestPasswordVerificationGrantValidationRejectsMalformedAndOversizedInput(t *testing.T) {
	for name, grant := range map[string]string{
		"empty":       "",
		"short":       strings.Repeat("a", passwordVerificationGrantLength-1),
		"long":        strings.Repeat("a", passwordVerificationGrantLength+1),
		"oversized":   strings.Repeat("a", maxAuthRequestBytes),
		"padding":     strings.Repeat("a", passwordVerificationGrantLength-1) + "=",
		"non-urlsafe": strings.Repeat("a", passwordVerificationGrantLength-1) + "+",
	} {
		t.Run(name, func(t *testing.T) {
			if validPasswordVerificationGrant(grant) {
				t.Fatalf("invalid grant was accepted: %q", grant)
			}
		})
	}
}

func TestLoginOwnershipVerificationHandlerReturnsFixedGrantContract(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "handler-owner@example.com"
	clientIdentity := "198.51.100.60"
	challenge, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity)
	if err != nil {
		t.Fatalf("ensure ownership challenge: %v", err)
	}
	code, err := client.HGet(ctx, loginOwnershipActiveKey(email), "code").Result()
	if err != nil {
		t.Fatalf("load ownership code: %v", err)
	}
	body, err := json.Marshal(loginOwnershipVerificationRequest{
		Email:       email,
		ChallengeID: challenge.ChallengeID,
		Code:        code,
	})
	if err != nil {
		t.Fatalf("encode verification request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login/ownership/verify", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIdentity + ":49152"
	response := httptest.NewRecorder()

	handleLoginOwnershipVerification(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Grant            string `json:"password_verification_grant"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
		MaxAttempts      int    `json:"max_attempts"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode grant response: %v", err)
	}
	if !validPasswordVerificationGrant(result.Grant) || result.ExpiresInSeconds != 300 || result.MaxAttempts != 3 {
		t.Fatalf("grant response = %#v", result)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
}

func TestLoginOwnershipVerificationRejectsMalformedBoundaryValues(t *testing.T) {
	useIntegrationRedis(t)
	for name, body := range map[string]string{
		"non-canonical challenge id": `{"email":"user@example.com","challenge_id":"2CD53940-FC0D-4972-921B-086061DDE6E5","code":"ABCDEFGHJKLMNPQ2"}`,
		"invalid code alphabet":      `{"email":"user@example.com","challenge_id":"2cd53940-fc0d-4972-921b-086061dde6e5","code":"ABCDEFGHIJKLMNO1"}`,
		"lowercase code":             `{"email":"user@example.com","challenge_id":"2cd53940-fc0d-4972-921b-086061dde6e5","code":"abcdefghjklmnpq2"}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/auth/login/ownership/verify", bytes.NewBufferString(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handleLoginOwnershipVerification(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestLoginOwnershipVerificationWrongCodeResponseIsNeverCached(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "wrong-code-owner@example.com"
	clientIdentity := "198.51.100.65"
	challenge, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity)
	if err != nil {
		t.Fatalf("ensure ownership challenge: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login/ownership/verify", bytes.NewBufferString(fmt.Sprintf(
		`{"email":%q,"challenge_id":%q,"code":"AAAAAAAAAAAAAAAA"}`,
		email,
		challenge.ChallengeID,
	)))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIdentity + ":49152"
	response := httptest.NewRecorder()
	handleLoginOwnershipVerification(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
}

func TestLoginHandlerRequiresOwnershipBeforeDatabaseWork(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "threshold-owner@example.com"
	clientIdentity := "198.51.100.61"
	if err := client.Set(ctx, accountLoginAttemptKey(email), accountLoginAttemptLimit, accountLoginAttemptWindow).Err(); err != nil {
		t.Fatalf("prime account attempts: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"threshold-owner@example.com","password":"Password1"}`))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIdentity + ":49152"
	response := httptest.NewRecorder()
	lookupCalled := false

	handleLoginWithUserLookup(response, request, func(context.Context, string) (User, string) {
		lookupCalled = true
		return User{}, ""
	})

	if lookupCalled {
		t.Fatal("ownership-required login reached the database lookup")
	}
	assertOwnershipRequiredResponse(t, response)
}

func TestLoginHandlerBusyDoesNotConsumePasswordVerificationGrant(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "busy-owner@example.com"
	clientIdentity := "198.51.100.62"
	grant, err := generatePasswordVerificationGrant()
	if err != nil {
		t.Fatalf("generate grant: %v", err)
	}
	grantKey := loginOwnershipGrantKey(email, clientIdentity, grant)
	if err := client.Set(ctx, grantKey, loginOwnershipGrantMaxAttempts, loginOwnershipGrantTTL).Err(); err != nil {
		t.Fatalf("store grant: %v", err)
	}
	previousLimiter := loginPasswordVerificationConcurrency
	loginPasswordVerificationConcurrency = newConcurrencyLimiter(1)
	if !loginPasswordVerificationConcurrency.tryAcquire() {
		t.Fatal("occupy bcrypt lane")
	}
	t.Cleanup(func() { loginPasswordVerificationConcurrency = previousLimiter })

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(fmt.Sprintf(`{"email":%q,"password":"Password1","password_verification_grant":%q}`, email, grant)))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIdentity + ":49152"
	response := httptest.NewRecorder()
	lookupCalled := false
	handleLoginWithUserLookup(response, request, func(context.Context, string) (User, string) {
		lookupCalled = true
		return User{}, ""
	})

	if response.Code != http.StatusTooManyRequests || lookupCalled {
		t.Fatalf("busy response = %d lookup=%v body=%s", response.Code, lookupCalled, response.Body.String())
	}
	if remaining, err := client.Get(ctx, grantKey).Int64(); err != nil || remaining != loginOwnershipGrantMaxAttempts {
		t.Fatalf("busy response consumed grant: remaining=%d error=%v", remaining, err)
	}
}

func TestLoginHandlerTwentyFifthWrongPasswordImmediatelyReturnsOwnershipChallenge(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "boundary-owner@example.com"
	clientIdentity := "198.51.100.63"
	if err := client.Set(ctx, accountLoginAttemptKey(email), accountLoginAttemptLimit-1, accountLoginAttemptWindow).Err(); err != nil {
		t.Fatalf("prime account attempts: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"boundary-owner@example.com","password":"wrong-password"}`))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIdentity + ":49152"
	response := httptest.NewRecorder()
	lookupCalls := 0
	handleLoginWithUserLookup(response, request, func(context.Context, string) (User, string) {
		lookupCalls++
		return User{}, ""
	})

	if lookupCalls != 1 {
		t.Fatalf("database lookup calls = %d, want 1", lookupCalls)
	}
	assertOwnershipRequiredResponse(t, response)
}

func TestLoginHandlerExhaustedGrantDoesNotReturnPhantomChallengeDuringCooldown(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "exhausted-owner@example.com"
	clientIdentity := "198.51.100.64"
	if _, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity); err != nil {
		t.Fatalf("ensure ownership challenge: %v", err)
	}
	if err := client.Del(ctx, loginOwnershipActiveKey(email)).Err(); err != nil {
		t.Fatalf("consume active challenge: %v", err)
	}
	if err := client.Set(ctx, accountLoginAttemptKey(email), accountLoginAttemptLimit, accountLoginAttemptWindow).Err(); err != nil {
		t.Fatalf("prime account attempts: %v", err)
	}
	grant, err := generatePasswordVerificationGrant()
	if err != nil {
		t.Fatalf("generate exhausted grant: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(fmt.Sprintf(`{"email":%q,"password":"Password1","password_verification_grant":%q}`, email, grant)))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIdentity + ":49152"
	response := httptest.NewRecorder()

	handleLoginWithUserLookup(response, request, func(context.Context, string) (User, string) {
		t.Fatal("exhausted grant must not reach database lookup")
		return User{}, ""
	})

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", response.Code, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte("ownership_challenge")) {
		t.Fatalf("response exposed a phantom ownership challenge: %s", response.Body.String())
	}
}

func assertOwnershipRequiredResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429; body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Code      string                  `json:"code"`
		Challenge loginOwnershipChallenge `json:"ownership_challenge"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode ownership response: %v", err)
	}
	if result.Code != loginOwnershipRequiredCode || result.Challenge.ChallengeID == "" ||
		result.Challenge.CodeFormat != "base32-16-v1" || result.Challenge.ExpiresInSeconds <= 0 {
		t.Fatalf("ownership response = %#v", result)
	}
}

func TestRedisCorrectOwnershipCodeOverridesWrongGuessBudgetAndCreatesBoundGrant(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "verified-owner@example.com"
	clientIdentity := "198.51.100.30"
	challenge, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity)
	if err != nil {
		t.Fatalf("ensure ownership challenge: %v", err)
	}
	code, err := client.HGet(ctx, loginOwnershipActiveKey(email), "code").Result()
	if err != nil {
		t.Fatalf("load ownership code: %v", err)
	}

	for attempt := 1; attempt <= loginOwnershipVerificationLimit; attempt++ {
		candidate, err := generatePasswordVerificationGrant()
		if err != nil {
			t.Fatalf("generate rejected grant: %v", err)
		}
		result, err := exchangeLoginOwnershipCode(ctx, email, challenge.ChallengeID, clientIdentity, "AAAAAAAAAAAAAAAA", candidate)
		if err != nil {
			t.Fatalf("wrong code attempt %d: %v", attempt, err)
		}
		if attempt < loginOwnershipVerificationLimit && result != ownershipVerificationRejected {
			t.Fatalf("wrong code attempt %d result = %d, want rejected", attempt, result)
		}
		if attempt == loginOwnershipVerificationLimit && result != ownershipVerificationLocked {
			t.Fatalf("wrong code attempt %d result = %d, want locked", attempt, result)
		}
	}

	grant, err := generatePasswordVerificationGrant()
	if err != nil {
		t.Fatalf("generate accepted grant: %v", err)
	}
	result, err := exchangeLoginOwnershipCode(ctx, email, challenge.ChallengeID, clientIdentity, code, grant)
	if err != nil {
		t.Fatalf("exchange correct ownership code: %v", err)
	}
	if result != ownershipVerificationAccepted {
		t.Fatalf("correct code result = %d, want accepted", result)
	}
	remaining, err := client.Get(ctx, loginOwnershipGrantKey(email, clientIdentity, grant)).Int64()
	if err != nil || remaining != loginOwnershipGrantMaxAttempts {
		t.Fatalf("grant uses = %d, error %v; want %d", remaining, err, loginOwnershipGrantMaxAttempts)
	}
	if ttl, err := client.PTTL(ctx, loginOwnershipGrantKey(email, clientIdentity, grant)).Result(); err != nil || ttl <= 0 || ttl > loginOwnershipGrantTTL {
		t.Fatalf("grant TTL = %s, error %v", ttl, err)
	}
	if exists, err := client.Exists(ctx, loginOwnershipActiveKey(email)).Result(); err != nil || exists != 0 {
		t.Fatalf("active challenge exists after exchange = %d, error %v", exists, err)
	}
	if exists, err := client.Exists(ctx, loginOwnershipGrantKey(email, "198.51.100.31", grant)).Result(); err != nil || exists != 0 {
		t.Fatalf("grant was usable from another client: exists=%d error=%v", exists, err)
	}
	if exists, err := client.Exists(ctx, loginOwnershipGrantKey("other@example.com", clientIdentity, grant)).Result(); err != nil || exists != 0 {
		t.Fatalf("grant was usable for another email: exists=%d error=%v", exists, err)
	}
}

func TestRedisPasswordVerificationGrantBypassesLimitsAndAllowsExactlyThreeBcryptAdmissions(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "grant-owner@example.com"
	clientIdentity := "198.51.100.40"
	grant, err := generatePasswordVerificationGrant()
	if err != nil {
		t.Fatalf("generate grant: %v", err)
	}
	if err := client.Set(ctx, loginOwnershipGrantKey(email, clientIdentity, grant), loginOwnershipGrantMaxAttempts, loginOwnershipGrantTTL).Err(); err != nil {
		t.Fatalf("store grant: %v", err)
	}
	if err := client.Set(ctx, loginAttemptKey(email, clientIdentity), loginAttemptLimit, loginAttemptWindow).Err(); err != nil {
		t.Fatalf("prime client attempts: %v", err)
	}
	if err := client.Set(ctx, accountLoginAttemptKey(email), accountLoginAttemptLimit, accountLoginAttemptWindow).Err(); err != nil {
		t.Fatalf("prime account attempts: %v", err)
	}

	admission, _, _, err := reserveLoginAttempt(ctx, email, clientIdentity, grant)
	if err != nil {
		t.Fatalf("classify valid grant: %v", err)
	}
	if admission != loginAdmissionGrantCandidate {
		t.Fatalf("valid grant admission = %d, want candidate", admission)
	}

	const replays = 32
	type consumeResult struct {
		allowed bool
		err     error
	}
	results := make(chan consumeResult, replays)
	var group sync.WaitGroup
	for range replays {
		group.Add(1)
		go func() {
			defer group.Done()
			allowed, _, _, _, err := consumePasswordVerificationGrantAndReserveAttempt(ctx, email, clientIdentity, grant)
			results <- consumeResult{allowed: allowed, err: err}
		}()
	}
	group.Wait()
	close(results)

	allowed := 0
	for current := range results {
		if current.err != nil {
			t.Fatalf("consume grant: %v", current.err)
		}
		if current.allowed {
			allowed++
		}
	}
	if allowed != loginOwnershipGrantMaxAttempts {
		t.Fatalf("allowed grant replays = %d, want %d", allowed, loginOwnershipGrantMaxAttempts)
	}
	if exists, err := client.Exists(ctx, loginOwnershipGrantKey(email, clientIdentity, grant)).Result(); err != nil || exists != 0 {
		t.Fatalf("exhausted grant remains: exists=%d error=%v", exists, err)
	}
	accountAttempts, err := client.Get(ctx, accountLoginAttemptKey(email)).Int64()
	if err != nil || accountAttempts != accountLoginAttemptLimit+loginOwnershipGrantMaxAttempts {
		t.Fatalf("account attempts = %d, error %v", accountAttempts, err)
	}
	clientAttempts, err := client.Get(ctx, loginAttemptKey(email, clientIdentity)).Int64()
	if err != nil || clientAttempts != loginAttemptLimit+loginOwnershipGrantMaxAttempts {
		t.Fatalf("client attempts = %d, error %v", clientAttempts, err)
	}
}

func TestPasswordVerificationGrantKeyBindsEverySecurityPrincipal(t *testing.T) {
	grant := base64.RawURLEncoding.EncodeToString(make([]byte, passwordVerificationGrantBytes))
	base := loginOwnershipGrantKey("user@example.com", "198.51.100.1", grant)
	for name, other := range map[string]string{
		"email":  loginOwnershipGrantKey("other@example.com", "198.51.100.1", grant),
		"client": loginOwnershipGrantKey("user@example.com", "198.51.100.2", grant),
		"token":  loginOwnershipGrantKey("user@example.com", "198.51.100.1", fmt.Sprintf("%sx", grant)),
	} {
		if other == base {
			t.Fatalf("changing %s did not change grant key", name)
		}
	}
}

func TestRedisRevokePasswordVerificationGrantRemovesRemainingUses(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "successful-owner@example.com"
	clientIdentity := "198.51.100.50"
	grant, err := generatePasswordVerificationGrant()
	if err != nil {
		t.Fatalf("generate grant: %v", err)
	}
	key := loginOwnershipGrantKey(email, clientIdentity, grant)
	if err := client.Set(ctx, key, loginOwnershipGrantMaxAttempts-1, loginOwnershipGrantTTL).Err(); err != nil {
		t.Fatalf("store remaining grant: %v", err)
	}
	if err := revokePasswordVerificationGrant(ctx, email, clientIdentity, grant); err != nil {
		t.Fatalf("revoke grant: %v", err)
	}
	if exists, err := client.Exists(ctx, key).Result(); err != nil || exists != 0 {
		t.Fatalf("revoked grant remains: exists=%d error=%v", exists, err)
	}
}

func TestRedisEnsureLoginOwnershipChallengeQueueFailureLeavesNoPhantomState(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "queue-failure@example.com"
	clientIdentity := "198.51.100.20"
	if err := client.Set(ctx, contracts.TaskStreamKey, "wrong type", 0).Err(); err != nil {
		t.Fatalf("poison task stream: %v", err)
	}

	if _, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity); err == nil {
		t.Fatal("WRONGTYPE task stream was accepted")
	}
	assertNoLoginOwnershipIssuanceState(t, ctx, client, email, clientIdentity)

	if err := client.Del(ctx, contracts.TaskStreamKey).Err(); err != nil {
		t.Fatalf("clear poisoned task stream: %v", err)
	}
	for index := 0; index < maxTaskStreamLength; index++ {
		if err := client.XAdd(ctx, &redis.XAddArgs{Stream: contracts.TaskStreamKey, Values: map[string]any{"type": "test"}}).Err(); err != nil {
			t.Fatalf("fill task stream at %d: %v", index, err)
		}
	}
	if _, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity); !errors.Is(err, errTaskQueueFull) {
		t.Fatalf("full stream error = %v, want %v", err, errTaskQueueFull)
	}
	assertNoLoginOwnershipIssuanceState(t, ctx, client, email, clientIdentity)
}

func TestRedisEnsureLoginOwnershipChallengeRejectsNonCanonicalCountersWithoutPartialState(t *testing.T) {
	counterKeys := map[string]func(string, string) string{
		"recipient hourly": func(email, _ string) string { return loginOwnershipSendHourlyKey(email) },
		"recipient daily":  func(email, _ string) string { return loginOwnershipSendDailyKey(email) },
		"client hourly":    func(_, clientIdentity string) string { return loginOwnershipClientSendHourlyKey(clientIdentity) },
		"client daily":     func(_, clientIdentity string) string { return loginOwnershipClientSendDailyKey(clientIdentity) },
	}
	for name, counterKey := range counterKeys {
		t.Run(name, func(t *testing.T) {
			client := useIntegrationRedis(t)
			ctx := context.Background()
			email := "noncanonical-counter@example.com"
			clientIdentity := "198.51.100.70"
			poisonedKey := counterKey(email, clientIdentity)
			if err := client.Set(ctx, poisonedKey, "2.0", time.Hour).Err(); err != nil {
				t.Fatalf("store non-canonical counter: %v", err)
			}

			if _, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity); err == nil {
				t.Fatal("non-canonical ownership send counter was accepted")
			}
			if value, err := client.Get(ctx, poisonedKey).Result(); err != nil || value != "2.0" {
				t.Fatalf("poisoned counter = %q, error %v; want unchanged 2.0", value, err)
			}
			for _, key := range []string{
				loginOwnershipActiveKey(email),
				loginOwnershipSendCooldownKey(email),
			} {
				if exists, err := client.Exists(ctx, key).Result(); err != nil || exists != 0 {
					t.Fatalf("partial ownership state %q exists = %d, error %v", key, exists, err)
				}
			}
			if count, err := client.XLen(ctx, contracts.TaskStreamKey).Result(); err != nil || count != 0 {
				t.Fatalf("partial ownership task count = %d, error %v; want 0", count, err)
			}
		})
	}
}

func TestRedisConsumePasswordVerificationGrantRejectsNonCanonicalAttemptsWithoutConsumingGrant(t *testing.T) {
	for _, poisonedCounter := range []string{"client", "account"} {
		t.Run(poisonedCounter, func(t *testing.T) {
			client := useIntegrationRedis(t)
			ctx := context.Background()
			email := "noncanonical-attempts@example.com"
			clientIdentity := "198.51.100.71"
			grant, err := generatePasswordVerificationGrant()
			if err != nil {
				t.Fatalf("generate password verification grant: %v", err)
			}
			grantKey := loginOwnershipGrantKey(email, clientIdentity, grant)
			clientAttemptsKey := loginAttemptKey(email, clientIdentity)
			accountAttemptsKey := accountLoginAttemptKey(email)
			if err := client.Set(ctx, grantKey, loginOwnershipGrantMaxAttempts, loginOwnershipGrantTTL).Err(); err != nil {
				t.Fatalf("store grant: %v", err)
			}
			clientAttempts := "4"
			accountAttempts := "7"
			if poisonedCounter == "client" {
				clientAttempts = "2.0"
			} else {
				accountAttempts = "2.0"
			}
			if err := client.Set(ctx, clientAttemptsKey, clientAttempts, loginAttemptWindow).Err(); err != nil {
				t.Fatalf("store client attempts: %v", err)
			}
			if err := client.Set(ctx, accountAttemptsKey, accountAttempts, accountLoginAttemptWindow).Err(); err != nil {
				t.Fatalf("store account attempts: %v", err)
			}

			if _, _, _, _, err := consumePasswordVerificationGrantAndReserveAttempt(ctx, email, clientIdentity, grant); err == nil {
				t.Fatal("non-canonical login attempt counter was accepted")
			}
			if remaining, err := client.Get(ctx, grantKey).Result(); err != nil || remaining != "3" {
				t.Fatalf("grant remaining uses = %q, error %v; want unchanged 3", remaining, err)
			}
			if actual, err := client.Get(ctx, clientAttemptsKey).Result(); err != nil || actual != clientAttempts {
				t.Fatalf("client attempts = %q, error %v; want unchanged %q", actual, err, clientAttempts)
			}
			if actual, err := client.Get(ctx, accountAttemptsKey).Result(); err != nil || actual != accountAttempts {
				t.Fatalf("account attempts = %q, error %v; want unchanged %q", actual, err, accountAttempts)
			}
		})
	}
}

func TestRedisConsumePasswordVerificationGrantRejectsNonCanonicalGrantWithoutCounterMutation(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "noncanonical-grant@example.com"
	clientIdentity := "198.51.100.72"
	grant, err := generatePasswordVerificationGrant()
	if err != nil {
		t.Fatalf("generate password verification grant: %v", err)
	}
	grantKey := loginOwnershipGrantKey(email, clientIdentity, grant)
	clientAttemptsKey := loginAttemptKey(email, clientIdentity)
	accountAttemptsKey := accountLoginAttemptKey(email)
	if err := client.Set(ctx, grantKey, "1.0", loginOwnershipGrantTTL).Err(); err != nil {
		t.Fatalf("store non-canonical grant: %v", err)
	}
	if err := client.Set(ctx, clientAttemptsKey, "4", loginAttemptWindow).Err(); err != nil {
		t.Fatalf("store client attempts: %v", err)
	}
	if err := client.Set(ctx, accountAttemptsKey, "7", accountLoginAttemptWindow).Err(); err != nil {
		t.Fatalf("store account attempts: %v", err)
	}

	if _, _, _, _, err := consumePasswordVerificationGrantAndReserveAttempt(ctx, email, clientIdentity, grant); err == nil {
		t.Fatal("non-canonical password verification grant was accepted")
	}
	if remaining, err := client.Get(ctx, grantKey).Result(); err != nil || remaining != "1.0" {
		t.Fatalf("grant remaining uses = %q, error %v; want unchanged 1.0", remaining, err)
	}
	if actual, err := client.Get(ctx, clientAttemptsKey).Result(); err != nil || actual != "4" {
		t.Fatalf("client attempts = %q, error %v; want unchanged 4", actual, err)
	}
	if actual, err := client.Get(ctx, accountAttemptsKey).Result(); err != nil || actual != "7" {
		t.Fatalf("account attempts = %q, error %v; want unchanged 7", actual, err)
	}
}

func assertNoLoginOwnershipIssuanceState(t *testing.T, ctx context.Context, client *redis.Client, email, clientIdentity string) {
	t.Helper()
	keys := []string{
		loginOwnershipActiveKey(email),
		loginOwnershipSendCooldownKey(email),
		loginOwnershipSendHourlyKey(email),
		loginOwnershipSendDailyKey(email),
		loginOwnershipClientSendHourlyKey(clientIdentity),
		loginOwnershipClientSendDailyKey(clientIdentity),
	}
	if count, err := client.Exists(ctx, keys...).Result(); err != nil {
		t.Fatalf("check phantom ownership state: %v", err)
	} else if count != 0 {
		t.Fatalf("phantom ownership keys remaining: %d", count)
	}
}

func TestRedisEnsureLoginOwnershipChallengeCreatesOneAtomicEmailTask(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "owner@example.com"
	clientIdentity := "198.51.100.10"

	challenge, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity)
	if err != nil {
		t.Fatalf("ensure ownership challenge: %v", err)
	}
	if challenge.ChallengeID == "" || challenge.CodeFormat != loginOwnershipCodeFormat {
		t.Fatalf("challenge = %#v", challenge)
	}
	if challenge.ExpiresInSeconds <= 0 || challenge.ExpiresInSeconds > int(loginOwnershipChallengeTTL.Seconds()) {
		t.Fatalf("expires_in_seconds = %d", challenge.ExpiresInSeconds)
	}

	active, err := client.HGetAll(ctx, loginOwnershipActiveKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge: %v", err)
	}
	if active["challenge_id"] != challenge.ChallengeID || !validLoginOwnershipCode(active["code"]) {
		t.Fatalf("active challenge = %#v", active)
	}
	if _, err := uuid.Parse(active["delivery_id"]); err != nil || active["delivery_state"] != "queued" {
		t.Fatalf("active delivery = %#v, UUID error %v", active, err)
	}

	messages, err := client.XRange(ctx, contracts.TaskStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("read ownership email task: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("task count = %d, want 1", len(messages))
	}
	message := messages[0].Values
	if message["type"] != contracts.TaskSendVerificationEmail || message["attempts"] != "0" {
		t.Fatalf("task fields = %#v", message)
	}
	var payload contracts.LoginOwnershipEmailPayload
	if err := json.Unmarshal([]byte(message["payload"].(string)), &payload); err != nil {
		t.Fatalf("decode ownership email payload: %v", err)
	}
	if payload.Purpose != contracts.EmailPurposeLoginOwnership || payload.Email != email || payload.ChallengeID != challenge.ChallengeID || payload.Code != active["code"] {
		t.Fatalf("email payload = %#v, active = %#v", payload, active)
	}
	if payload.DeliveryID != active["delivery_id"] {
		t.Fatalf("email delivery = %q, active delivery = %q", payload.DeliveryID, active["delivery_id"])
	}
	if payload.ExpiresAtUnixMilli <= time.Now().UnixMilli() {
		t.Fatalf("email payload already expired: %#v", payload)
	}
}

func TestRedisEnsureLoginOwnershipChallengeIsStableAcrossConcurrentCallers(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	const callers = 64
	email := "concurrent-owner@example.com"
	type result struct {
		challenge loginOwnershipChallenge
		err       error
	}
	results := make(chan result, callers)
	var group sync.WaitGroup
	for index := range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			challenge, err := ensureLoginOwnershipChallenge(ctx, email, fmt.Sprintf("198.51.100.%d", index))
			results <- result{challenge: challenge, err: err}
		}()
	}
	group.Wait()
	close(results)

	challengeID := ""
	for current := range results {
		if current.err != nil {
			t.Fatalf("ensure concurrent challenge: %v", current.err)
		}
		if challengeID == "" {
			challengeID = current.challenge.ChallengeID
		}
		if current.challenge.ChallengeID != challengeID {
			t.Fatalf("concurrent challenge IDs differ: %q and %q", challengeID, current.challenge.ChallengeID)
		}
	}
	if count, err := client.XLen(ctx, contracts.TaskStreamKey).Result(); err != nil || count != 1 {
		t.Fatalf("concurrent ownership tasks = %d, error %v; want 1", count, err)
	}
}

func TestRedisEnsureLoginOwnershipChallengeKeepsStableChallengeAndTTL(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "stable-owner@example.com"
	clientIdentity := "198.51.100.11"

	first, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity)
	if err != nil {
		t.Fatalf("ensure first challenge: %v", err)
	}
	firstTTL, err := client.PTTL(ctx, loginOwnershipActiveKey(email)).Result()
	if err != nil {
		t.Fatalf("load first TTL: %v", err)
	}
	if err := client.PExpire(ctx, loginOwnershipActiveKey(email), 10*time.Minute).Err(); err != nil {
		t.Fatalf("shorten active TTL: %v", err)
	}

	second, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity)
	if err != nil {
		t.Fatalf("ensure during cooldown: %v", err)
	}
	if second.ChallengeID != first.ChallengeID {
		t.Fatalf("challenge changed during cooldown: %q -> %q", first.ChallengeID, second.ChallengeID)
	}
	if count, err := client.XLen(ctx, contracts.TaskStreamKey).Result(); err != nil || count != 1 {
		t.Fatalf("task count during cooldown = %d, error %v; want 1", count, err)
	}

	if err := client.Del(ctx, loginOwnershipSendCooldownKey(email)).Err(); err != nil {
		t.Fatalf("advance resend cooldown: %v", err)
	}
	third, err := ensureLoginOwnershipChallenge(ctx, email, "198.51.100.12")
	if err != nil {
		t.Fatalf("resend ownership challenge: %v", err)
	}
	if third.ChallengeID != first.ChallengeID {
		t.Fatalf("challenge changed on resend: %q -> %q", first.ChallengeID, third.ChallengeID)
	}
	if count, err := client.XLen(ctx, contracts.TaskStreamKey).Result(); err != nil || count != 2 {
		t.Fatalf("task count after resend = %d, error %v; want 2", count, err)
	}
	messages, err := client.XRange(ctx, contracts.TaskStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("read resend tasks: %v", err)
	}
	var firstPayload, resendPayload contracts.LoginOwnershipEmailPayload
	if err := json.Unmarshal([]byte(messages[0].Values["payload"].(string)), &firstPayload); err != nil {
		t.Fatalf("decode first delivery: %v", err)
	}
	if err := json.Unmarshal([]byte(messages[1].Values["payload"].(string)), &resendPayload); err != nil {
		t.Fatalf("decode resend delivery: %v", err)
	}
	if firstPayload.DeliveryID == resendPayload.DeliveryID || firstPayload.ChallengeID != resendPayload.ChallengeID || firstPayload.Code != resendPayload.Code {
		t.Fatalf("resend deliveries = first %#v, resend %#v", firstPayload, resendPayload)
	}
	active, err := client.HGetAll(ctx, loginOwnershipActiveKey(email)).Result()
	if err != nil {
		t.Fatalf("load active resend: %v", err)
	}
	if active["delivery_id"] != resendPayload.DeliveryID || active["delivery_state"] != contracts.LoginOwnershipDeliveryStateQueued {
		t.Fatalf("active resend delivery = %#v, payload = %#v", active, resendPayload)
	}
	remainingTTL, err := client.PTTL(ctx, loginOwnershipActiveKey(email)).Result()
	if err != nil {
		t.Fatalf("load remaining TTL: %v", err)
	}
	if remainingTTL <= 0 || remainingTTL > 10*time.Minute {
		t.Fatalf("resend extended active TTL to %s", remainingTTL)
	}
	if firstTTL <= 0 {
		t.Fatalf("initial TTL = %s", firstTTL)
	}
}

func TestRedisEnsureLoginOwnershipChallengeNeverReturnsPhantomAfterActiveWasConsumed(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "no-phantom@example.com"
	clientIdentity := "198.51.100.13"
	if _, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity); err != nil {
		t.Fatalf("ensure initial challenge: %v", err)
	}
	if err := client.Del(ctx, loginOwnershipActiveKey(email)).Err(); err != nil {
		t.Fatalf("consume active challenge: %v", err)
	}

	if _, err := ensureLoginOwnershipChallenge(ctx, email, clientIdentity); !errors.Is(err, errLoginOwnershipSendLimited) {
		t.Fatalf("ensure without active during cooldown = %v, want send limited", err)
	}
	if count, err := client.XLen(ctx, contracts.TaskStreamKey).Result(); err != nil || count != 1 {
		t.Fatalf("task count after rejected ensure = %d, error %v; want 1", count, err)
	}
	if exists, err := client.Exists(ctx, loginOwnershipActiveKey(email)).Result(); err != nil || exists != 0 {
		t.Fatalf("phantom active challenge exists = %d, error %v", exists, err)
	}
}
