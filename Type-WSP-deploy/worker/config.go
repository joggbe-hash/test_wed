package main

import "os"

// Config 是 worker 啟動所需設定；worker 不處理登入，所以不需要 user_db/session 設定。
type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	DBSystem         string

	RedisURL string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
}

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
