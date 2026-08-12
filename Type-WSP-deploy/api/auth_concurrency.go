package main

type concurrencyLimiter struct {
	slots chan struct{}
}

func newConcurrencyLimiter(limit int) *concurrencyLimiter {
	if limit <= 0 {
		panic("concurrency limit must be positive")
	}
	return &concurrencyLimiter{slots: make(chan struct{}, limit)}
}

func (limiter *concurrencyLimiter) tryAcquire() bool {
	select {
	case limiter.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (limiter *concurrencyLimiter) release() {
	<-limiter.slots
}
