package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"typewsp/shared/authstate"
	"typewsp/shared/contracts"
)

var (
	checkLoginOwnershipChallenge   = redisLoginOwnershipChallengeIsQueued
	completeLoginOwnershipDelivery = redisCompleteLoginOwnershipDelivery
)

var loginOwnershipDeliveryIsQueuedScript = redis.NewScript(`
local activeType = redis.call('TYPE', KEYS[1])
if type(activeType) == 'table' then activeType = activeType['ok'] end
if activeType == 'none' then return 0 end
if activeType ~= 'hash' then return redis.error_reply('unexpected login ownership challenge type') end
if redis.call('PTTL', KEYS[1]) <= 0 then return 0 end
if redis.call('HGET', KEYS[1], ARGV[1]) ~= ARGV[2]
  or redis.call('HGET', KEYS[1], ARGV[3]) ~= ARGV[4]
  or redis.call('HGET', KEYS[1], ARGV[5]) ~= ARGV[6] then
  return 0
end
return 1
`)

var completeLoginOwnershipDeliveryScript = redis.NewScript(`
local activeType = redis.call('TYPE', KEYS[1])
if type(activeType) == 'table' then activeType = activeType['ok'] end
if activeType == 'none' then return 0 end
if activeType ~= 'hash' then return redis.error_reply('unexpected login ownership challenge type') end
if redis.call('PTTL', KEYS[1]) <= 0 then return 0 end
if redis.call('HGET', KEYS[1], ARGV[1]) ~= ARGV[2]
  or redis.call('HGET', KEYS[1], ARGV[3]) ~= ARGV[4]
  or redis.call('HGET', KEYS[1], ARGV[5]) ~= ARGV[6] then
  return 0
end
redis.call('HSET', KEYS[1], ARGV[3], ARGV[7])
return 1
`)

var finalizeLoginOwnershipDeliveryFailureScript = redis.NewScript(`
local function keyType(key)
  local result = redis.call('TYPE', key)
  if type(result) == 'table' then return result['ok'] end
  return result
end
local function canonicalNonnegativeInteger(raw)
  if not string.match(raw, '^%d+$') then return false end
  if #raw > 1 and string.sub(raw, 1, 1) == '0' then return false end
  if #raw > 19 then return false end
  if #raw == 19 and raw > '9223372036854775807' then return false end
  return true
end
local activeType = keyType(KEYS[1])
if activeType == 'none' then return 0 end
if activeType ~= 'hash' then return redis.error_reply('unexpected login ownership challenge type') end
if redis.call('PTTL', KEYS[1]) <= 0
  or redis.call('HGET', KEYS[1], ARGV[1]) ~= ARGV[4]
  or redis.call('HGET', KEYS[1], ARGV[2]) ~= ARGV[5]
  or redis.call('HGET', KEYS[1], ARGV[3]) ~= ARGV[6] then
  return 0
end

for index = 2, 4 do
  local actual = keyType(KEYS[index])
  if actual ~= 'none' and actual ~= 'string' then
    return redis.error_reply('unexpected login ownership delivery counter type')
  end
end
local counters = {}
for index = 3, 4 do
  local raw = redis.call('GET', KEYS[index])
  if raw then
    if not canonicalNonnegativeInteger(raw) or redis.call('PTTL', KEYS[index]) <= 0 then
      return redis.error_reply('invalid login ownership delivery counter')
    end
    counters[index] = raw
  end
end

redis.call('HSET', KEYS[1], ARGV[3], ARGV[7])
redis.call('DEL', KEYS[2])
for index = 3, 4 do
  local value = counters[index]
  if value then
    if value == '0' or value == '1' then redis.call('DEL', KEYS[index]) else redis.call('DECR', KEYS[index]) end
  end
end
return 1
`)

func handleSendLoginOwnershipEmail(ctx context.Context, payload contracts.LoginOwnershipEmailPayload) error {
	return handleSendLoginOwnershipEmailAt(ctx, payload, time.Now())
}

func handleSendLoginOwnershipEmailAt(ctx context.Context, payload contracts.LoginOwnershipEmailPayload, now time.Time) error {
	if !validLoginOwnershipEmailPayload(payload) {
		return fmt.Errorf("invalid login ownership email payload")
	}

	expiresAt := time.UnixMilli(payload.ExpiresAtUnixMilli)
	if !expiresAt.After(now) {
		log.Printf("expired login ownership email skipped recipient=%s", maskEmail(payload.Email))
		return nil
	}

	queued, err := checkLoginOwnershipChallenge(ctx, payload)
	if err != nil {
		return fmt.Errorf("check login ownership delivery state failed: %w", err)
	}
	if !queued {
		log.Printf("stale login ownership email skipped recipient=%s", maskEmail(payload.Email))
		return nil
	}

	if err := mailSender.SendLoginOwnershipCode(ctx, payload.Email, payload.Code, expiresAt); err != nil {
		return fmt.Errorf("send login ownership email failed: %w", err)
	}
	completed, err := completeLoginOwnershipDelivery(ctx, payload)
	if err != nil {
		return fmt.Errorf("complete login ownership delivery failed: %w", err)
	}
	if completed {
		log.Printf("login ownership email sent recipient=%s", maskEmail(payload.Email))
	}
	return nil
}

func validLoginOwnershipEmailPayload(payload contracts.LoginOwnershipEmailPayload) bool {
	return payload.Purpose == contracts.EmailPurposeLoginOwnership &&
		strings.TrimSpace(payload.Email) != "" &&
		validCanonicalUUID(payload.ChallengeID) &&
		validCanonicalUUID(payload.DeliveryID) &&
		payload.ExpiresAtUnixMilli > 0 &&
		validLoginOwnershipCode(payload.Code)
}

func validCanonicalUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

func redisLoginOwnershipChallengeIsQueued(ctx context.Context, payload contracts.LoginOwnershipEmailPayload) (bool, error) {
	result, err := loginOwnershipDeliveryIsQueuedScript.Run(
		ctx,
		rdb,
		[]string{authstate.LoginOwnershipActiveKey(payload.Email)},
		authstate.LoginOwnershipChallengeIDField,
		payload.ChallengeID,
		authstate.LoginOwnershipDeliveryStateField,
		contracts.LoginOwnershipDeliveryStateQueued,
		authstate.LoginOwnershipCodeField,
		payload.Code,
	).Int64()
	return result == 1, err
}

func redisCompleteLoginOwnershipDelivery(ctx context.Context, payload contracts.LoginOwnershipEmailPayload) (bool, error) {
	result, err := completeLoginOwnershipDeliveryScript.Run(
		ctx,
		rdb,
		[]string{authstate.LoginOwnershipActiveKey(payload.Email)},
		authstate.LoginOwnershipChallengeIDField,
		payload.ChallengeID,
		authstate.LoginOwnershipDeliveryStateField,
		contracts.LoginOwnershipDeliveryStateQueued,
		authstate.LoginOwnershipCodeField,
		payload.Code,
		contracts.LoginOwnershipDeliveryStateDelivered,
	).Int64()
	return result == 1, err
}

func finalizeLoginOwnershipDeliveryFailure(ctx context.Context, payload contracts.LoginOwnershipEmailPayload) error {
	if !validLoginOwnershipEmailPayload(payload) {
		return fmt.Errorf("invalid login ownership email payload")
	}
	_, err := finalizeLoginOwnershipDeliveryFailureScript.Run(
		ctx,
		rdb,
		[]string{
			authstate.LoginOwnershipActiveKey(payload.Email),
			authstate.LoginOwnershipSendCooldownKey(payload.Email),
			authstate.LoginOwnershipSendHourlyKey(payload.Email),
			authstate.LoginOwnershipSendDailyKey(payload.Email),
		},
		authstate.LoginOwnershipChallengeIDField,
		authstate.LoginOwnershipDeliveryIDField,
		authstate.LoginOwnershipDeliveryStateField,
		payload.ChallengeID,
		payload.DeliveryID,
		contracts.LoginOwnershipDeliveryStateQueued,
		contracts.LoginOwnershipDeliveryStateDelivered,
	).Int64()
	if err != nil {
		return fmt.Errorf("finalize login ownership delivery failed: %w", err)
	}
	return nil
}
