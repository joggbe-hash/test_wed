package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestVerifyLoginPasswordUsesDummyHashForUnknownAccount(t *testing.T) {
	comparisonCount := 0
	matched := verifyLoginPasswordWithCompare("", "candidate-password", func(hash, password []byte) error {
		comparisonCount++
		if len(hash) == 0 {
			t.Fatal("unknown account used an empty password hash")
		}
		if !bytes.Equal(password, []byte("candidate-password")) {
			t.Fatalf("password = %q", password)
		}
		return nil
	})

	if matched {
		t.Fatal("unknown account authenticated when the dummy comparison matched")
	}
	if comparisonCount != 1 {
		t.Fatalf("comparison count = %d, want 1", comparisonCount)
	}
}

func TestVerifyLoginPasswordPreservesExistingAccountBehavior(t *testing.T) {
	matched := verifyLoginPasswordWithCompare("stored-hash", "correct-password", func(hash, password []byte) error {
		if !bytes.Equal(hash, []byte("stored-hash")) || !bytes.Equal(password, []byte("correct-password")) {
			t.Fatalf("comparison inputs = %q / %q", hash, password)
		}
		return nil
	})
	if !matched {
		t.Fatal("existing account with a matching password was rejected")
	}

	matched = verifyLoginPasswordWithCompare("stored-hash", "wrong-password", func([]byte, []byte) error {
		return errors.New("password mismatch")
	})
	if matched {
		t.Fatal("existing account with a mismatched password authenticated")
	}
}

func TestDummyLoginPasswordHashMatchesRegistrationCost(t *testing.T) {
	cost, err := bcrypt.Cost(dummyLoginPasswordHash)
	if err != nil {
		t.Fatalf("dummy login hash is invalid: %v", err)
	}
	if cost != 12 {
		t.Fatalf("dummy login hash cost = %d, want 12", cost)
	}
}

func TestLoginAttemptKeyIsScopedToAccountAndClient(t *testing.T) {
	key := loginAttemptKey(" User@Example.com ", "203.0.113.5")
	if key != loginAttemptKey("user@example.com", "203.0.113.5") {
		t.Fatal("equivalent email forms produced different login-attempt keys")
	}
	if key == loginAttemptKey("user@example.com", "203.0.113.6") {
		t.Fatal("different clients shared one login-attempt key")
	}
	if strings.Contains(strings.ToLower(key), "user@example.com") {
		t.Fatalf("login-attempt key exposes the account identifier: %q", key)
	}
}

func TestLoginAttemptPolicyBlocksAttemptsPastTheAccountLimit(t *testing.T) {
	if !loginAttemptAllowed(loginAttemptLimit) {
		t.Fatalf("attempt %d should remain allowed", loginAttemptLimit)
	}
	if loginAttemptAllowed(loginAttemptLimit + 1) {
		t.Fatalf("attempt %d should be blocked", loginAttemptLimit+1)
	}
}

func TestLoginPreAuthenticationBlockUsesOnlyClientBudget(t *testing.T) {
	if !loginPreAuthenticationShouldBlock(loginAttemptLimit, 0) {
		t.Fatal("login was allowed after the client failure budget was exhausted")
	}
	if loginPreAuthenticationShouldBlock(0, accountLoginAttemptLimit) {
		t.Fatal("an attacker exhausted the account-wide signal and locked out a clean client")
	}
	if loginPreAuthenticationShouldBlock(loginAttemptLimit-1, accountLoginAttemptLimit+1) {
		t.Fatal("account-wide risk signal became a pre-authentication account lock")
	}
}

func TestRegistrationVerificationCodeHasAtLeastEightyBitsOfEntropy(t *testing.T) {
	code, err := generateRegistrationVerificationCode()
	if err != nil {
		t.Fatalf("generate registration verification code: %v", err)
	}
	if len(code) != registrationVerificationCodeLength {
		t.Fatalf("code length = %d, want %d", len(code), registrationVerificationCodeLength)
	}
	if !validRegistrationVerificationCode(code) {
		t.Fatalf("generated code is outside the registration alphabet: %q", code)
	}
}

func TestLoginPasswordVerificationLimiterBoundsExpensiveWork(t *testing.T) {
	limiter := newConcurrencyLimiter(2)
	if !limiter.tryAcquire() || !limiter.tryAcquire() {
		t.Fatal("limiter rejected work below its capacity")
	}
	if limiter.tryAcquire() {
		t.Fatal("limiter admitted bcrypt work above its capacity")
	}
	limiter.release()
	if !limiter.tryAcquire() {
		t.Fatal("limiter did not restore capacity after release")
	}
}

func TestLoginAtomicallyReservesAttemptBeforePasswordVerification(t *testing.T) {
	source, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatalf("read auth.go: %v", err)
	}
	handlerStart := bytes.Index(source, []byte("func handleLogin("))
	if handlerStart < 0 {
		t.Fatal("handleLogin source was not found")
	}
	handlerEnd := bytes.Index(source[handlerStart:], []byte("\ntype loginVerificationRequest struct"))
	if handlerEnd < 0 {
		t.Fatal("handleLogin end was not found")
	}
	handler := source[handlerStart : handlerStart+handlerEnd]
	reserveAt := bytes.Index(handler, []byte("reserveLoginAttempt("))
	passwordAt := bytes.Index(handler, []byte("verifyLoginPassword("))
	if reserveAt < 0 || passwordAt < 0 || reserveAt > passwordAt {
		t.Fatal("login attempt budget is not atomically reserved before password verification")
	}
}

func TestLoginVerificationKeysAreIsolatedFromRegistration(t *testing.T) {
	email := "user@example.com"
	challengeID := "69ca80a8-e3be-4dbf-8a4b-6600146f5574"

	if loginVerificationCodeKey(email) == verificationCodeKey(email, challengeID) {
		t.Fatal("login and registration shared a verification-code key")
	}
	if loginVerificationActiveChallengeKey(email) == verificationLatestChallengeKey(email) {
		t.Fatal("login and registration shared an active-challenge key")
	}
	if loginVerificationChallengeAttemptKey(challengeID) == verificationChallengeAttemptKey(challengeID) {
		t.Fatal("login and registration shared a challenge-attempt key")
	}
	if loginVerificationSendCooldownKey(email) == verificationSendCooldownKey(email) {
		t.Fatal("login and registration shared a send-cooldown key")
	}
}

func TestLoginVerificationResponseRequiresCodeBeforeSession(t *testing.T) {
	previous := debugVerificationCode
	debugVerificationCode = false
	t.Cleanup(func() { debugVerificationCode = previous })

	response := loginVerificationCodeResponse("challenge-id", "123456")
	if response["requires_verification"] != true {
		t.Fatalf("requires_verification = %#v, want true", response["requires_verification"])
	}
	if response["challenge_id"] != "challenge-id" {
		t.Fatalf("challenge_id = %#v", response["challenge_id"])
	}
	if _, exists := response["user"]; exists {
		t.Fatal("password phase exposed an authenticated user")
	}
	if _, exists := response["debug_code"]; exists {
		t.Fatal("production login challenge exposed the verification code")
	}
}

func TestAccountLoginAttemptKeyCannotBeResetByChangingClient(t *testing.T) {
	first := accountLoginAttemptKey(" User@Example.com ")
	if first != accountLoginAttemptKey("user@example.com") {
		t.Fatal("equivalent email forms produced different account-attempt keys")
	}
	if strings.Contains(first, "user@example.com") {
		t.Fatalf("account-attempt key exposes the account identifier: %q", first)
	}
}

func TestVerificationAttemptKeysAreScopedToChallengeAndClient(t *testing.T) {
	challengeID := "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
	first := verificationClientAttemptKey(challengeID, "203.0.113.5")
	if first == verificationClientAttemptKey(challengeID, "203.0.113.6") {
		t.Fatal("different clients shared one verification-attempt key")
	}
	if first == verificationClientAttemptKey("79f2ca8c-8670-48dc-b095-017eace51bc4", "203.0.113.5") {
		t.Fatal("different challenges shared one verification-attempt key")
	}
	if verificationChallengeAttemptKey(challengeID) == verificationChallengeAttemptKey("79f2ca8c-8670-48dc-b095-017eace51bc4") {
		t.Fatal("different challenges shared one global verification-attempt key")
	}
	if strings.Contains(first, challengeID) {
		t.Fatalf("verification key exposes the challenge id: %q", first)
	}
}

func TestVerificationAttemptPolicyLocksWrongGuessesAtLimit(t *testing.T) {
	if !verificationAttemptAllowed(verificationCodeAttemptLimit - 1) {
		t.Fatal("attempt below the limit was rejected")
	}
	if verificationAttemptAllowed(verificationCodeAttemptLimit) {
		t.Fatal("attempt at the limit remained allowed")
	}
}

func TestVerificationSendPolicyHasHourlyAndDailyRecipientBounds(t *testing.T) {
	if !verificationSendAllowed(verificationCodeHourlySendLimit-1, verificationCodeDailySendLimit-1) {
		t.Fatal("legitimate send below both recipient limits was rejected")
	}
	if verificationSendAllowed(verificationCodeHourlySendLimit, 0) {
		t.Fatal("send at the hourly recipient limit was accepted")
	}
	if verificationSendAllowed(0, verificationCodeDailySendLimit) {
		t.Fatal("send at the daily recipient limit was accepted")
	}
}

func TestVerificationClientSendKeyIsScopedToSource(t *testing.T) {
	first := verificationClientSendHourlyKey("203.0.113.5")
	if first == verificationClientSendHourlyKey("203.0.113.6") {
		t.Fatal("different clients shared one verification-send key")
	}
	if strings.Contains(first, "203.0.113.5") {
		t.Fatalf("verification-send key exposes the client identity: %q", first)
	}
}
