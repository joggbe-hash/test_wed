package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

const (
	taskDeadLetterKey         = "task_stream:dead_letter"
	taskConsumerGroup         = "type-wsp-workers"
	maxDeadLetterStreamLength = 10_000
	maxTaskAttempts           = 3
	staleTaskIdleTime         = 5 * time.Minute
	taskReadBlockTime         = 5 * time.Second
)

type Task struct {
	MessageID string
	Type      string
	Payload   json.RawMessage
	Attempts  int
}

func ensureTaskConsumerGroup(ctx context.Context) error {
	err := rdb.XGroupCreateMkStream(ctx, contracts.TaskStreamKey, taskConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create task consumer group failed: %w", err)
	}
	return nil
}

func runTaskWorker(ctx context.Context, consumerName string) error {
	if err := ensureTaskConsumerGroup(ctx); err != nil {
		return err
	}

	for {
		message, ok, err := claimStaleTask(ctx, consumerName)
		if err != nil {
			log.Printf("claim stale task failed: %v", err)
		}
		if !ok {
			message, ok, err = readNewTask(ctx, consumerName)
			if errors.Is(err, redis.Nil) {
				continue
			}
			if err != nil {
				log.Printf("read task stream failed: %v", err)
				time.Sleep(time.Second)
				continue
			}
		}
		if ok {
			handleTaskMessage(ctx, message)
		}
	}
}

func claimStaleTask(ctx context.Context, consumerName string) (redis.XMessage, bool, error) {
	messages, _, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream: contracts.TaskStreamKey, Group: taskConsumerGroup, Consumer: consumerName,
		MinIdle: staleTaskIdleTime, Start: "0-0", Count: 1,
	}).Result()
	if err != nil || len(messages) == 0 {
		return redis.XMessage{}, false, err
	}
	return messages[0], true, nil
}

func readNewTask(ctx context.Context, consumerName string) (redis.XMessage, bool, error) {
	streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: taskConsumerGroup, Consumer: consumerName,
		Streams: []string{contracts.TaskStreamKey, ">"}, Count: 1, Block: taskReadBlockTime,
	}).Result()
	if err != nil || len(streams) == 0 || len(streams[0].Messages) == 0 {
		return redis.XMessage{}, false, err
	}
	return streams[0].Messages[0], true, nil
}

func taskFromMessage(message redis.XMessage) (Task, error) {
	taskType, ok := messageValue(message, "type")
	if !ok || taskType == "" {
		return Task{}, fmt.Errorf("task type is missing")
	}
	payload, ok := messageValue(message, "payload")
	if !ok || !json.Valid([]byte(payload)) {
		return Task{}, fmt.Errorf("task payload is invalid")
	}
	attemptsRaw, _ := messageValue(message, "attempts")
	attempts, err := strconv.Atoi(attemptsRaw)
	if err != nil || attempts < 0 {
		attempts = 0
	}

	return Task{MessageID: message.ID, Type: taskType, Payload: json.RawMessage(payload), Attempts: attempts}, nil
}

func messageValue(message redis.XMessage, key string) (string, bool) {
	value, ok := message.Values[key]
	if !ok {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	default:
		return fmt.Sprint(typed), true
	}
}

func handleTaskMessage(ctx context.Context, message redis.XMessage) {
	task, err := taskFromMessage(message)
	if err != nil {
		if deadLetterErr := deadLetterMessage(ctx, message, err); deadLetterErr != nil {
			log.Printf("dead-letter invalid task failed: %v", deadLetterErr)
		}
		return
	}

	if err := processTask(ctx, task); err != nil {
		if retryErr := retryOrDeadLetterTask(ctx, task, err); retryErr != nil {
			log.Printf("retry task %s failed: %v", task.MessageID, retryErr)
		}
		return
	}

	if err := acknowledgeTask(ctx, task.MessageID); err != nil {
		log.Printf("acknowledge task %s failed: %v", task.MessageID, err)
	}
}

func processTask(ctx context.Context, task Task) error {
	switch task.Type {
	case contracts.TaskProcessImagePost:
		var payload ImagePostPayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil {
			return fmt.Errorf("decode ImagePostPayload failed: %w", err)
		}
		return processImagePost(ctx, payload)
	case contracts.TaskDeleteImages:
		var payload ImageDeletePayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil {
			return fmt.Errorf("decode ImageDeletePayload failed: %w", err)
		}
		return deleteImages(ctx, payload)
	case contracts.TaskSendVerificationEmail:
		var payload EmailPayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil {
			return fmt.Errorf("decode EmailPayload failed: %w", err)
		}
		return handleSendEmail(ctx, payload)
	default:
		return fmt.Errorf("unknown task type %q", task.Type)
	}
}

func retryOrDeadLetterTask(ctx context.Context, task Task, cause error) error {
	nextAttempt := task.Attempts + 1
	if nextAttempt < maxTaskAttempts {
		if err := moveTaskToStream(ctx, contracts.TaskStreamKey, task.MessageID, map[string]any{
			"type": task.Type, "payload": string(task.Payload), "attempts": nextAttempt,
		}); err != nil {
			return fmt.Errorf("enqueue retry failed: %w", err)
		}
		log.Printf("task %s failed; retry %d/%d: %v", task.MessageID, nextAttempt, maxTaskAttempts-1, cause)
		return nil
	}

	return handleExhaustedTask(ctx, task, nextAttempt, cause, finalizeFailedImagePost, moveTaskToDeadLetter)
}

type imageFailureFinalizer func(context.Context, ImagePostPayload) error
type deadLetterWriter func(context.Context, string, string, string, int, error) error

func handleExhaustedTask(
	ctx context.Context,
	task Task,
	attempts int,
	cause error,
	finalizeImage imageFailureFinalizer,
	writeDeadLetter deadLetterWriter,
) error {
	if task.Type == contracts.TaskProcessImagePost {
		var payload ImagePostPayload
		if json.Unmarshal(task.Payload, &payload) == nil && payload.PostID > 0 && len(payload.RawKeys) > 0 {
			if err := finalizeImage(ctx, payload); err != nil {
				return fmt.Errorf("finalize exhausted image task: %w", err)
			}
		}
	}
	return writeDeadLetter(ctx, task.MessageID, task.Type, string(task.Payload), attempts, cause)
}

func deadLetterMessage(ctx context.Context, message redis.XMessage, cause error) error {
	taskType, _ := messageValue(message, "type")
	payload, _ := messageValue(message, "payload")
	return moveTaskToDeadLetter(ctx, message.ID, taskType, payload, maxTaskAttempts, cause)
}

func moveTaskToDeadLetter(
	ctx context.Context,
	messageID string,
	taskType string,
	payload string,
	attempts int,
	cause error,
) error {
	return moveTaskToStream(ctx, taskDeadLetterKey, messageID, map[string]any{
		"type": taskType, "payload": payload, "attempts": attempts,
		"error": cause.Error(), "failed_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func moveTaskToStream(
	ctx context.Context,
	destination string,
	messageID string,
	values map[string]any,
) error {
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.XAdd(ctx, taskStreamAddArgs(destination, values))
		pipe.XAck(ctx, contracts.TaskStreamKey, taskConsumerGroup, messageID)
		pipe.XDel(ctx, contracts.TaskStreamKey, messageID)
		return nil
	})
	return err
}

func taskStreamAddArgs(destination string, values map[string]any) *redis.XAddArgs {
	args := &redis.XAddArgs{Stream: destination, Values: values}
	if destination == taskDeadLetterKey {
		args.MaxLen = maxDeadLetterStreamLength
		args.Approx = true
	}
	return args
}

func acknowledgeTask(ctx context.Context, messageID string) error {
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.XAck(ctx, contracts.TaskStreamKey, taskConsumerGroup, messageID)
		pipe.XDel(ctx, contracts.TaskStreamKey, messageID)
		return nil
	})
	return err
}
