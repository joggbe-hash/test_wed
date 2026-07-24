package contracts

import "fmt"

const (
	TaskStreamKey        = "task_stream"
	FeedCacheKey         = "feed:latest"
	RawImagePrefix       = "raw/"
	ProcessedImagePrefix = "processed/"

	TaskProcessImagePost      = "process_image_post"
	TaskDeleteImages          = "delete_images"
	TaskSendVerificationEmail = "send_verification_email"
)

func NotifyUserChannel(userID int) string {
	return fmt.Sprintf("notify:user:%d", userID)
}
