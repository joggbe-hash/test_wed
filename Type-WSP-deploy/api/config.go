package main

import "os"

// Config 集中管理所有外部服務的連線參數，全部從環境變數讀取
type Config struct {
	// PostgreSQL 連線參數
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	DBSystem         string // system_db：存放貼文等系統資料
	DBUser           string // user_db：存放使用者帳號資料

	// Redis 連線字串（格式：redis://:password@host:port/db）
	RedisURL string

	// MinIO 物件儲存連線參數
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string // 統一的上傳 bucket 名稱

	// Session 加密金鑰與存活時間
	SecretKey  string
	SessionTTL int // 秒，預設 86400（24 小時）
}

// LoadConfig 從環境變數載入設定，若未設定則使用開發用預設值
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

// envOr 讀取環境變數，若不存在則回傳 fallback 值
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
