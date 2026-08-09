package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

func TestTaskFromMessage(t *testing.T) {
	message := redis.XMessage{ID: "1-0", Values: map[string]any{
		"type": "image_post", "payload": `{"post_id":7}`, "attempts": "2",
	}}
	task, err := taskFromMessage(message)
	if err != nil {
		t.Fatalf("taskFromMessage: %v", err)
	}
	if task.MessageID != "1-0" || task.Type != "image_post" || task.Attempts != 2 {
		t.Fatalf("unexpected task: %#v", task)
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
		func(context.Context, string, string, string, int, error) error {
			deadLetterCalled = true
			return nil
		},
	)
	if err != nil || finalizerCalled || !deadLetterCalled {
		t.Fatalf("malformed task handling changed; err=%v finalizer=%v deadLetter=%v", err, finalizerCalled, deadLetterCalled)
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
