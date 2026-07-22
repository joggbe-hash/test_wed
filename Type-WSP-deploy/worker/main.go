package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	minioCreds "github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

var (
	rdb         *redis.Client
	minioClient *minio.Client
	systemPool  *pgxpool.Pool
	minioBucket string
)

// ImagePostPayload 由 API 建立圖文貼文後送進 queue，worker 負責處理 raw_keys。
type ImagePostPayload struct {
	PostID  int      `json:"post_id"`
	UserID  int      `json:"user_id"`
	RawKeys []string `json:"raw_keys"`
}

// EmailPayload 第一版先只印出驗證碼；之後可以在這裡接 SMTP 或第三方寄信服務。
type EmailPayload struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type ImageDeletePayload struct {
	Keys []string `json:"keys"`
}

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("load worker config failed: %v", err)
	}
	ctx := context.Background()

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parse Redis URL failed: %v", err)
	}
	rdb = redis.NewClient(opt)

	minioClient, err = minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  minioCreds.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioSecure,
	})
	if err != nil {
		log.Fatalf("connect MinIO failed: %v", err)
	}
	minioBucket = cfg.MinioBucket

	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err != nil {
		log.Fatalf("check MinIO bucket failed: %v", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("create MinIO bucket failed: %v", err)
		}
	}

	dsn := workerPostgresDSN(cfg)
	systemPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect system_db failed: %v", err)
	}
	defer systemPool.Close()

	consumerName, err := os.Hostname()
	if err != nil || consumerName == "" {
		consumerName = fmt.Sprintf("worker-%d", os.Getpid())
	}
	log.Printf("Worker started, consumer=%s", consumerName)
	if err := runTaskWorker(ctx, consumerName); err != nil {
		log.Fatalf("task worker stopped: %v", err)
	}
}

func workerPostgresDSN(cfg *Config) string {
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.PostgresUser, cfg.PostgresPassword),
		Host:   net.JoinHostPort(cfg.PostgresHost, cfg.PostgresPort),
		Path:   cfg.DBSystem,
	}
	query := dsn.Query()
	query.Set("sslmode", cfg.PostgresSSLMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func handleSendEmail(payload EmailPayload) error {
	if payload.Email == "" || payload.Code == "" {
		return fmt.Errorf("invalid email payload")
	}
	log.Printf("[email] verification email job accepted for %s", payload.Email)
	return nil
}
