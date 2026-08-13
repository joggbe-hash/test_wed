package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"typewsp/shared/contracts"
)

type fakeMailSender struct {
	recipient               string
	code                    string
	loginOwnershipRecipient string
	loginOwnershipCode      string
	loginOwnershipExpiresAt time.Time
	loginOwnershipCalls     int
	err                     error
}

func (sender *fakeMailSender) SendVerificationCode(_ context.Context, recipient, code string) error {
	sender.recipient = recipient
	sender.code = code
	return sender.err
}

func (sender *fakeMailSender) SendLoginOwnershipCode(_ context.Context, recipient, code string, expiresAt time.Time) error {
	sender.loginOwnershipRecipient = recipient
	sender.loginOwnershipCode = code
	sender.loginOwnershipExpiresAt = expiresAt
	sender.loginOwnershipCalls++
	return sender.err
}

func TestHandleSendEmailUsesConfiguredSender(t *testing.T) {
	previous := mailSender
	t.Cleanup(func() { mailSender = previous })
	fake := &fakeMailSender{}
	mailSender = fake

	err := handleSendEmail(context.Background(), EmailPayload{Email: "user@example.test", Code: "123456"})
	if err != nil {
		t.Fatalf("handleSendEmail: %v", err)
	}
	if fake.recipient != "user@example.test" || fake.code != "123456" {
		t.Fatalf("sender received recipient=%q code=%q", fake.recipient, fake.code)
	}
}

func TestHandleSendEmailReturnsSenderFailureForQueueRetry(t *testing.T) {
	previous := mailSender
	t.Cleanup(func() { mailSender = previous })
	mailSender = &fakeMailSender{err: errors.New("SMTP unavailable")}

	err := handleSendEmail(context.Background(), EmailPayload{Email: "user@example.test", Code: "123456"})
	if err == nil {
		t.Fatal("expected SMTP error")
	}
}

func TestProcessTaskDecodesAndDispatchesLoginOwnershipEmail(t *testing.T) {
	previous := mailSender
	previousChecker := checkLoginOwnershipChallenge
	previousCompleter := completeLoginOwnershipDelivery
	t.Cleanup(func() {
		mailSender = previous
		checkLoginOwnershipChallenge = previousChecker
		completeLoginOwnershipDelivery = previousCompleter
	})
	fake := &fakeMailSender{}
	mailSender = fake
	checkLoginOwnershipChallenge = func(context.Context, contracts.LoginOwnershipEmailPayload) (bool, error) {
		return true, nil
	}
	completeLoginOwnershipDelivery = func(context.Context, contracts.LoginOwnershipEmailPayload) (bool, error) {
		return true, nil
	}
	expiresAt := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	payload, err := json.Marshal(contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "user@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: expiresAt.UnixMilli(),
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	err = processTask(context.Background(), Task{
		Type:    contracts.TaskSendVerificationEmail,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("processTask: %v", err)
	}
	if fake.loginOwnershipCalls != 1 || fake.loginOwnershipRecipient != "user@example.test" || fake.loginOwnershipCode != "ABCDEFGHJKLMNPQR" {
		t.Fatalf("unexpected ownership email dispatch: %#v", fake)
	}
	if !fake.loginOwnershipExpiresAt.Equal(expiresAt) {
		t.Fatalf("expiry = %s; want %s", fake.loginOwnershipExpiresAt, expiresAt)
	}
}

func TestProcessTaskRejectsMalformedLoginOwnershipEmailPayload(t *testing.T) {
	err := processTask(context.Background(), Task{
		Type: contracts.TaskSendVerificationEmail,
		Payload: json.RawMessage(
			`{"purpose":"login_ownership","email":"user@example.test","code":"ABCDEFGHJKLMNPQR"}`,
		),
	})
	if err == nil {
		t.Fatal("expected payload decode error")
	}
}

func TestHandleSendLoginOwnershipEmailSkipsExpiredPayload(t *testing.T) {
	previous := mailSender
	previousChecker := checkLoginOwnershipChallenge
	previousCompleter := completeLoginOwnershipDelivery
	t.Cleanup(func() {
		mailSender = previous
		checkLoginOwnershipChallenge = previousChecker
		completeLoginOwnershipDelivery = previousCompleter
	})
	fake := &fakeMailSender{}
	mailSender = fake
	checkLoginOwnershipChallenge = func(context.Context, contracts.LoginOwnershipEmailPayload) (bool, error) {
		t.Fatal("expired task checked Redis instead of skipping")
		return false, nil
	}
	now := time.UnixMilli(1_800_000_000_000)
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "user@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: now.UnixMilli(),
	}

	if err := handleSendLoginOwnershipEmailAt(context.Background(), payload, now); err != nil {
		t.Fatalf("expired task must be acknowledged without error: %v", err)
	}
	if fake.loginOwnershipCalls != 0 {
		t.Fatalf("expired task sent %d emails", fake.loginOwnershipCalls)
	}
}

func TestHandleSendLoginOwnershipEmailRejectsInvalidCodeFormats(t *testing.T) {
	previous := mailSender
	t.Cleanup(func() { mailSender = previous })
	fake := &fakeMailSender{}
	mailSender = fake
	now := time.UnixMilli(1_800_000_000_000)

	for _, code := range []string{
		"123456",
		"abcdefghjklmnpqr",
		"ABCDEFGHILMNPQRS",
		"ABCDEFGHJKLMNPQ",
	} {
		t.Run(code, func(t *testing.T) {
			payload := contracts.LoginOwnershipEmailPayload{
				Purpose:            contracts.EmailPurposeLoginOwnership,
				Email:              "user@example.test",
				Code:               code,
				ChallengeID:        "11111111-1111-4111-8111-111111111111",
				DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
				ExpiresAtUnixMilli: now.Add(time.Minute).UnixMilli(),
			}
			if err := handleSendLoginOwnershipEmailAt(context.Background(), payload, now); err == nil {
				t.Fatalf("accepted invalid login ownership code %q", code)
			}
		})
	}
	if fake.loginOwnershipCalls != 0 {
		t.Fatalf("invalid tasks sent %d emails", fake.loginOwnershipCalls)
	}
}

func TestProcessTaskReturnsLoginOwnershipSMTPFailureForQueueRetry(t *testing.T) {
	previous := mailSender
	previousChecker := checkLoginOwnershipChallenge
	previousCompleter := completeLoginOwnershipDelivery
	t.Cleanup(func() {
		mailSender = previous
		checkLoginOwnershipChallenge = previousChecker
		completeLoginOwnershipDelivery = previousCompleter
	})
	smtpErr := errors.New("SMTP unavailable")
	mailSender = &fakeMailSender{err: smtpErr}
	checkLoginOwnershipChallenge = func(context.Context, contracts.LoginOwnershipEmailPayload) (bool, error) {
		return true, nil
	}
	completeLoginOwnershipDelivery = func(context.Context, contracts.LoginOwnershipEmailPayload) (bool, error) {
		return true, nil
	}
	payload, err := json.Marshal(contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "user@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	err = processTask(context.Background(), Task{
		Type:    contracts.TaskSendVerificationEmail,
		Payload: payload,
	})
	if !errors.Is(err, smtpErr) {
		t.Fatalf("processTask error = %v; want wrapped SMTP error", err)
	}
}

func TestHandleSendLoginOwnershipEmailSkipsStaleChallengeBeforeSMTP(t *testing.T) {
	previous := mailSender
	previousChecker := checkLoginOwnershipChallenge
	t.Cleanup(func() {
		mailSender = previous
		checkLoginOwnershipChallenge = previousChecker
	})
	fake := &fakeMailSender{}
	mailSender = fake
	checkLoginOwnershipChallenge = func(_ context.Context, payload contracts.LoginOwnershipEmailPayload) (bool, error) {
		if payload.ChallengeID != "11111111-1111-4111-8111-111111111111" {
			t.Fatalf("checked challenge %q", payload.ChallengeID)
		}
		return false, nil
	}
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "user@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	}

	if err := handleSendLoginOwnershipEmail(context.Background(), payload); err != nil {
		t.Fatalf("stale task must be acknowledged without error: %v", err)
	}
	if fake.loginOwnershipCalls != 0 {
		t.Fatalf("stale task sent %d emails", fake.loginOwnershipCalls)
	}
}

func TestHandleSendLoginOwnershipEmailReturnsChallengeCheckFailureForRetry(t *testing.T) {
	previousChecker := checkLoginOwnershipChallenge
	t.Cleanup(func() { checkLoginOwnershipChallenge = previousChecker })
	redisErr := errors.New("Redis unavailable")
	checkLoginOwnershipChallenge = func(context.Context, contracts.LoginOwnershipEmailPayload) (bool, error) {
		return false, redisErr
	}
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "user@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	}

	if err := handleSendLoginOwnershipEmail(context.Background(), payload); !errors.Is(err, redisErr) {
		t.Fatalf("error = %v; want wrapped Redis error", err)
	}
}
