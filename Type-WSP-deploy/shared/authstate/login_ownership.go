package authstate

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	LoginOwnershipActiveKeyPrefix       = "login:ownership:active:"
	LoginOwnershipSendCooldownKeyPrefix = "login:ownership:send-cooldown:"
	LoginOwnershipSendHourlyKeyPrefix   = "login:ownership:send-hourly:"
	LoginOwnershipSendDailyKeyPrefix    = "login:ownership:send-daily:"

	LoginOwnershipChallengeIDField   = "challenge_id"
	LoginOwnershipCodeField          = "code"
	LoginOwnershipDeliveryIDField    = "delivery_id"
	LoginOwnershipDeliveryStateField = "delivery_state"
)

func LoginOwnershipActiveKey(email string) string {
	return loginOwnershipEmailKey(LoginOwnershipActiveKeyPrefix, email)
}

func LoginOwnershipSendCooldownKey(email string) string {
	return loginOwnershipEmailKey(LoginOwnershipSendCooldownKeyPrefix, email)
}

func LoginOwnershipSendHourlyKey(email string) string {
	return loginOwnershipEmailKey(LoginOwnershipSendHourlyKeyPrefix, email)
}

func LoginOwnershipSendDailyKey(email string) string {
	return loginOwnershipEmailKey(LoginOwnershipSendDailyKeyPrefix, email)
}

func loginOwnershipEmailKey(prefix, email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	digest := sha256.Sum256([]byte(normalized))
	return prefix + hex.EncodeToString(digest[:])
}
