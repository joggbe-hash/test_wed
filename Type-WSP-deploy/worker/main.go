package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

// Task 是 Redis task_queue 裡的通用工作格式，Type 決定要解析成哪種 Payload。
type Task struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

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

func main() {
	cfg := LoadConfig()
	ctx := context.Background()

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parse Redis URL failed: %v", err)
	}
	rdb = redis.NewClient(opt)

	minioClient, err = minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  minioCreds.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
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

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser, cfg.PostgresPassword,
		cfg.PostgresHost, cfg.PostgresPort, cfg.DBSystem,
	)
	systemPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect system_db failed: %v", err)
	}
	defer systemPool.Close()

	log.Println("Worker started, waiting for Redis tasks")

	for {
		result, err := rdb.BLPop(ctx, 0, "task_queue").Result()
		if err != nil {
			log.Printf("BLPop failed: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var task Task
		if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
			log.Printf("decode task failed: %v", err)
			continue
		}

		switch task.Type {
		case "process_image_post":
			var payload ImagePostPayload
			if err := json.Unmarshal(task.Payload, &payload); err != nil {
				log.Printf("decode ImagePostPayload failed: %v", err)
				continue
			}
			processImagePost(ctx, payload)

		case "send_verification_email":
			var payload EmailPayload
			if err := json.Unmarshal(task.Payload, &payload); err != nil {
				log.Printf("decode EmailPayload failed: %v", err)
				continue
			}
			handleSendEmail(payload)

		default:
			log.Printf("unknown task type: %s", task.Type)
		}
	}
}

func handleSendEmail(payload EmailPayload) {
	log.Printf("[email] send verification email to %s, code: %s", payload.Email, payload.Code)
}
