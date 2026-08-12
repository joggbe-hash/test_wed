package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisVerificationAttemptBudgetDoesNotInvalidateCorrectCode(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	challengeID := "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
	if err := activateVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
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
	if err := activateVerificationChallenge(ctx, email, challengeID, "123456"); err != nil {
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

func TestRedisResendKeepsPreviousVerificationChallengeUsable(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	oldChallengeID := "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
	newChallengeID := "79f2ca8c-8670-48dc-b095-017eace51bc4"
	if err := activateVerificationChallenge(ctx, email, oldChallengeID, "123456"); err != nil {
		t.Fatalf("activate old challenge: %v", err)
	}
	if err := activateVerificationChallenge(ctx, email, newChallengeID, "654321"); err != nil {
		t.Fatalf("activate new challenge: %v", err)
	}

	result, err := consumeVerificationCode(ctx, email, oldChallengeID, "client", "123456")
	if err != nil {
		t.Fatalf("consume old challenge: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("old challenge result = %d, want accepted", result)
	}
	result, err = consumeVerificationCode(ctx, email, newChallengeID, "client", "654321")
	if err != nil {
		t.Fatalf("consume new challenge: %v", err)
	}
	if result != verificationAccepted {
		t.Fatalf("new challenge result = %d, want accepted", result)
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

func TestRedisAccountBudgetBlocksCleanClientBeforePasswordVerification(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	email := "user@example.com"
	var accountAttempts int64
	for attempt := 1; attempt <= accountLoginAttemptLimit; attempt++ {
		allowed, _, current, err := reserveLoginAttempt(ctx, email, fmt.Sprintf("203.0.113.%d", attempt))
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if !allowed {
			t.Fatalf("attempt %d was rejected before the account budget was exhausted", attempt)
		}
		accountAttempts = current
	}
	if !accountLoginAttemptShouldBlock(accountAttempts) {
		t.Fatalf("account attempts = %d, want blocked", accountAttempts)
	}
	allowed, _, current, err := reserveLoginAttempt(ctx, email, "198.51.100.1")
	if err != nil {
		t.Fatalf("bounded attempt: %v", err)
	}
	if allowed || current != accountLoginAttemptLimit {
		t.Fatalf("clean-client attempt after account exhaustion = allowed %v, count %d", allowed, current)
	}
}

func TestRedisLoginAccountBudgetIsAtomicAcrossConcurrentClients(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	const attempts = 64

	type result struct {
		allowed bool
		err     error
	}
	results := make(chan result, attempts)
	var group sync.WaitGroup
	for attempt := 0; attempt < attempts; attempt++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			allowed, _, _, err := reserveLoginAttempt(ctx, "target@example.com", fmt.Sprintf("198.51.100.%d", index))
			results <- result{allowed: allowed, err: err}
		}(attempt)
	}
	group.Wait()
	close(results)

	allowedCount := 0
	for current := range results {
		if current.err != nil {
			t.Fatalf("concurrent reservation failed: %v", current.err)
		}
		if current.allowed {
			allowedCount++
		}
	}
	if allowedCount != accountLoginAttemptLimit {
		t.Fatalf("concurrent clean clients allowed %d attempts, want %d", allowedCount, accountLoginAttemptLimit)
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
