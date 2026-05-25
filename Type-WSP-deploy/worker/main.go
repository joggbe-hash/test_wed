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

// Task 代表從 Redis 佇列取出的任務
type Task struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ImagePostPayload 圖片貼文處理任務的參數
// API 已經把貼文寫入 DB（status=processing），Worker 負責處理圖片後更新該記錄
type ImagePostPayload struct {
	PostID  int      `json:"post_id"`
	UserID  int      `json:"user_id"`
	RawKeys []string `json:"raw_keys"`
}

// EmailPayload 寄送驗證碼信件的任務參數
type EmailPayload struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func main() {
	cfg := LoadConfig()

	// 初始化 Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Redis URL 解析失敗: %v", err)
	}
	rdb = redis.NewClient(opt)

	// 初始化 MinIO
	minioClient, err = minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  minioCreds.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("MinIO 連線失敗: %v", err)
	}
	minioBucket = cfg.MinioBucket

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err != nil {
		log.Fatalf("檢查 MinIO bucket 失敗: %v", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("建立 MinIO bucket 失敗: %v", err)
		}
	}

	// 初始化 PostgreSQL
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser, cfg.PostgresPassword,
		cfg.PostgresHost, cfg.PostgresPort, cfg.DBSystem,
	)
	systemPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("無法連線 system_db: %v", err)
	}
	defer systemPool.Close()

	log.Println("Worker 已啟動，等待任務...")

	// 主迴圈：從 Redis 佇列取出任務並分派處理
	for {
		result, err := rdb.BLPop(ctx, 0, "task_queue").Result()
		if err != nil {
			log.Printf("BLPop 錯誤: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var task Task
		if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
			log.Printf("任務 JSON 解析失敗: %v", err)
			continue
		}

		switch task.Type {
		case "process_image_post":
			var payload ImagePostPayload
			if err := json.Unmarshal(task.Payload, &payload); err != nil {
				log.Printf("ImagePostPayload 解析失敗: %v", err)
				continue
			}
			processImagePost(ctx, payload)

		case "send_verification_email":
			var payload EmailPayload
			if err := json.Unmarshal(task.Payload, &payload); err != nil {
				log.Printf("EmailPayload 解析失敗: %v", err)
				continue
			}
			handleSendEmail(payload)

		default:
			log.Printf("未知的任務類型: %s", task.Type)
		}
	}
}

// handleSendEmail 處理寄送驗證碼信件（目前為開發用佔位實作）
func handleSendEmail(payload EmailPayload) {
	log.Printf("[email] 驗證碼已寄送至 %s（驗證碼: %s）", payload.Email, payload.Code)
}
