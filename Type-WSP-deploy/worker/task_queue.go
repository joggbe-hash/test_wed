package main

import (
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

var moveTaskToStreamScript = redis.NewScript(`
local command = {'XADD', KEYS[2]}
local maxLength = tonumber(ARGV[3])
if maxLength and maxLength > 0 then
  table.insert(command, 'MAXLEN')
  table.insert(command, '~')
  table.insert(command, tostring(maxLength))
end
table.insert(command, '*')
for index = 4, #ARGV do table.insert(command, ARGV[index]) end
local added = redis.pcall(unpack(command))
if type(added) == 'table' and added['err'] then
  return redis.error_reply(added['err'])
end
redis.call('XACK', KEYS[1], ARGV[1], ARGV[2])
redis.call('XDEL', KEYS[1], ARGV[2])
return added
`)

const (
	taskDeadLetterKey         = "task_stream:dead_letter"
	taskConsumerGroup         = "type-wsp-workers"
	maxDeadLetterStreamLength = 10_000
	maxTaskAttempts           = 3
	staleTaskIdleTime         = 5 * time.Minute
	taskReadBlockTime         = 5 * time.Second
	loginOwnershipRetryBase   = 15 * time.Second
	maxTaskReadyWait          = 45 * time.Second
)

type Task struct {
	MessageID          string
	Type               string
	Payload            json.RawMessage
	Attempts           int
	NotBeforeUnixMilli int64
}

type delayedTask struct {
	task    Task
	readyAt time.Time
}

// delayedTaskHeap keeps not-yet-ready stream entries in memory without
// acknowledging them; a worker crash therefore leaves Redis able to reclaim
// the pending entries through XAUTOCLAIM.
type delayedTaskHeap []delayedTask

func (tasks delayedTaskHeap) Len() int { return len(tasks) }

func (tasks delayedTaskHeap) Less(left, right int) bool {
	if tasks[left].readyAt.Equal(tasks[right].readyAt) {
		return tasks[left].task.MessageID < tasks[right].task.MessageID
	}
	return tasks[left].readyAt.Before(tasks[right].readyAt)
}

func (tasks delayedTaskHeap) Swap(left, right int) {
	tasks[left], tasks[right] = tasks[right], tasks[left]
}

func (tasks *delayedTaskHeap) Push(value any) {
	*tasks = append(*tasks, value.(delayedTask))
}

func (tasks *delayedTaskHeap) Pop() any {
	items := *tasks
	last := len(items) - 1
	item := items[last]
	items[last] = delayedTask{}
	*tasks = items[:last]
	return item
}

func (tasks *delayedTaskHeap) schedule(task Task, now time.Time) bool {
	if task.NotBeforeUnixMilli <= 0 {
		return false
	}
	readyAt := time.UnixMilli(task.NotBeforeUnixMilli)
	if !readyAt.After(now) {
		return false
	}
	if latest := now.Add(maxTaskReadyWait); readyAt.After(latest) {
		readyAt = latest
	}
	heap.Push(tasks, delayedTask{task: task, readyAt: readyAt})
	return true
}

func (tasks *delayedTaskHeap) popReady(now time.Time) (Task, bool) {
	if tasks.Len() == 0 || (*tasks)[0].readyAt.After(now) {
		return Task{}, false
	}
	return heap.Pop(tasks).(delayedTask).task, true
}

func (tasks delayedTaskHeap) nextReadBlock(now time.Time) time.Duration {
	if len(tasks) == 0 {
		return taskReadBlockTime
	}
	wait := tasks[0].readyAt.Sub(now)
	if wait < time.Millisecond {
		return time.Millisecond
	}
	if wait < taskReadBlockTime {
		return wait
	}
	return taskReadBlockTime
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

	var delayedTasks delayedTaskHeap
	for {
		if task, ok := delayedTasks.popReady(time.Now()); ok {
			handleReadyTask(ctx, task)
			continue
		}
		message, ok, err := claimStaleTask(ctx, consumerName)
		if err != nil {
			log.Printf("claim stale task failed: %v", err)
		}
		if !ok {
			message, ok, err = readNewTask(ctx, consumerName, delayedTasks.nextReadBlock(time.Now()))
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
			handleTaskMessage(ctx, message, &delayedTasks, time.Now())
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

func readNewTask(ctx context.Context, consumerName string, block time.Duration) (redis.XMessage, bool, error) {
	streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: taskConsumerGroup, Consumer: consumerName,
		Streams: []string{contracts.TaskStreamKey, ">"}, Count: 1, Block: block,
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
	notBeforeUnixMilli := int64(0)
	if raw, ok := messageValue(message, "not_before_unix_milli"); ok && raw != "" {
		notBeforeUnixMilli, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || notBeforeUnixMilli < 0 {
			return Task{}, fmt.Errorf("task not-before is invalid")
		}
	}

	return Task{
		MessageID:          message.ID,
		Type:               taskType,
		Payload:            json.RawMessage(payload),
		Attempts:           attempts,
		NotBeforeUnixMilli: notBeforeUnixMilli,
	}, nil
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

func handleTaskMessage(ctx context.Context, message redis.XMessage, delayedTasks *delayedTaskHeap, now time.Time) {
	task, err := taskFromMessage(message)
	if err != nil {
		if deadLetterErr := deadLetterMessage(ctx, message, err); deadLetterErr != nil {
			log.Printf("dead-letter invalid task failed: %v", deadLetterErr)
		}
		return
	}
	if delayedTasks.schedule(task, now) {
		return
	}
	handleReadyTask(ctx, task)
}

func handleReadyTask(ctx context.Context, task Task) {
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
		if payload.JobID != "" {
			return processImageDeletionJob(ctx, payload.JobID)
		}
		return deleteImages(ctx, payload)
	case contracts.TaskSendVerificationEmail:
		var payload EmailPayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil {
			return fmt.Errorf("decode EmailPayload failed: %w", err)
		}
		if payload.Purpose == contracts.EmailPurposeLoginOwnership {
			return handleSendLoginOwnershipEmail(ctx, contracts.LoginOwnershipEmailPayload{
				Purpose:            payload.Purpose,
				Email:              payload.Email,
				Code:               payload.Code,
				ChallengeID:        payload.ChallengeID,
				DeliveryID:         payload.DeliveryID,
				ExpiresAtUnixMilli: payload.ExpiresAtUnixMilli,
			})
		}
		if payload.Purpose != "" {
			return fmt.Errorf("unknown verification email purpose %q", payload.Purpose)
		}
		return handleSendEmail(ctx, payload)
	default:
		return fmt.Errorf("unknown task type %q", task.Type)
	}
}

func retryOrDeadLetterTask(ctx context.Context, task Task, cause error) error {
	nextAttempt := task.Attempts + 1
	if nextAttempt < maxTaskAttempts {
		if err := moveTaskToStream(
			ctx,
			contracts.TaskStreamKey,
			task.MessageID,
			retryTaskValuesAt(task, nextAttempt, time.Now()),
		); err != nil {
			return fmt.Errorf("enqueue retry failed: %w", err)
		}
		log.Printf("task %s failed; retry %d/%d: %s", task.MessageID, nextAttempt, maxTaskAttempts-1, safeTaskError(task, cause))
		return nil
	}

	return handleExhaustedTask(
		ctx,
		task,
		nextAttempt,
		cause,
		finalizeFailedImagePost,
		finalizeLoginOwnershipDeliveryFailure,
		moveTaskToDeadLetter,
	)
}

func retryTaskValuesAt(task Task, nextAttempt int, now time.Time) map[string]any {
	values := map[string]any{
		"type": task.Type, "payload": string(task.Payload), "attempts": nextAttempt,
	}
	if isLoginOwnershipEmailTask(task.Type, task.Payload) {
		delay := loginOwnershipRetryBase * time.Duration(1<<max(nextAttempt-1, 0))
		values["not_before_unix_milli"] = now.Add(delay).UnixMilli()
	}
	return values
}

func isLoginOwnershipEmailTask(taskType string, payload json.RawMessage) bool {
	if taskType != contracts.TaskSendVerificationEmail {
		return false
	}
	var metadata struct {
		Purpose string `json:"purpose"`
	}
	return json.Unmarshal(payload, &metadata) == nil && metadata.Purpose == contracts.EmailPurposeLoginOwnership
}

func safeTaskError(task Task, cause error) string {
	if task.Type == contracts.TaskSendVerificationEmail {
		return "email delivery failed"
	}
	return cause.Error()
}

type imageFailureFinalizer func(context.Context, ImagePostPayload) error
type loginOwnershipFailureFinalizer func(context.Context, contracts.LoginOwnershipEmailPayload) error
type deadLetterWriter func(context.Context, string, string, string, int, error) error

func handleExhaustedTask(
	ctx context.Context,
	task Task,
	attempts int,
	cause error,
	finalizeImage imageFailureFinalizer,
	finalizeLoginOwnership loginOwnershipFailureFinalizer,
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
	if isLoginOwnershipEmailTask(task.Type, task.Payload) {
		var payload contracts.LoginOwnershipEmailPayload
		if json.Unmarshal(task.Payload, &payload) == nil && validLoginOwnershipEmailPayload(payload) {
			if err := finalizeLoginOwnership(ctx, payload); err != nil {
				return fmt.Errorf("finalize exhausted login ownership email: %w", err)
			}
		}
	}
	deadPayload, deadCause := safeDeadLetterPayload(task.Type, string(task.Payload), cause)
	return writeDeadLetter(ctx, task.MessageID, task.Type, deadPayload, attempts, deadCause)
}

func deadLetterMessage(ctx context.Context, message redis.XMessage, cause error) error {
	taskType, _ := messageValue(message, "type")
	payload, _ := messageValue(message, "payload")
	deadPayload, deadCause := safeDeadLetterPayload(taskType, payload, cause)
	return moveTaskToDeadLetter(ctx, message.ID, taskType, deadPayload, maxTaskAttempts, deadCause)
}

func safeDeadLetterPayload(taskType, payload string, cause error) (string, error) {
	var raw any
	validJSON := json.Unmarshal([]byte(payload), &raw) == nil
	shouldRedact := !validJSON ||
		taskType == "" ||
		taskType == contracts.TaskSendVerificationEmail ||
		(validJSON && containsSensitiveTaskField(raw))
	if !shouldRedact {
		return payload, cause
	}
	metadata := struct {
		Purpose     string `json:"purpose,omitempty"`
		ChallengeID string `json:"challenge_id,omitempty"`
		DeliveryID  string `json:"delivery_id,omitempty"`
		Redacted    bool   `json:"redacted"`
	}{Redacted: true}
	var decoded struct {
		Purpose     string `json:"purpose"`
		ChallengeID string `json:"challenge_id"`
		DeliveryID  string `json:"delivery_id"`
	}
	if json.Unmarshal([]byte(payload), &decoded) == nil {
		if decoded.Purpose == contracts.EmailPurposeLoginOwnership {
			metadata.Purpose = decoded.Purpose
		}
		if validCanonicalUUID(decoded.ChallengeID) {
			metadata.ChallengeID = decoded.ChallengeID
		}
		if validCanonicalUUID(decoded.DeliveryID) {
			metadata.DeliveryID = decoded.DeliveryID
		}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return `{"redacted":true}`, errors.New("email delivery failed")
	}
	return string(encoded), errors.New("email delivery failed")
}

func containsSensitiveTaskField(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalizedKey := strings.ToLower(key)
			if normalizedKey == "email" || normalizedKey == "code" ||
				strings.Contains(normalizedKey, "api_key") ||
				strings.Contains(normalizedKey, "apikey") ||
				strings.Contains(normalizedKey, "password") ||
				strings.Contains(normalizedKey, "secret") ||
				strings.Contains(normalizedKey, "token") ||
				strings.Contains(normalizedKey, "grant") {
				return true
			}
			if containsSensitiveTaskField(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsSensitiveTaskField(child) {
				return true
			}
		}
	}
	return false
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
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	maxLength := taskStreamAddArgs(destination, values).MaxLen
	arguments := make([]any, 0, 3+len(keys)*2)
	arguments = append(arguments, taskConsumerGroup, messageID, maxLength)
	for _, key := range keys {
		arguments = append(arguments, key, values[key])
	}
	return moveTaskToStreamScript.Run(
		ctx,
		rdb,
		[]string{contracts.TaskStreamKey, destination},
		arguments...,
	).Err()
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
