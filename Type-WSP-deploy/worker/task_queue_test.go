package main

import (
	"testing"

	"github.com/redis/go-redis/v9"
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
