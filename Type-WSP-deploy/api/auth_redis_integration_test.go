package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

func TestRedisVerificationAttemptBudgetDoesNotInvalidateCorrectCode(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	challengeID := "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
	if _, err := activateVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
		t.Fatalf("activate verification challenge: %v", err)
	}

	for attempt := 1; attempt <= verificationCodeAttemptLimit; attempt++ {
		result, err := consumeVerificationCode(ctx, email, challengeID, "attacker", "000000")
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if attempt < verificationCodeAttemptLimit && result != verificationRejected {
			t.Fatalf("attempt %d result = %d, want rejected", attempt, result)
		}
		if attempt == verificationCodeAttemptLimit && result != verificationLocked {
			t.Fatalf("attempt %d result = %d, want locked", attempt, result)
		}
	}
	result, err := consumeVerificationCode(ctx, email, challengeID, "legitimate-client", "123456")
	if err != nil {
		t.Fatalf("post-lock attempt: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("post-lock result = %d, want accepted", result)
	}
}

func TestRedisVerificationChallengeBudgetDoesNotInvalidateHighEntropyCorrectCode(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	challengeID := "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
	if _, err := activateVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
		t.Fatalf("activate verification challenge: %v", err)
	}

	for attempt := 1; attempt <= verificationChallengeAttemptLimit; attempt++ {
		result, err := consumeVerificationCode(ctx, email, challengeID, fmt.Sprintf("attacker-%d", attempt), "000000")
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if attempt < verificationChallengeAttemptLimit && result != verificationRejected {
			t.Fatalf("attempt %d result = %d, want rejected", attempt, result)
		}
		if attempt == verificationChallengeAttemptLimit && result != verificationLocked {
			t.Fatalf("attempt %d result = %d, want locked", attempt, result)
		}
	}

	result, err := consumeVerificationCode(ctx, email, challengeID, "legitimate-client", "123456")
	if err != nil {
		t.Fatalf("post-lock attempt: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("post-lock result = %d, want accepted", result)
	}
	remaining, err := client.Exists(ctx, verificationCodeKey(email, challengeID)).Result()
	if err != nil {
		t.Fatalf("check invalidated challenge: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("challenge keys remaining after lock = %d, want 0", remaining)
	}
}

func TestRedisResendInvalidatesPreviousVerificationChallenge(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	oldChallengeID := "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
	newChallengeID := "79f2ca8c-8670-48dc-b095-017eace51bc4"
	if _, err := activateVerificationChallenge(ctx, email, oldChallengeID, "123456"); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	if _, err := activateVerificationChallenge(ctx, email, newChallengeID, "654321"); err != nil {
		t.Fatalf("activate new challenge: %v", err)
	}

	result, err := consumeVerificationCode(ctx, email, oldChallengeID, "client", "123456")
	if err != nil {
		t.Fatalf("consume old challenge: %v", err)
	}
	if result != verificationExpired {
		t.Fatalf("old challenge result = %d, want expired", result)
	}
	result, err = consumeVerificationCode(ctx, email, newChallengeID, "client", "654321")
	if err != nil {
		t.Fatalf("consume new challenge: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("new challenge result = %d, want accepted", result)
	}
}

func TestRedisRegistrationCommitPublishesMatchingChallengeAndTask(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "committed-registration@example.com"
	challengeID := "4ef42894-07f6-4b53-b198-77508f87b774"
	code := "ABCDEFGHJKLMNPQ2"

	if err := commitRegistrationVerificationChallenge(ctx, email, challengeID, code); err != nil {
		t.Fatalf("commit registration challenge: %v", err)
	}
	activeChallengeID, err := client.Get(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge: %v", err)
	}
	if activeChallengeID != challengeID {
		t.Fatalf("active challenge = %q, want %q", activeChallengeID, challengeID)
	}
	storedCode, err := client.Get(ctx, verificationCodeKey(email, challengeID)).Result()
	if err != nil {
		t.Fatalf("load committed code: %v", err)
	}
	if storedCode != code {
		t.Fatalf("stored code = %q, want %q", storedCode, code)
	}

	tasks, err := client.XRange(ctx, contracts.TaskStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("load committed task: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("committed tasks = %d, want 1", len(tasks))
	}
	if got := fmt.Sprint(tasks[0].Values["type"]); got != contracts.TaskSendVerificationEmail {
		t.Fatalf("task type = %q, want %q", got, contracts.TaskSendVerificationEmail)
	}
	if got := fmt.Sprint(tasks[0].Values["attempts"]); got != "0" {
		t.Fatalf("task attempts = %q, want 0", got)
	}
	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal([]byte(fmt.Sprint(tasks[0].Values["payload"])), &payload); err != nil {
		t.Fatalf("decode committed task payload: %v", err)
	}
	if payload.Email != email || payload.Code != code {
		t.Fatalf("task payload = %#v, want email %q and matching code", payload, email)
	}
}

func TestRedisFullQueueLeavesRegistrationChallengeUnchanged(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "full-queue@example.com"
	oldChallengeID := "d7063b64-c09b-4652-a6a1-e530a461256d"
	newChallengeID := "1c0a877d-76fa-4bd5-a5da-38719cc2ecba"
	oldCode := "ABCDEFGHJKLMNPQ2"
	if _, err := activateVerificationChallenge(ctx, email, oldChallengeID, oldCode); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	oldTTL := 30 * time.Second
	if err := client.PExpire(ctx, verificationCodeKey(email, oldChallengeID), oldTTL).Err(); err != nil {
		t.Fatalf("shorten old challenge TTL: %v", err)
	}
	if err := client.PExpire(ctx, verificationLatestChallengeKey(email), oldTTL).Err(); err != nil {
		t.Fatalf("shorten active challenge TTL: %v", err)
	}

	fillStream := redis.NewScript(`
for index = 1, tonumber(ARGV[1]) do
  redis.call('XADD', KEYS[1], '*', 'type', 'test', 'payload', '{}', 'attempts', '0')
end
return redis.call('XLEN', KEYS[1])
`)
	length, err := fillStream.Run(ctx, client, []string{contracts.TaskStreamKey}, maxTaskStreamLength).Int64()
	if err != nil {
		t.Fatalf("fill task stream: %v", err)
	}
	if length != maxTaskStreamLength {
		t.Fatalf("task stream length = %d, want %d", length, maxTaskStreamLength)
	}

	err = commitRegistrationVerificationChallenge(ctx, email, newChallengeID, "QRSTUVWXYZ234567")
	if !errors.Is(err, errTaskQueueFull) {
		t.Fatalf("full queue commit error = %v, want %v", err, errTaskQueueFull)
	}
	activeChallengeID, err := client.Get(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge: %v", err)
	}
	if activeChallengeID != oldChallengeID {
		t.Fatalf("active challenge = %q, want %q", activeChallengeID, oldChallengeID)
	}
	if exists, err := client.Exists(ctx, verificationCodeKey(email, newChallengeID)).Result(); err != nil {
		t.Fatalf("check rejected challenge: %v", err)
	} else if exists != 0 {
		t.Fatal("queue-full commit left a new challenge")
	}
	if length, err := client.XLen(ctx, contracts.TaskStreamKey).Result(); err != nil {
		t.Fatalf("load task stream length: %v", err)
	} else if length != maxTaskStreamLength {
		t.Fatalf("task stream length after rejection = %d, want %d", length, maxTaskStreamLength)
	}
	activeTTL, err := client.PTTL(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge TTL: %v", err)
	}
	if activeTTL <= 0 || activeTTL > oldTTL {
		t.Fatalf("active challenge TTL = %s, want > 0 and <= %s", activeTTL, oldTTL)
	}
}

func TestRedisFailedRegistrationCommitPreservesPreviousChallenge(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "failed-commit@example.com"
	oldChallengeID := "2d5519ab-5ee7-46e6-88ae-e18752f7fd1d"
	newChallengeID := "61b5863c-d197-42ea-964f-f64e8f5e1aae"
	if _, err := activateVerificationChallenge(ctx, email, oldChallengeID, "ABCDEFGHJKLMNPQ2"); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	oldTTL := 5 * time.Second
	if err := client.PExpire(ctx, verificationCodeKey(email, oldChallengeID), oldTTL).Err(); err != nil {
		t.Fatalf("shorten old challenge TTL: %v", err)
	}
	if err := client.PExpire(ctx, verificationLatestChallengeKey(email), oldTTL).Err(); err != nil {
		t.Fatalf("shorten active challenge TTL: %v", err)
	}
	if err := client.Set(ctx, contracts.TaskStreamKey, "force enqueue WRONGTYPE", 0).Err(); err != nil {
		t.Fatalf("force enqueue failure: %v", err)
	}

	if err := commitRegistrationVerificationChallenge(ctx, email, newChallengeID, "QRSTUVWXYZ234567"); err == nil {
		t.Fatal("registration commit succeeded with a WRONGTYPE task stream")
	}
	activeChallengeID, err := client.Get(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge: %v", err)
	}
	if activeChallengeID != oldChallengeID {
		t.Fatalf("active challenge = %q, want %q", activeChallengeID, oldChallengeID)
	}
	activeTTL, err := client.PTTL(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge TTL: %v", err)
	}
	if activeTTL <= 0 || activeTTL > oldTTL {
		t.Fatalf("active challenge TTL = %s, want > 0 and <= %s", activeTTL, oldTTL)
	}
	if exists, err := client.Exists(ctx, verificationCodeKey(email, newChallengeID)).Result(); err != nil {
		t.Fatalf("check failed challenge: %v", err)
	} else if exists != 0 {
		t.Fatal("failed registration challenge was left active")
	}

	result, err := consumeVerificationCode(ctx, email, oldChallengeID, "client", "ABCDEFGHJKLMNPQ2")
	if err != nil {
		t.Fatalf("consume preserved challenge: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("preserved challenge result = %d, want accepted", result)
	}
}

func TestRedisFailedRegistrationCommitDoesNotReplaceNewerChallenge(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "concurrent-failure@example.com"
	oldChallengeID := "048433c2-82f2-47e7-af72-ac3c7ad0bc9e"
	failedChallengeID := "a8573a3f-d768-4438-8052-f34a6a6e7ef7"
	newestChallengeID := "42576820-b348-47b9-b14a-591374462b89"
	if _, err := activateVerificationChallenge(ctx, email, oldChallengeID, "ABCDEFGHJKLMNPQ2"); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	if _, err := activateVerificationChallenge(ctx, email, newestChallengeID, "23456789ABCDEFGH"); err != nil {
		t.Fatalf("activate newest challenge: %v", err)
	}
	if err := client.Set(ctx, contracts.TaskStreamKey, "force enqueue WRONGTYPE", 0).Err(); err != nil {
		t.Fatalf("force enqueue failure: %v", err)
	}
	if err := commitRegistrationVerificationChallenge(ctx, email, failedChallengeID, "QRSTUVWXYZ234567"); err == nil {
		t.Fatal("registration commit succeeded with a WRONGTYPE task stream")
	}
	activeChallengeID, err := client.Get(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge: %v", err)
	}
	if activeChallengeID != newestChallengeID {
		t.Fatalf("active challenge = %q, want newest %q", activeChallengeID, newestChallengeID)
	}
	if exists, err := client.Exists(ctx, verificationCodeKey(email, failedChallengeID)).Result(); err != nil {
		t.Fatalf("check failed challenge: %v", err)
	} else if exists != 0 {
		t.Fatal("failed registration challenge was left active")
	}
	result, err := consumeVerificationCode(ctx, email, newestChallengeID, "client", "23456789ABCDEFGH")
	if err != nil {
		t.Fatalf("consume newest challenge: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("newest challenge result = %d, want accepted", result)
	}
}

func TestRedisSendCodeHandlerRestoresPreviousChallengeWhenEnqueueFails(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "handler-rollback@example.com"
	oldChallengeID := "6439e7b6-34ee-4f74-9dcc-17f32fe83615"
	oldCode := "ABCDEFGHJKLMNPQ2"
	if _, err := activateVerificationChallenge(ctx, email, oldChallengeID, oldCode); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	oldTTL := 5 * time.Second
	if err := client.PExpire(ctx, verificationCodeKey(email, oldChallengeID), oldTTL).Err(); err != nil {
		t.Fatalf("shorten old challenge TTL: %v", err)
	}
	if err := client.PExpire(ctx, verificationLatestChallengeKey(email), oldTTL).Err(); err != nil {
		t.Fatalf("shorten active challenge TTL: %v", err)
	}
	if err := client.Set(ctx, contracts.TaskStreamKey, "force enqueue WRONGTYPE", 0).Err(); err != nil {
		t.Fatalf("force enqueue failure: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/auth/send-code", bytes.NewBufferString(`{"email":"handler-rollback@example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	clientIdentity := requestClientIdentity(request)
	response := httptest.NewRecorder()
	handleSendCode(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("send-code status = %d, want %d; body = %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}

	activeChallengeID, err := client.Get(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load restored active challenge: %v", err)
	}
	if activeChallengeID != oldChallengeID {
		t.Fatalf("active challenge = %q, want %q", activeChallengeID, oldChallengeID)
	}
	activeTTL, err := client.PTTL(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load restored active challenge TTL: %v", err)
	}
	if activeTTL <= 0 || activeTTL > oldTTL {
		t.Fatalf("restored active challenge TTL = %s, want > 0 and <= %s", activeTTL, oldTTL)
	}
	result, err := consumeVerificationCode(ctx, email, oldChallengeID, "client", oldCode)
	if err != nil {
		t.Fatalf("consume restored challenge: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("restored challenge result = %d, want accepted", result)
	}
	reservationKeys := []string{
		verificationSendCooldownKey(email),
		verificationSendHourlyKey(email),
		verificationSendDailyKey(email),
		verificationClientSendHourlyKey(clientIdentity),
		verificationClientSendDailyKey(clientIdentity),
	}
	if remaining, err := client.Exists(ctx, reservationKeys...).Result(); err != nil {
		t.Fatalf("check verification send reservation rollback: %v", err)
	} else if remaining != 0 {
		t.Fatalf("verification send reservation keys remaining = %d, want 0", remaining)
	}
}

func TestRedisLimitedResendNeverObservesUnqueuedRegistrationChallenge(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "atomic-registration@example.com"
	oldChallengeID := "91c0a780-03bf-49a4-b389-acde68e96e8f"
	oldCode := "ABCDEFGHJKLMNPQ2"
	if _, err := activateVerificationChallenge(ctx, email, oldChallengeID, oldCode); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	if err := client.Set(ctx, contracts.TaskStreamKey, "force enqueue WRONGTYPE", 0).Err(); err != nil {
		t.Fatalf("force enqueue failure: %v", err)
	}

	barrier := &redisCommandBarrier{
		matchingArgument: contracts.TaskStreamKey,
		blocked:          make(chan struct{}),
		release:          make(chan struct{}),
	}
	client.AddHook(barrier)

	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/send-code", bytes.NewBufferString(`{"email":"atomic-registration@example.com"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handleSendCode(response, request)
		firstDone <- response
	}()

	select {
	case <-barrier.blocked:
	case <-time.After(3 * time.Second):
		close(barrier.release)
		t.Fatal("first send-code request did not reach the task stream command")
	}

	secondRequest := httptest.NewRequest(http.MethodPost, "/api/auth/send-code", bytes.NewBufferString(`{"email":"atomic-registration@example.com"}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	secondResponse := httptest.NewRecorder()
	handleSendCode(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusOK {
		close(barrier.release)
		t.Fatalf("limited resend status = %d, want %d; body = %s", secondResponse.Code, http.StatusOK, secondResponse.Body.String())
	}
	var limitedResponse struct {
		ChallengeID string `json:"challenge_id"`
	}
	if err := json.Unmarshal(secondResponse.Body.Bytes(), &limitedResponse); err != nil {
		close(barrier.release)
		t.Fatalf("decode limited resend response: %v", err)
	}
	if limitedResponse.ChallengeID != oldChallengeID {
		close(barrier.release)
		t.Fatalf("limited resend exposed challenge %q, want previously committed %q", limitedResponse.ChallengeID, oldChallengeID)
	}

	close(barrier.release)
	select {
	case firstResponse := <-firstDone:
		if firstResponse.Code != http.StatusInternalServerError {
			t.Fatalf("failed enqueue status = %d, want %d; body = %s", firstResponse.Code, http.StatusInternalServerError, firstResponse.Body.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("first send-code request did not finish")
	}

	activeChallengeID, err := client.Get(ctx, verificationLatestChallengeKey(email)).Result()
	if err != nil {
		t.Fatalf("load active challenge: %v", err)
	}
	if activeChallengeID != oldChallengeID {
		t.Fatalf("active challenge = %q, want %q", activeChallengeID, oldChallengeID)
	}
}

func TestRedisLimitedResendRejectsLatestChallengeWithoutCode(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "orphaned-registration@example.com"
	orphanedChallengeID := "f450c7d7-d3b9-422d-b3e0-40de1247d824"
	if err := client.Set(ctx, verificationLatestChallengeKey(email), orphanedChallengeID, time.Minute).Err(); err != nil {
		t.Fatalf("store orphaned latest challenge: %v", err)
	}
	if err := client.Set(ctx, verificationSendCooldownKey(email), "1", time.Minute).Err(); err != nil {
		t.Fatalf("store resend cooldown: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/auth/send-code", bytes.NewBufferString(`{"email":"orphaned-registration@example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handleSendCode(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("limited resend status = %d, want %d; body = %s", response.Code, http.StatusTooManyRequests, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode limited resend response: %v", err)
	}
	if _, exposed := body["challenge_id"]; exposed {
		t.Fatalf("limited resend exposed orphaned challenge: %s", response.Body.String())
	}
}

func TestRedisLimitedResendDoesNotReturnChallengeReplacedDuringLookup(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "replaced-registration@example.com"
	oldChallengeID := "7190baf2-f157-42af-a83c-70036f767e4a"
	newChallengeID := "2f8af84f-f4dd-4264-903c-e8bc5d23cded"
	if _, err := activateVerificationChallenge(ctx, email, oldChallengeID, "ABCDEFGHJKLMNPQ2"); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	if err := client.Set(ctx, verificationSendCooldownKey(email), "1", time.Minute).Err(); err != nil {
		t.Fatalf("store resend cooldown: %v", err)
	}

	barrier := &redisCommandBarrier{
		matchingArgument: verificationCodeKey(email, oldChallengeID),
		blocked:          make(chan struct{}),
		release:          make(chan struct{}),
	}
	client.AddHook(barrier)
	limitedDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/send-code", bytes.NewBufferString(`{"email":"replaced-registration@example.com"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handleSendCode(response, request)
		limitedDone <- response
	}()

	select {
	case <-barrier.blocked:
	case <-time.After(3 * time.Second):
		close(barrier.release)
		t.Fatal("limited resend did not reach active challenge validation")
	}
	if err := commitRegistrationVerificationChallenge(ctx, email, newChallengeID, "QRSTUVWXYZ234567"); err != nil {
		close(barrier.release)
		t.Fatalf("commit replacement challenge: %v", err)
	}
	close(barrier.release)

	select {
	case response := <-limitedDone:
		if response.Code != http.StatusOK {
			t.Fatalf("limited resend status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
		}
		var body struct {
			ChallengeID string `json:"challenge_id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode limited resend response: %v", err)
		}
		if body.ChallengeID != newChallengeID {
			t.Fatalf("limited resend returned stale challenge %q, want latest %q", body.ChallengeID, newChallengeID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("limited resend did not finish")
	}
}

func TestRedisLoginVerificationChallengeIsSingleUseAndFlowIsolated(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	challengeID := "2cd53940-fc0d-4972-921b-086061dde6e5"

	if err := activateLoginVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
		t.Fatalf("activate login challenge: %v", err)
	}

	registrationResult, err := consumeVerificationCode(ctx, email, challengeID, "client", "123456")
	if err != nil {
		t.Fatalf("consume login code through registration flow: %v", err)
	}
	if registrationResult != verificationExpired {
		t.Fatalf("registration accepted a login code: result = %d", registrationResult)
	}

	result, err := consumeLoginVerificationCode(ctx, email, challengeID, "client", "123456")
	if err != nil {
		t.Fatalf("consume login challenge: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("login challenge result = %d, want accepted", result)
	}

	result, err = consumeLoginVerificationCode(ctx, email, challengeID, "client", "123456")
	if err != nil {
		t.Fatalf("reuse login challenge: %v", err)
	}
	if result != verificationExpired {
		t.Fatalf("reused login challenge result = %d, want expired", result)
	}
}

func TestRedisLoginVerificationRejectedBeforeUserLookup(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "known-user@example.com"
	challengeID := "ad3be93f-6a39-44b9-8834-fcd3db8f0c90"
	if err := activateLoginVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
		t.Fatalf("activate login challenge: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login/verify", bytes.NewBufferString(fmt.Sprintf(
		`{"email":%q,"code":"000000","challenge_id":%q}`,
		email,
		challengeID,
	)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	lookupCalls := 0
	handleLoginVerificationWithUserLookup(response, request, func(context.Context, string) (User, error) {
		lookupCalls++
		return User{}, errors.New("user lookup must not run for a rejected code")
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if body := response.Body.String(); body != "{\"error\":\"invalid login verification code\"}\n" {
		t.Fatalf("body = %q", body)
	}
	if lookupCalls != 0 {
		t.Fatalf("user lookup calls = %d, want 0", lookupCalls)
	}
}

func TestRedisLoginVerificationExpiredChallengeDoesNotRevealKnownEmail(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	knownEmail := "known-user@example.com"
	activeChallengeID := "4a09b2bb-aafd-4d4e-8209-a3cc9a40295f"
	arbitraryChallengeID := "79ac0dfb-b964-41ed-84e4-22665d88e0d8"
	if err := activateLoginVerificationChallenge(ctx, knownEmail, activeChallengeID, "123456"); err != nil {
		t.Fatalf("activate login challenge: %v", err)
	}

	var responses []string
	lookupCalls := 0
	for _, email := range []string{knownEmail, "unknown-user@example.com"} {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login/verify", bytes.NewBufferString(fmt.Sprintf(
			`{"email":%q,"code":"123456","challenge_id":%q}`,
			email,
			arbitraryChallengeID,
		)))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handleLoginVerificationWithUserLookup(response, request, func(context.Context, string) (User, error) {
			lookupCalls++
			return User{}, errors.New("user lookup must not run for an expired challenge")
		})

		if response.Code != http.StatusBadRequest {
			t.Fatalf("email %q status = %d, want %d; body = %s", email, response.Code, http.StatusBadRequest, response.Body.String())
		}
		responses = append(responses, response.Body.String())
	}

	if lookupCalls != 0 {
		t.Fatalf("user lookup calls = %d, want 0", lookupCalls)
	}
	if responses[0] != responses[1] || responses[0] != "{\"error\":\"login verification code expired or not found\"}\n" {
		t.Fatalf("known and unknown responses differ: %#v", responses)
	}
}

func TestRedisLoginVerificationLockedBeforeUserLookup(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "known-user@example.com"
	challengeID := "a84191ec-0957-4077-a002-d988dd43b3b7"
	if err := activateLoginVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
		t.Fatalf("activate login challenge: %v", err)
	}

	lookupCalls := 0
	var finalResponse *httptest.ResponseRecorder
	for attempt := 1; attempt <= verificationCodeAttemptLimit; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login/verify", bytes.NewBufferString(fmt.Sprintf(
			`{"email":%q,"code":"000000","challenge_id":%q}`,
			email,
			challengeID,
		)))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handleLoginVerificationWithUserLookup(response, request, func(context.Context, string) (User, error) {
			lookupCalls++
			return User{}, errors.New("user lookup must not run before code acceptance")
		})
		finalResponse = response
	}

	if finalResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body = %s", finalResponse.Code, http.StatusTooManyRequests, finalResponse.Body.String())
	}
	if lookupCalls != 0 {
		t.Fatalf("user lookup calls = %d, want 0", lookupCalls)
	}
}

func TestRedisLoginVerificationAcceptedThenLooksUpUser(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "deleted-user@example.com"
	challengeID := "0dac50d0-8cca-484f-8457-501e558474a4"
	if err := activateLoginVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
		t.Fatalf("activate login challenge: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login/verify", bytes.NewBufferString(fmt.Sprintf(
		`{"email":%q,"code":"123456","challenge_id":%q}`,
		email,
		challengeID,
	)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	lookupCalls := 0
	handleLoginVerificationWithUserLookup(response, request, func(_ context.Context, gotEmail string) (User, error) {
		lookupCalls++
		if gotEmail != email {
			t.Fatalf("lookup email = %q, want %q", gotEmail, email)
		}
		return User{}, pgx.ErrNoRows
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if body := response.Body.String(); body != "{\"error\":\"login verification code expired or not found\"}\n" {
		t.Fatalf("body = %q", body)
	}
	if lookupCalls != 1 {
		t.Fatalf("user lookup calls = %d, want 1", lookupCalls)
	}
}

func TestRedisVerificationSendBudgetHasCumulativeHourlyBound(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	for send := 1; send <= verificationCodeHourlySendLimit; send++ {
		result, err := reserveVerificationSend(ctx, email, "203.0.113.5")
		if err != nil {
			t.Fatalf("send %d: %v", send, err)
		}
		if result != 1 {
			t.Fatalf("send %d result = %d, want allowed", send, result)
		}
		if err := client.Del(ctx, verificationSendCooldownKey(email)).Err(); err != nil {
			t.Fatalf("advance cooldown after send %d: %v", send, err)
		}
	}
	result, err := reserveVerificationSend(ctx, email, "203.0.113.5")
	if err != nil {
		t.Fatalf("bounded send: %v", err)
	}
	if result != -2 {
		t.Fatalf("bounded send result = %d, want hourly rejection", result)
	}
}

func TestRedisAccountBudgetRequiresOwnershipBeforePasswordVerification(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	var accountAttempts int64
	for attempt := 1; attempt <= accountLoginAttemptLimit; attempt++ {
		admission, _, current, err := reserveLoginAttempt(ctx, email, fmt.Sprintf("203.0.113.%d", attempt), "")
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if admission != loginAdmissionAllowed {
			t.Fatalf("attempt %d admission = %d, want allowed", attempt, admission)
		}
		accountAttempts = current
	}
	if !accountLoginAttemptShouldBlock(accountAttempts) {
		t.Fatalf("account attempts = %d, want blocked", accountAttempts)
	}
	accountTTLBefore, err := client.PTTL(ctx, accountLoginAttemptKey(email)).Result()
	if err != nil {
		t.Fatalf("load account TTL before ownership challenge: %v", err)
	}
	admission, _, current, err := reserveLoginAttempt(ctx, email, "198.51.100.1", "")
	if err != nil {
		t.Fatalf("clean-client attempt: %v", err)
	}
	if admission != loginAdmissionOwnershipRequired || current != accountLoginAttemptLimit {
		t.Fatalf("clean-client admission after account threshold = %d, count %d", admission, current)
	}
	accountTTLAfter, err := client.PTTL(ctx, accountLoginAttemptKey(email)).Result()
	if err != nil {
		t.Fatalf("load account TTL after ownership challenge: %v", err)
	}
	if accountTTLAfter <= 0 || accountTTLAfter > accountTTLBefore {
		t.Fatalf("account TTL changed from %s to %s while requesting ownership", accountTTLBefore, accountTTLAfter)
	}
	if exists, err := client.Exists(ctx, loginAttemptKey(email, "198.51.100.1")).Result(); err != nil {
		t.Fatalf("check clean-client counter: %v", err)
	} else if exists != 0 {
		t.Fatal("requesting ownership consumed the clean client's attempt budget")
	}
}

func TestRedisLoginAccountBudgetIsAtomicAcrossConcurrentClients(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	const attempts = 64

	type result struct {
		admission loginAdmission
		err       error
	}
	results := make(chan result, attempts)
	var group sync.WaitGroup
	for attempt := 0; attempt < attempts; attempt++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			admission, _, _, err := reserveLoginAttempt(
				ctx,
				"target@example.com",
				fmt.Sprintf("198.51.100.%d", index),
				"",
			)
			results <- result{admission: admission, err: err}
		}(attempt)
	}
	group.Wait()
	close(results)

	allowedCount := 0
	ownershipRequiredCount := 0
	for current := range results {
		if current.err != nil {
			t.Fatalf("concurrent reservation failed: %v", current.err)
		}
		switch current.admission {
		case loginAdmissionAllowed:
			allowedCount++
		case loginAdmissionOwnershipRequired:
			ownershipRequiredCount++
		default:
			t.Fatalf("unexpected concurrent admission %d", current.admission)
		}
	}
	if allowedCount != accountLoginAttemptLimit || ownershipRequiredCount != attempts-accountLoginAttemptLimit {
		t.Fatalf("concurrent admissions = %d allowed / %d ownership-required", allowedCount, ownershipRequiredCount)
	}
	accountAttempts, err := rdb.Get(ctx, accountLoginAttemptKey("target@example.com")).Int64()
	if err != nil {
		t.Fatalf("load concurrent account attempts: %v", err)
	}
	if accountAttempts != accountLoginAttemptLimit {
		t.Fatalf("concurrent account attempts = %d, want %d", accountAttempts, accountLoginAttemptLimit)
	}
}

func TestRedisDestroySessionDeletesStateAndPublishesRevocation(t *testing.T) {
	client := useIntegrationRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	previousSecret := sessionSecret
	sessionSecret = []byte("integration-test-secret")
	t.Cleanup(func() { sessionSecret = previousSecret })

	sid := "session-id"
	channel := sessionRevocationChannel(sid)
	subscription := client.Subscribe(ctx, channel)
	t.Cleanup(func() { _ = subscription.Close() })
	if _, err := subscription.Receive(ctx); err != nil {
		t.Fatalf("subscribe to revocation: %v", err)
	}
	if err := client.Set(ctx, sessionPrefix+sid, "{}", time.Minute).Err(); err != nil {
		t.Fatalf("store session: %v", err)
	}

	if err := DestroySession(ctx, signSID(sid)); err != nil {
		t.Fatalf("destroy session: %v", err)
	}
	if err := client.Get(ctx, sessionPrefix+sid).Err(); !errors.Is(err, redis.Nil) {
		t.Fatalf("session still exists after logout: %v", err)
	}
	message, err := subscription.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive revocation: %v", err)
	}
	if message.Channel != channel || message.Payload != "revoked" {
		t.Fatalf("revocation message = %#v", message)
	}
}

type redisCommandBarrier struct {
	matchingArgument string
	blocked          chan struct{}
	release          chan struct{}
	once             sync.Once
}

func (barrier *redisCommandBarrier) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (barrier *redisCommandBarrier) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		for _, arg := range cmd.Args() {
			if fmt.Sprint(arg) != barrier.matchingArgument {
				continue
			}
			barrier.once.Do(func() {
				close(barrier.blocked)
				select {
				case <-barrier.release:
				case <-ctx.Done():
				}
			})
			break
		}
		return next(ctx, cmd)
	}
}

func (barrier *redisCommandBarrier) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func useIntegrationRedis(t *testing.T) *redis.Client {
	t.Helper()
	rawURL := os.Getenv("TEST_REDIS_URL")
	if rawURL == "" {
		t.Skip("TEST_REDIS_URL is not set")
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		t.Fatalf("parse TEST_REDIS_URL: %v", err)
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.FlushDB(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("flush test Redis: %v", err)
	}

	previous := rdb
	rdb = client
	t.Cleanup(func() {
		rdb = previous
		_ = client.Close()
	})
	return client
}
