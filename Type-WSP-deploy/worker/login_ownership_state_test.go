package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/authstate"
	"typewsp/shared/contracts"
)

func TestLoginOwnershipStateScriptsCompareAndSetChallenge(t *testing.T) {
	client := useWorkerIntegrationRedis(t)
	ctx := context.Background()
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "owner@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	}
	activeKey := authstate.LoginOwnershipActiveKey(payload.Email)
	if err := client.HSet(ctx, activeKey,
		"challenge_id", payload.ChallengeID,
		"code", payload.Code,
		"delivery_id", payload.DeliveryID,
		"delivery_state", contracts.LoginOwnershipDeliveryStateQueued,
	).Err(); err != nil {
		t.Fatalf("seed active challenge: %v", err)
	}
	if err := client.PExpire(ctx, activeKey, time.Hour).Err(); err != nil {
		t.Fatalf("expire active challenge: %v", err)
	}

	active, err := redisLoginOwnershipChallengeIsQueued(ctx, payload)
	if err != nil || !active {
		t.Fatalf("matching challenge active=%v err=%v", active, err)
	}

	stale := payload
	stale.ChallengeID = "22222222-2222-4222-8222-222222222222"
	active, err = redisLoginOwnershipChallengeIsQueued(ctx, stale)
	if err != nil || active {
		t.Fatalf("stale challenge active=%v err=%v", active, err)
	}

	olderDelivery := payload
	olderDelivery.DeliveryID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	active, err = redisLoginOwnershipChallengeIsQueued(ctx, olderDelivery)
	if err != nil || !active {
		t.Fatalf("same-challenge older delivery active=%v err=%v", active, err)
	}
	completed, err := redisCompleteLoginOwnershipDelivery(ctx, olderDelivery)
	if err != nil || !completed {
		t.Fatalf("same-challenge older delivery completed=%v err=%v", completed, err)
	}
	if got, err := client.HGet(ctx, activeKey, "delivery_state").Result(); err != nil || got != contracts.LoginOwnershipDeliveryStateDelivered {
		t.Fatalf("current delivery state = %q err=%v; want delivered", got, err)
	}
	if err := client.HSet(ctx, activeKey,
		"delivery_state", contracts.LoginOwnershipDeliveryStateQueued,
		"code", "QRSTUVWXYZ234567",
	).Err(); err != nil {
		t.Fatalf("replace active code: %v", err)
	}
	active, err = redisLoginOwnershipChallengeIsQueued(ctx, payload)
	if err != nil || active {
		t.Fatalf("mismatched-code task active=%v err=%v", active, err)
	}
}

func TestFinalizeLoginOwnershipDeliveryFailureCompensatesOnlyMatchingChallenge(t *testing.T) {
	client := useWorkerIntegrationRedis(t)
	ctx := context.Background()
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "owner@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	}
	activeKey := authstate.LoginOwnershipActiveKey(payload.Email)
	cooldownKey := authstate.LoginOwnershipSendCooldownKey(payload.Email)
	hourlyKey := authstate.LoginOwnershipSendHourlyKey(payload.Email)
	dailyKey := authstate.LoginOwnershipSendDailyKey(payload.Email)
	if err := client.HSet(ctx, activeKey,
		"challenge_id", payload.ChallengeID,
		"code", payload.Code,
		"delivery_id", payload.DeliveryID,
		"delivery_state", contracts.LoginOwnershipDeliveryStateQueued,
	).Err(); err != nil {
		t.Fatalf("seed active challenge: %v", err)
	}
	if err := client.PExpire(ctx, activeKey, time.Hour).Err(); err != nil {
		t.Fatalf("expire active challenge: %v", err)
	}
	if err := client.Set(ctx, cooldownKey, "1", time.Minute).Err(); err != nil {
		t.Fatalf("seed cooldown: %v", err)
	}
	if err := client.Set(ctx, hourlyKey, "3", time.Hour).Err(); err != nil {
		t.Fatalf("seed hourly counter: %v", err)
	}
	if err := client.Set(ctx, dailyKey, "7", 24*time.Hour).Err(); err != nil {
		t.Fatalf("seed daily counter: %v", err)
	}

	if err := finalizeLoginOwnershipDeliveryFailure(ctx, payload); err != nil {
		t.Fatalf("finalize matching challenge: %v", err)
	}
	if exists, _ := client.Exists(ctx, activeKey, cooldownKey).Result(); exists != 1 {
		t.Fatalf("active/cooldown state count = %d; want active only", exists)
	}
	if got, err := client.HGet(ctx, activeKey, "delivery_state").Result(); err != nil || got != contracts.LoginOwnershipDeliveryStateDelivered {
		t.Fatalf("delivery state = %q err=%v; want delivered", got, err)
	}
	if got, _ := client.Get(ctx, hourlyKey).Int64(); got != 2 {
		t.Fatalf("hourly counter = %d; want 2", got)
	}
	if got, _ := client.Get(ctx, dailyKey).Int64(); got != 6 {
		t.Fatalf("daily counter = %d; want 6", got)
	}

	newChallenge := payload.ChallengeID
	newDelivery := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	if err := client.HSet(ctx, activeKey,
		"challenge_id", newChallenge,
		"code", payload.Code,
		"delivery_id", newDelivery,
		"delivery_state", contracts.LoginOwnershipDeliveryStateQueued,
	).Err(); err != nil {
		t.Fatalf("seed newer challenge: %v", err)
	}
	if err := client.PExpire(ctx, activeKey, time.Hour).Err(); err != nil {
		t.Fatalf("expire newer challenge: %v", err)
	}
	if err := client.Set(ctx, cooldownKey, "1", time.Minute).Err(); err != nil {
		t.Fatalf("seed newer cooldown: %v", err)
	}
	if err := finalizeLoginOwnershipDeliveryFailure(ctx, payload); err != nil {
		t.Fatalf("finalize stale challenge: %v", err)
	}
	if got, err := client.HGet(ctx, activeKey, "challenge_id").Result(); err != nil || got != newChallenge {
		t.Fatalf("newer challenge was modified: got=%q err=%v", got, err)
	}
	if got, err := client.HGet(ctx, activeKey, "delivery_id").Result(); err != nil || got != newDelivery {
		t.Fatalf("newer delivery was modified: got=%q err=%v", got, err)
	}
	if got, err := client.HGet(ctx, activeKey, "delivery_state").Result(); err != nil || got != contracts.LoginOwnershipDeliveryStateQueued {
		t.Fatalf("newer delivery state was modified: got=%q err=%v", got, err)
	}
	if exists, _ := client.Exists(ctx, cooldownKey).Result(); exists != 1 {
		t.Fatalf("newer cooldown was removed: exists=%d", exists)
	}
	if got, _ := client.Get(ctx, hourlyKey).Int64(); got != 2 {
		t.Fatalf("stale finalizer changed hourly counter to %d", got)
	}
	if got, _ := client.Get(ctx, dailyKey).Int64(); got != 6 {
		t.Fatalf("stale finalizer changed daily counter to %d", got)
	}
}

func TestFinalizeLoginOwnershipDeliveryFailureValidatesBeforeMutating(t *testing.T) {
	client := useWorkerIntegrationRedis(t)
	ctx := context.Background()
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "atomic-owner@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	}
	activeKey := authstate.LoginOwnershipActiveKey(payload.Email)
	cooldownKey := authstate.LoginOwnershipSendCooldownKey(payload.Email)
	hourlyKey := authstate.LoginOwnershipSendHourlyKey(payload.Email)
	dailyKey := authstate.LoginOwnershipSendDailyKey(payload.Email)
	if err := client.HSet(ctx, activeKey,
		"challenge_id", payload.ChallengeID,
		"code", payload.Code,
		"delivery_id", payload.DeliveryID,
		"delivery_state", contracts.LoginOwnershipDeliveryStateQueued,
	).Err(); err != nil {
		t.Fatalf("seed active challenge: %v", err)
	}
	if err := client.PExpire(ctx, activeKey, time.Hour).Err(); err != nil {
		t.Fatalf("expire active challenge: %v", err)
	}
	if err := client.Set(ctx, cooldownKey, "1", time.Minute).Err(); err != nil {
		t.Fatalf("seed cooldown: %v", err)
	}
	if err := client.Set(ctx, hourlyKey, "3", time.Hour).Err(); err != nil {
		t.Fatalf("seed hourly counter: %v", err)
	}
	if err := client.Set(ctx, dailyKey, "invalid", 24*time.Hour).Err(); err != nil {
		t.Fatalf("seed invalid daily counter: %v", err)
	}

	err := finalizeLoginOwnershipDeliveryFailure(ctx, payload)
	if err == nil {
		t.Fatal("invalid counter was accepted")
	}
	if got, _ := client.HGet(ctx, activeKey, "delivery_state").Result(); got != contracts.LoginOwnershipDeliveryStateQueued {
		t.Fatalf("state partially mutated to %q", got)
	}
	if exists, _ := client.Exists(ctx, cooldownKey).Result(); exists != 1 {
		t.Fatalf("cooldown partially removed: exists=%d", exists)
	}
	if got, _ := client.Get(ctx, hourlyKey).Result(); got != "3" {
		t.Fatalf("hourly counter partially compensated to %q", got)
	}
}

func TestFinalizeLoginOwnershipDeliveryFailureRejectsNonCanonicalCounterBeforeMutating(t *testing.T) {
	client := useWorkerIntegrationRedis(t)
	ctx := context.Background()
	payload := contracts.LoginOwnershipEmailPayload{
		Purpose:            contracts.EmailPurposeLoginOwnership,
		Email:              "noncanonical-owner@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "11111111-1111-4111-8111-111111111111",
		DeliveryID:         "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ExpiresAtUnixMilli: time.Now().Add(time.Hour).UnixMilli(),
	}
	activeKey := authstate.LoginOwnershipActiveKey(payload.Email)
	cooldownKey := authstate.LoginOwnershipSendCooldownKey(payload.Email)
	hourlyKey := authstate.LoginOwnershipSendHourlyKey(payload.Email)
	dailyKey := authstate.LoginOwnershipSendDailyKey(payload.Email)
	if err := client.HSet(ctx, activeKey,
		"challenge_id", payload.ChallengeID,
		"code", payload.Code,
		"delivery_id", payload.DeliveryID,
		"delivery_state", contracts.LoginOwnershipDeliveryStateQueued,
	).Err(); err != nil {
		t.Fatalf("seed active challenge: %v", err)
	}
	if err := client.PExpire(ctx, activeKey, time.Hour).Err(); err != nil {
		t.Fatalf("expire active challenge: %v", err)
	}
	if err := client.Set(ctx, cooldownKey, "1", time.Minute).Err(); err != nil {
		t.Fatalf("seed cooldown: %v", err)
	}
	if err := client.Set(ctx, hourlyKey, "3", time.Hour).Err(); err != nil {
		t.Fatalf("seed hourly counter: %v", err)
	}
	if err := client.Set(ctx, dailyKey, "2.0", 24*time.Hour).Err(); err != nil {
		t.Fatalf("seed non-canonical daily counter: %v", err)
	}

	err := finalizeLoginOwnershipDeliveryFailure(ctx, payload)
	if err == nil {
		t.Fatal("non-canonical counter was accepted")
	}
	if got, _ := client.HGet(ctx, activeKey, "delivery_state").Result(); got != contracts.LoginOwnershipDeliveryStateQueued {
		t.Fatalf("state partially mutated to %q", got)
	}
	if exists, _ := client.Exists(ctx, cooldownKey).Result(); exists != 1 {
		t.Fatalf("cooldown partially removed: exists=%d", exists)
	}
	if got, _ := client.Get(ctx, hourlyKey).Result(); got != "3" {
		t.Fatalf("hourly counter partially compensated to %q", got)
	}
}

func TestMoveTaskToStreamKeepsOriginalWhenDestinationXAddFails(t *testing.T) {
	client := useWorkerIntegrationRedis(t)
	ctx := context.Background()
	if err := client.XGroupCreateMkStream(ctx, contracts.TaskStreamKey, taskConsumerGroup, "0").Err(); err != nil {
		t.Fatalf("create task group: %v", err)
	}
	messageID, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: contracts.TaskStreamKey,
		Values: map[string]any{"type": "test", "payload": `{}`, "attempts": "0"},
	}).Result()
	if err != nil {
		t.Fatalf("add original task: %v", err)
	}
	if _, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: taskConsumerGroup, Consumer: "test-consumer",
		Streams: []string{contracts.TaskStreamKey, ">"}, Count: 1,
	}).Result(); err != nil {
		t.Fatalf("read original task: %v", err)
	}
	if err := client.Set(ctx, taskDeadLetterKey, "wrong type", 0).Err(); err != nil {
		t.Fatalf("poison DLQ destination: %v", err)
	}

	err = moveTaskToStream(ctx, taskDeadLetterKey, messageID, map[string]any{
		"type": "test", "payload": `{}`, "attempts": maxTaskAttempts,
	})
	if err == nil {
		t.Fatal("wrong-type DLQ destination was accepted")
	}
	if length, err := client.XLen(ctx, contracts.TaskStreamKey).Result(); err != nil || length != 1 {
		t.Fatalf("original stream length = %d err=%v; want 1", length, err)
	}
	pending, err := client.XPending(ctx, contracts.TaskStreamKey, taskConsumerGroup).Result()
	if err != nil || pending.Count != 1 {
		t.Fatalf("pending count = %#v err=%v; want 1", pending, err)
	}
}

func TestDelayedPendingTaskDoesNotBlockReadingNextReadyTask(t *testing.T) {
	client := useWorkerIntegrationRedis(t)
	ctx := context.Background()
	if err := client.XGroupCreateMkStream(ctx, contracts.TaskStreamKey, taskConsumerGroup, "0").Err(); err != nil {
		t.Fatalf("create task group: %v", err)
	}
	delayedID, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: contracts.TaskStreamKey,
		Values: map[string]any{
			"type": "test", "payload": `{}`, "attempts": "1",
			"not_before_unix_milli": time.Now().Add(30 * time.Second).UnixMilli(),
		},
	}).Result()
	if err != nil {
		t.Fatalf("add delayed task: %v", err)
	}
	readyID, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: contracts.TaskStreamKey,
		Values: map[string]any{"type": "test", "payload": `{}`, "attempts": "0"},
	}).Result()
	if err != nil {
		t.Fatalf("add ready task: %v", err)
	}

	first, ok, err := readNewTask(ctx, "test-consumer", time.Second)
	if err != nil || !ok || first.ID != delayedID {
		t.Fatalf("first task = %#v ok=%v err=%v; want delayed %s", first, ok, err, delayedID)
	}
	var scheduled delayedTaskHeap
	handleTaskMessage(ctx, first, &scheduled, time.Now())
	if scheduled.Len() != 1 {
		t.Fatalf("scheduled task count = %d; want 1", scheduled.Len())
	}
	if pending, err := client.XPending(ctx, contracts.TaskStreamKey, taskConsumerGroup).Result(); err != nil || pending.Count != 1 {
		t.Fatalf("pending after scheduling = %#v err=%v; want 1", pending, err)
	}

	second, ok, err := readNewTask(ctx, "test-consumer", time.Second)
	if err != nil || !ok || second.ID != readyID {
		t.Fatalf("second task = %#v ok=%v err=%v; want ready %s", second, ok, err, readyID)
	}
}

func useWorkerIntegrationRedis(t *testing.T) *redis.Client {
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
	ctx := context.Background()
	if err := client.FlushDB(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("flush worker Redis: %v", err)
	}
	previous := rdb
	rdb = client
	t.Cleanup(func() {
		rdb = previous
		_ = client.Close()
	})
	return client
}
