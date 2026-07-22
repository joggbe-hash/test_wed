package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// API 目前分兩個資料庫：user_db 管帳號，system_db 管貼文與系統資料。
var (
	userPool   *pgxpool.Pool
	systemPool *pgxpool.Pool
)

// InitDB 建立兩組 pgx connection pool，讓 handler 可以依資料邊界選擇資料庫。
func InitDB(cfg *Config) {
	var err error

	userDSN := postgresDSN(cfg, cfg.DBUser)
	userPool, err = pgxpool.New(context.Background(), userDSN)
	if err != nil {
		log.Fatalf("connect user_db failed: %v", err)
	}

	systemDSN := postgresDSN(cfg, cfg.DBSystem)
	systemPool, err = pgxpool.New(context.Background(), systemDSN)
	if err != nil {
		log.Fatalf("connect system_db failed: %v", err)
	}

	log.Println("PostgreSQL pools initialized")
}

func postgresDSN(cfg *Config, database string) string {
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.PostgresUser, cfg.PostgresPassword),
		Host:   net.JoinHostPort(cfg.PostgresHost, cfg.PostgresPort),
		Path:   database,
	}
	query := dsn.Query()
	query.Set("sslmode", cfg.PostgresSSLMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func CloseDB() {
	if userPool != nil {
		userPool.Close()
	}
	if systemPool != nil {
		systemPool.Close()
	}
}

// WithTx 包住一段交易流程；fn 回傳 error 時 rollback，成功時 commit。
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
