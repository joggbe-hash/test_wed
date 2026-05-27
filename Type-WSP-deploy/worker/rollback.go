package main

import "log"

type rollbackStep struct {
	desc string
	fn   func() error
}

// AtomicRollback 讓 worker 在圖片處理流程失敗時，可以清掉已寫入的 processed/ 檔案。
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
