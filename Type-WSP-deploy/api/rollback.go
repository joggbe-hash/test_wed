package main

import "log"

type rollbackStep struct {
	desc string
	fn   func() error
}

// AtomicRollback 用在跨系統操作，例如「先上傳 MinIO，再寫 DB，再推 Redis job」。
// 任何一步失敗時，Execute 會反向執行已登記的補償動作。
type AtomicRollback struct {
	steps []rollbackStep
}

func NewAtomicRollback() *AtomicRollback {
	return &AtomicRollback{}
}

func (r *AtomicRollback) Add(desc string, fn func() error) {
	r.steps = append(r.steps, rollbackStep{desc: desc, fn: fn})
}

func (r *AtomicRollback) Execute() {
	for i := len(r.steps) - 1; i >= 0; i-- {
		step := r.steps[i]
		if err := step.fn(); err != nil {
			log.Printf("[rollback FAILED] %s: %v", step.desc, err)
		} else {
			log.Printf("[rollback OK] %s", step.desc)
		}
	}
}
