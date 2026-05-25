package main

import "log"

// rollbackStep 代表一個可回滾的補償操作
type rollbackStep struct {
	desc string       // 這個步驟的描述（用於日誌）
	fn   func() error // 回滾時要執行的補償函式
}

// AtomicRollback 蒐集多個跨服務操作的補償動作，
// 當任一步驟失敗時，按照反向順序逐一執行回滾
type AtomicRollback struct {
	steps []rollbackStep
}

// NewAtomicRollback 建立一個新的原子回滾追蹤器
func NewAtomicRollback() *AtomicRollback {
	return &AtomicRollback{}
}

// Add 註冊一個補償操作，在回滾時會反向執行
// desc 用於日誌記錄，fn 是實際的補償函式
func (r *AtomicRollback) Add(desc string, fn func() error) {
	r.steps = append(r.steps, rollbackStep{desc: desc, fn: fn})
}

// Execute 按照註冊的反向順序執行所有補償操作
// 即使某個補償操作失敗，仍會繼續執行剩餘的操作
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
