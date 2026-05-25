package main

import "log"

// rollbackStep 代表一個可回滾的補償操作
type rollbackStep struct {
	desc string
	fn   func() error
}

// AtomicRollback 蒐集跨服務操作的補償動作，
// 發生錯誤時按反向順序逐一回滾
type AtomicRollback struct {
	steps []rollbackStep
}

// NewAtomicRollback 建立一個新的原子回滾追蹤器
func NewAtomicRollback() *AtomicRollback {
	return &AtomicRollback{}
}

// Add 註冊一個補償操作
func (r *AtomicRollback) Add(desc string, fn func() error) {
	r.steps = append(r.steps, rollbackStep{desc: desc, fn: fn})
}

// Execute 反向執行所有已註冊的補償操作
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
