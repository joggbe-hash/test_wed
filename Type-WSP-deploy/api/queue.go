package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

func enqueueTask(ctx context.Context, taskType string, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s task payload failed: %w", taskType, err)
	}

	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: contracts.TaskStreamKey,
		Values: map[string]any{
			"type":     taskType,
			"payload":  string(payloadJSON),
			"attempts": 0,
		},
	}).Err(); err != nil {
		return fmt.Errorf("enqueue %s task failed: %w", taskType, err)
	}

	return nil
}
