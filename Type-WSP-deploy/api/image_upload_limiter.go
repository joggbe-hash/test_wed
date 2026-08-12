package main

type imageUploadLimiter struct {
	slots chan struct{}
}

var imageUploadConcurrency = newImageUploadLimiter(maxConcurrentImageUploads)

func newImageUploadLimiter(limit int) *imageUploadLimiter {
	if limit <= 0 {
		panic("image upload concurrency limit must be positive")
	}
	return &imageUploadLimiter{slots: make(chan struct{}, limit)}
}

func (limiter *imageUploadLimiter) tryAcquire() bool {
	select {
	case limiter.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (limiter *imageUploadLimiter) release() {
	<-limiter.slots
}
