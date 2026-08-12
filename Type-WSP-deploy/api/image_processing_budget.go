package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"typewsp/shared/contracts"
)

const (
	imageProcessingBudgetPrefix           = "image:processing-pixels:"
	imageProcessingBudgetWindow           = 15 * time.Minute
	maxImageProcessingPixelsPerUserWindow = int64(maxImagesPerPost) * int64(contracts.MaxImagePixels)
)

var (
	reserveImageProcessingBudgetScript = redis.NewScript(`
local requested = tonumber(ARGV[1])
local maximum = tonumber(ARGV[2])
if not requested or requested <= 0 or requested > maximum then
  return 0
end
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if current + requested > maximum then
  return 0
end
redis.call('INCRBY', KEYS[1], requested)
if redis.call('PTTL', KEYS[1]) < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[3])
end
return 1
`)
	releaseImageProcessingBudgetScript = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local amount = tonumber(ARGV[1])
if not amount or amount <= 0 or current <= amount then
  redis.call('DEL', KEYS[1])
  return 0
end
return redis.call('DECRBY', KEYS[1], amount)
`)
)

func imageProcessingBudgetKey(userID int) string {
	return fmt.Sprintf("%s%d", imageProcessingBudgetPrefix, userID)
}

func hasImageProcessingCapacity(currentPixels, requestedPixels int64) bool {
	return currentPixels >= 0 && requestedPixels > 0 &&
		requestedPixels <= maxImageProcessingPixelsPerUserWindow &&
		currentPixels <= maxImageProcessingPixelsPerUserWindow-requestedPixels
}

func reserveImageProcessingBudget(ctx context.Context, userID int, pixels int64) error {
	if userID <= 0 || !hasImageProcessingCapacity(0, pixels) {
		return errImageProcessingQuotaExceeded
	}
	accepted, err := reserveImageProcessingBudgetScript.Run(
		ctx,
		rdb,
		[]string{imageProcessingBudgetKey(userID)},
		pixels,
		maxImageProcessingPixelsPerUserWindow,
		imageProcessingBudgetWindow.Milliseconds(),
	).Int()
	if err != nil {
		return fmt.Errorf("reserve image processing budget: %w", err)
	}
	if accepted == 0 {
		return errImageProcessingQuotaExceeded
	}
	return nil
}

func releaseImageProcessingBudget(ctx context.Context, userID int, pixels int64) error {
	if userID <= 0 || pixels <= 0 {
		return nil
	}
	if err := releaseImageProcessingBudgetScript.Run(
		ctx,
		rdb,
		[]string{imageProcessingBudgetKey(userID)},
		pixels,
	).Err(); err != nil {
		return fmt.Errorf("release image processing budget: %w", err)
	}
	return nil
}
