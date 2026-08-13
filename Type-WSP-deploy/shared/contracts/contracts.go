package contracts

import "fmt"

const (
	TaskStreamKey          = "task_stream"
	FeedCacheKey           = "feed:latest"
	RawImagePrefix         = "raw/"
	ProcessedImagePrefix   = "processed/"
	MaxProcessedImageBytes = 16 << 20
	MaxImagePixels         = 12_000_000

	TaskProcessImagePost      = "process_image_post"
	TaskDeleteImages          = "delete_images"
	TaskSendVerificationEmail = "send_verification_email"

	EmailPurposeLoginOwnership           = "login_ownership"
	LoginOwnershipDeliveryStateQueued    = "queued"
	LoginOwnershipDeliveryStateDelivered = "delivered"
)

type LoginOwnershipEmailPayload struct {
	Purpose            string `json:"purpose"`
	Email              string `json:"email"`
	Code               string `json:"code"`
	ChallengeID        string `json:"challenge_id"`
	DeliveryID         string `json:"delivery_id"`
	ExpiresAtUnixMilli int64  `json:"expires_at_unix_milli"`
}

func NotifyUserChannel(userID int) string {
	return fmt.Sprintf("notify:user:%d", userID)
}
