package main

import "os"

// Config 集中管理 Worker 所需的外部服務連線參數
type Config struct {
	// PostgreSQL 連線參數
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	DBSystem         string // system_db：Worker 寫入處理完成的貼文

	// Redis 連線字串
	RedisURL string

	// MinIO 物件儲存連線參數
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
}

// LoadConfig 從環境變數載入 Worker 設定
func LoadConfig() *Config {
	return &Config{
		PostgresHost:     envOr("POSTGRES_HOST", "localhost"),
		PostgresPort:     envOr("POSTGRES_PORT", "5432"),
		PostgresUser:     envOr("POSTGRES_USER", "app_admin"),
		PostgresPassword: envOr("POSTGRES_PASSWORD", "change_me"),
		DBSystem:         envOr("POSTGRES_DB_SYSTEM", "system_db"),
		RedisURL:         envOr("REDIS_URL", "redis://localhost:6379/0"),
		MinioEndpoint:    envOr("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:   envOr("MINIO_ACCESS_KEY", "minio_admin"),
		MinioSecretKey:   envOr("MINIO_SECRET_KEY", "change_me"),
		MinioBucket:      envOr("MINIO_BUCKET", "uploads"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
