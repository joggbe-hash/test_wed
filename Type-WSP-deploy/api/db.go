package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 全域資料庫連線池，分別對應 user_db 和 system_db
var (
	userPool   *pgxpool.Pool // user_db：使用者帳號相關
	systemPool *pgxpool.Pool // system_db：貼文等系統資料
)

// InitDB 根據設定檔建立兩個資料庫的連線池
// 連線池由 pgx 自動管理連線數量與回收
func InitDB(cfg *Config) {
	var err error

	userDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser, cfg.PostgresPassword,
		cfg.PostgresHost, cfg.PostgresPort, cfg.DBUser,
	)
	userPool, err = pgxpool.New(context.Background(), userDSN)
	if err != nil {
		log.Fatalf("無法連線 user_db: %v", err)
	}

	systemDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser, cfg.PostgresPassword,
		cfg.PostgresHost, cfg.PostgresPort, cfg.DBSystem,
	)
	systemPool, err = pgxpool.New(context.Background(), systemDSN)
	if err != nil {
		log.Fatalf("無法連線 system_db: %v", err)
	}

	log.Println("資料庫連線池已建立")
}

// CloseDB 關閉所有資料庫連線池，應在程式結束時呼叫
func CloseDB() {
	if userPool != nil {
		userPool.Close()
	}
	if systemPool != nil {
		systemPool.Close()
	}
}

// WithTx 在指定連線池上開啟一個交易，執行 fn 函式
// 若 fn 回傳 error 或 panic，交易自動 rollback
// 若 fn 正常結束，交易自動 commit
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("開啟交易失敗: %w", err)
	}
	// defer 確保不論成功或失敗都會正確結束交易
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
