package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

const maxTaskStreamLength = 10_000

var (
	errTaskQueueFull  = errors.New("task queue is full")
	enqueueTaskScript = redis.NewScript(`
local maxLength = tonumber(ARGV[1])
if redis.call('XLEN', KEYS[1]) >= maxLength then
  return 0
end
redis.call('XADD', KEYS[1], '*', 'type', ARGV[2], 'payload', ARGV[3], 'attempts', '0')
return 1
`)
)

func enqueueTask(ctx context.Context, taskType string, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s task payload failed: %w", taskType, err)
	}

	accepted, err := enqueueTaskScript.Run(ctx, rdb, []string{contracts.TaskStreamKey}, maxTaskStreamLength, taskType, string(payloadJSON)).Int()
	if err != nil {
		return fmt.Errorf("enqueue %s task failed: %w", taskType, err)
	}
	if accepted == 0 {
		return errTaskQueueFull
	}

	return nil
}
