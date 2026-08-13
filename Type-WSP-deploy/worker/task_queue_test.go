package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

func TestTaskFromMessage(t *testing.T) {
	notBefore := time.Now().Add(time.Minute).UnixMilli()
	message := redis.XMessage{ID: "1-0", Values: map[string]any{
		"type": "image_post", "payload": `{"post_id":7}`, "attempts": "2",
		"not_before_unix_milli": notBefore,
	}}
	task, err := taskFromMessage(message)
	if err != nil {
		t.Fatalf("taskFromMessage: %v", err)
	}
	if task.MessageID != "1-0" || task.Type != "image_post" || task.Attempts != 2 || task.NotBeforeUnixMilli != notBefore {
		t.Fatalf("unexpected task: %#v", task)
	}
}

func TestOwnershipRetryUsesExponentialNotBeforeDelay(t *testing.T) {
	now := time.UnixMilli(1_800_000_000_000)
	for _, test := range []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 15 * time.Second},
		{attempt: 2, want: 30 * time.Second},
	} {
		values := retryTaskValuesAt(Task{
			Type:     contracts.TaskSendVerificationEmail,
			Payload:  json.RawMessage(`{"purpose":"login_ownership"}`),
			Attempts: test.attempt - 1,
		}, test.attempt, now)
		got, ok := values["not_before_unix_milli"].(int64)
		if !ok || got != now.Add(test.want).UnixMilli() {
			t.Fatalf("attempt %d not-before = %#v; want %d", test.attempt, values["not_before_unix_milli"], now.Add(test.want).UnixMilli())
		}
	}

	values := retryTaskValuesAt(Task{
		Type:     contracts.TaskProcessImagePost,
		Payload:  json.RawMessage(`{"post_id":7}`),
		Attempts: 0,
	}, 1, now)
	if _, exists := values["not_before_unix_milli"]; exists {
		t.Fatalf("non-email retry unexpectedly delayed: %#v", values)
	}
}

func TestVerificationEmailRetryErrorIsSanitized(t *testing.T) {
	secretCause := errors.New("SMTP rejected owner@example.test: provider account 123")
	verificationTask := Task{
		Type:    contracts.TaskSendVerificationEmail,
		Payload: json.RawMessage(`{"email":"owner@example.test","code":"123456"}`),
	}
	if got := safeTaskError(verificationTask, secretCause); got != "email delivery failed" {
		t.Fatalf("verification task error = %q", got)
	}
	imageTask := Task{Type: contracts.TaskProcessImagePost, Payload: json.RawMessage(`{"post_id":7}`)}
	if got := safeTaskError(imageTask, secretCause); got != secretCause.Error() {
		t.Fatalf("non-email task error = %q; want original cause", got)
	}
}

func TestDelayedTaskSchedulerOrdersTasksWithoutBlockingTheWorker(t *testing.T) {
	now := time.UnixMilli(1_800_000_000_000)
	var scheduled delayedTaskHeap
	if !scheduled.schedule(Task{MessageID: "later", NotBeforeUnixMilli: now.Add(30 * time.Second).UnixMilli()}, now) {
		t.Fatal("future task was not scheduled")
	}
	if !scheduled.schedule(Task{MessageID: "sooner", NotBeforeUnixMilli: now.Add(15 * time.Second).UnixMilli()}, now) {
		t.Fatal("earlier future task was not scheduled")
	}
	if scheduled.schedule(Task{MessageID: "ready"}, now) {
		t.Fatal("ready task was delayed")
	}
	if _, ok := scheduled.popReady(now.Add(14 * time.Second)); ok {
		t.Fatal("task became ready before its not-before time")
	}
	first, ok := scheduled.popReady(now.Add(15 * time.Second))
	if !ok || first.MessageID != "sooner" {
		t.Fatalf("first ready task = %#v ok=%v; want sooner", first, ok)
	}
	second, ok := scheduled.popReady(now.Add(30 * time.Second))
	if !ok || second.MessageID != "later" {
		t.Fatalf("second ready task = %#v ok=%v; want later", second, ok)
	}
}

func TestDelayedTaskSchedulerBoundsUntrustedNotBeforeAndReadBlock(t *testing.T) {
	now := time.UnixMilli(1_800_000_000_000)
	var scheduled delayedTaskHeap
	if !scheduled.schedule(Task{
		MessageID:          "far-future",
		NotBeforeUnixMilli: now.Add(24 * time.Hour).UnixMilli(),
	}, now) {
		t.Fatal("far-future task was not scheduled")
	}
	if got := scheduled.nextReadBlock(now); got != taskReadBlockTime {
		t.Fatalf("initial read block = %s; want %s", got, taskReadBlockTime)
	}
	nearReady := now.Add(maxTaskReadyWait - 2*time.Second)
	if got := scheduled.nextReadBlock(nearReady); got != 2*time.Second {
		t.Fatalf("near-ready read block = %s; want 2s", got)
	}
	if task, ok := scheduled.popReady(now.Add(maxTaskReadyWait)); !ok || task.MessageID != "far-future" {
		t.Fatalf("bounded task = %#v ok=%v; want far-future ready after %s", task, ok, maxTaskReadyWait)
	}
}

func TestDelayedTaskSchedulerNeverUsesSubMillisecondRedisBlock(t *testing.T) {
	now := time.UnixMilli(1_800_000_000_000)
	scheduled := delayedTaskHeap{{
		task:    Task{MessageID: "almost-ready"},
		readyAt: now.Add(500 * time.Microsecond),
	}}
	if got := scheduled.nextReadBlock(now); got != time.Millisecond {
		t.Fatalf("sub-millisecond read block = %s; want %s", got, time.Millisecond)
	}
}

func TestTaskFromMessageRejectsInvalidTask(t *testing.T) {
	for _, values := range []map[string]any{
		{"payload": `{}`},
		{"type": "image_post", "payload": `{invalid`},
	} {
		if _, err := taskFromMessage(redis.XMessage{ID: "1-0", Values: values}); err == nil {
			t.Fatalf("accepted invalid task values: %#v", values)
		}
	}
}

func TestImageObjectKeyValidation(t *testing.T) {
	if !isRawImageKey("raw/example.jpg") || isRawImageKey("processed/example.jpg") || isRawImageKey("../secret") {
		t.Fatal("raw image key validation did not enforce the raw prefix")
	}
	if !isProcessedImageKey("processed/example.jpg") || isProcessedImageKey("raw/example.jpg") {
		t.Fatal("processed image key validation did not enforce the processed prefix")
	}
}

func TestHandleExhaustedImageTaskCleansBeforeDeadLetter(t *testing.T) {
	task := Task{
		MessageID: "7-0",
		Type:      contracts.TaskProcessImagePost,
		Payload:   json.RawMessage(`{"post_id":7,"user_id":3,"raw_keys":["raw/example.jpg"]}`),
	}
	var order []string
	err := handleExhaustedTask(
		context.Background(), task, maxTaskAttempts, errors.New("decode failed"),
		func(_ context.Context, payload ImagePostPayload) error {
			if payload.PostID != 7 || len(payload.RawKeys) != 1 || payload.RawKeys[0] != "raw/example.jpg" {
				t.Fatalf("unexpected cleanup payload: %#v", payload)
			}
			order = append(order, "cleanup")
			return nil
		},
		func(context.Context, contracts.LoginOwnershipEmailPayload) error { return nil },
		func(_ context.Context, messageID, taskType, payload string, attempts int, cause error) error {
			order = append(order, "dead-letter")
			return nil
		},
	)
	if err != nil {
		t.Fatalf("handleExhaustedTask: %v", err)
	}
	if len(order) != 2 || order[0] != "cleanup" || order[1] != "dead-letter" {
		t.Fatalf("terminal operations occurred out of order: %v", order)
	}
}

func TestHandleExhaustedImageTaskRetainsTaskWhenCleanupFails(t *testing.T) {
	task := Task{
		MessageID: "7-0",
		Type:      contracts.TaskProcessImagePost,
		Payload:   json.RawMessage(`{"post_id":7,"user_id":3,"raw_keys":["raw/example.jpg"]}`),
	}
	deadLetterCalled := false
	err := handleExhaustedTask(
		context.Background(), task, maxTaskAttempts, errors.New("decode failed"),
		func(context.Context, ImagePostPayload) error { return errors.New("storage unavailable") },
		func(context.Context, contracts.LoginOwnershipEmailPayload) error { return nil },
		func(context.Context, string, string, string, int, error) error {
			deadLetterCalled = true
			return nil
		},
	)
	if err == nil || deadLetterCalled {
		t.Fatalf("cleanup failure must leave the task retryable; err=%v deadLetterCalled=%v", err, deadLetterCalled)
	}
}

func TestHandleExhaustedMalformedImageTaskStillReachesDeadLetter(t *testing.T) {
	task := Task{
		MessageID: "8-0",
		Type:      contracts.TaskProcessImagePost,
		Payload:   json.RawMessage(`{"unexpected":true}`),
	}
	finalizerCalled := false
	deadLetterCalled := false
	err := handleExhaustedTask(
		context.Background(), task, maxTaskAttempts, errors.New("invalid payload"),
		func(context.Context, ImagePostPayload) error {
			finalizerCalled = true
			return nil
		},
		func(context.Context, contracts.LoginOwnershipEmailPayload) error { return nil },
		func(context.Context, string, string, string, int, error) error {
			deadLetterCalled = true
			return nil
		},
	)
	if err != nil || finalizerCalled || !deadLetterCalled {
		t.Fatalf("malformed task handling changed; err=%v finalizer=%v deadLetter=%v", err, finalizerCalled, deadLetterCalled)
	}
}

func TestHandleExhaustedOwnershipEmailFinalizesBeforeRedactedDeadLetter(t *testing.T) {
	const email = "owner@example.test"
	const code = "ABCDEFGHJKLMNPQR"
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              email,
		Code:               code,
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	task := Task{MessageID: "9-0", Type: contracts.TaskSendVerificationEmail, Payload: raw}
	var order []string
	err = handleExhaustedTask(
		context.Background(), task, maxTaskAttempts, errors.New("SMTP rejected owner@example.test"),
		func(context.Context, ImagePostPayload) error { return nil },
		func(_ context.Context, got contracts.LoginOwnershipEmailPayload) error {
			if got != payload {
				t.Fatalf("ownership finalizer payload = %#v; want %#v", got, payload)
			}
			order = append(order, "finalize")
			return nil
		},
		func(_ context.Context, _ string, taskType, deadPayload string, _ int, cause error) error {
			order = append(order, "dead-letter")
			if taskType != contracts.TaskSendVerificationEmail {
				t.Fatalf("DLQ task type = %q", taskType)
			}
			for _, secret := range []string{email, code, `"email"`, `"code"`} {
				if strings.Contains(deadPayload, secret) {
					t.Fatalf("DLQ payload contains secret %q: %q", secret, deadPayload)
				}
			}
			if !strings.Contains(deadPayload, `"redacted":true`) || !strings.Contains(deadPayload, payload.ChallengeID) {
				t.Fatalf("DLQ payload lacks safe metadata: %q", deadPayload)
			}
			if cause.Error() != "email delivery failed" {
				t.Fatalf("DLQ cause = %q", cause)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("handleExhaustedTask: %v", err)
	}
	if len(order) != 2 || order[0] != "finalize" || order[1] != "dead-letter" {
		t.Fatalf("terminal operations occurred out of order: %v", order)
	}
}

func TestHandleExhaustedOwnershipEmailRetainsTaskWhenFinalizerFails(t *testing.T) {
	payload, err := json.Marshal(contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "owner@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	finalizerErr := errors.New("Redis unavailable")
	deadLetterCalled := false
	err = handleExhaustedTask(
		context.Background(),
		Task{MessageID: "10-0", Type: contracts.TaskSendVerificationEmail, Payload: payload},
		maxTaskAttempts,
		errors.New("SMTP unavailable"),
		func(context.Context, ImagePostPayload) error { return nil },
		func(context.Context, contracts.LoginOwnershipEmailPayload) error { return finalizerErr },
		func(context.Context, string, string, string, int, error) error {
			deadLetterCalled = true
			return nil
		},
	)
	if !errors.Is(err, finalizerErr) || deadLetterCalled {
		t.Fatalf("finalizer failure must retain task; err=%v deadLetterCalled=%v", err, deadLetterCalled)
	}
}

func TestHandleExhaustedMalformedOwnershipEmailUsesRedactedDeadLetterWithoutFinalizer(t *testing.T) {
	task := Task{
		MessageID: "11-0",
		Type:      contracts.TaskSendVerificationEmail,
		Payload: json.RawMessage(
			`{"purpose":"login_ownership","email":"owner@example.test","code":"ABCDEFGHJKLMNPQR"}`,
		),
	}
	finalizerCalled := false
	deadLetterCalled := false
	err := handleExhaustedTask(
		context.Background(), task, maxTaskAttempts, errors.New("invalid payload"),
		func(context.Context, ImagePostPayload) error { return nil },
		func(context.Context, contracts.LoginOwnershipEmailPayload) error {
			finalizerCalled = true
			return nil
		},
		func(_ context.Context, _ string, _ string, payload string, _ int, cause error) error {
			deadLetterCalled = true
			if strings.Contains(payload, "owner@example.test") || strings.Contains(payload, "ABCDEFGHJKLMNPQR") {
				t.Fatalf("malformed ownership DLQ leaked secret: %q", payload)
			}
			if cause.Error() != "email delivery failed" {
				t.Fatalf("DLQ cause = %q", cause)
			}
			return nil
		},
	)
	if err != nil || finalizerCalled || !deadLetterCalled {
		t.Fatalf("malformed terminal handling err=%v finalizer=%v deadLetter=%v", err, finalizerCalled, deadLetterCalled)
	}
}

func TestVerificationEmailDeadLetterPayloadRedactsAllSecrets(t *testing.T) {
	for _, raw := range []string{
		`{"email":"registration@example.test","code":"ABCDEFGHJKLMNPQR"}`,
		`{"email":"login@example.test","code":"123456","challenge_id":"22222222-2222-4222-8222-222222222222"}`,
	} {
		deadPayload, cause := safeDeadLetterPayload(
			contracts.TaskSendVerificationEmail,
			raw,
			errors.New("SMTP rejected login@example.test"),
		)
		for _, secret := range []string{"registration@example.test", "login@example.test", "ABCDEFGHJKLMNPQR", "123456", `"email"`, `"code"`} {
			if strings.Contains(deadPayload, secret) || strings.Contains(cause.Error(), secret) {
				t.Fatalf("redaction leaked %q: payload=%q cause=%q", secret, deadPayload, cause)
			}
		}
		if !strings.Contains(deadPayload, `"redacted":true`) || cause.Error() != "email delivery failed" {
			t.Fatalf("unsafe DLQ metadata: payload=%q cause=%q", deadPayload, cause)
		}
	}
}

func TestMissingTaskTypeDeadLetterPayloadStillRedactsEmailCode(t *testing.T) {
	const raw = `{"email":"owner@example.test","code":"ABCDEFGHJKLMNPQR"}`
	deadPayload, cause := safeDeadLetterPayload("", raw, errors.New("task type is missing"))
	for _, secret := range []string{"owner@example.test", "ABCDEFGHJKLMNPQR", `"email"`, `"code"`} {
		if strings.Contains(deadPayload, secret) || strings.Contains(cause.Error(), secret) {
			t.Fatalf("missing-type DLQ leaked %q: payload=%q cause=%q", secret, deadPayload, cause)
		}
	}
	if deadPayload != `{"redacted":true}` || cause.Error() != "email delivery failed" {
		t.Fatalf("missing-type DLQ metadata payload=%q cause=%q", deadPayload, cause)
	}
}

func TestGenericDeadLetterRedactsNestedGrantAndInvalidJSON(t *testing.T) {
	for _, raw := range []string{
		`{"metadata":{"password_verification_grant":"secret-grant"}}`,
		`{"credentials":{"api_key":"provider-api-key"}}`,
		`{"truncated_secret":"do-not-store"`,
	} {
		deadPayload, _ := safeDeadLetterPayload("unknown_task", raw, errors.New("decode failed"))
		if deadPayload != `{"redacted":true}` {
			t.Fatalf("generic sensitive payload was retained: input=%q dead=%q", raw, deadPayload)
		}
		for _, secret := range []string{
			"secret-grant", "provider-api-key", "do-not-store",
			"password_verification_grant", "api_key", "truncated_secret",
		} {
			if strings.Contains(deadPayload, secret) {
				t.Fatalf("generic DLQ leaked %q: %q", secret, deadPayload)
			}
		}
	}
}

func TestDeadLetterDoesNotTrustSafeMetadataFieldValues(t *testing.T) {
	raw := `{"purpose":"secret-purpose","challenge_id":"owner@example.test","delivery_id":"ABCDEFGHJKLMNPQR","code":"123456"}`
	deadPayload, _ := safeDeadLetterPayload(contracts.TaskSendVerificationEmail, raw, errors.New("SMTP failure"))
	for _, secret := range []string{"secret-purpose", "owner@example.test", "ABCDEFGHJKLMNPQR", "123456"} {
		if strings.Contains(deadPayload, secret) {
			t.Fatalf("DLQ metadata leaked unvalidated value %q: %q", secret, deadPayload)
		}
	}
	if deadPayload != `{"redacted":true}` {
		t.Fatalf("unsafe DLQ metadata = %q", deadPayload)
	}
}

func TestTaskStreamAddArgsBoundsOnlyDeadLetterStream(t *testing.T) {
	deadLetter := taskStreamAddArgs(taskDeadLetterKey, map[string]any{"type": "image_post"})
	if deadLetter.MaxLen != maxDeadLetterStreamLength || !deadLetter.Approx {
		t.Fatalf("dead-letter stream is not bounded: %#v", deadLetter)
	}
	retry := taskStreamAddArgs(contracts.TaskStreamKey, map[string]any{"type": "image_post"})
	if retry.MaxLen != 0 || retry.Approx {
		t.Fatalf("retry stream semantics changed unexpectedly: %#v", retry)
	}
}
