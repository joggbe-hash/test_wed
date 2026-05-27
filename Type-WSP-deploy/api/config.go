package main

import "os"

// Config 集中管理 API 需要的環境變數；Docker Compose 會覆蓋這裡的開發預設值。
type Config struct {
	// PostgreSQL 連線設定。system_db 存貼文，user_db 存帳號。
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	DBSystem         string
	DBUser           string

	// Redis 同時用於 session、驗證碼、工作佇列和 WebSocket 通知。
	RedisURL string

	// MinIO 用於儲存原始圖片與 worker 壓縮後的圖片。
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string

	// Session cookie 會保存 signed session id，實際 user data 放 Redis。
	SecretKey  string
	SessionTTL int
}

// LoadConfig 從環境變數讀取設定；本機直接跑 API 時會使用 fallback。
func LoadConfig() *Config {
	return &Config{
		PostgresHost:     envOr("POSTGRES_HOST", "localhost"),
		PostgresPort:     envOr("POSTGRES_PORT", "5432"),
		PostgresUser:     envOr("POSTGRES_USER", "app_admin"),
		PostgresPassword: envOr("POSTGRES_PASSWORD", "change_me"),
		DBSystem:         envOr("POSTGRES_DB_SYSTEM", "system_db"),
		DBUser:           envOr("POSTGRES_DB_USER", "user_db"),
		RedisURL:         envOr("REDIS_URL", "redis://localhost:6379/0"),
		MinioEndpoint:    envOr("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:   envOr("MINIO_ACCESS_KEY", "minio_admin"),
		MinioSecretKey:   envOr("MINIO_SECRET_KEY", "change_me"),
		MinioBucket:      envOr("MINIO_BUCKET", "uploads"),
		SecretKey:        envOr("SECRET_KEY", "dev-secret-change-in-production"),
		SessionTTL:       86400,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
