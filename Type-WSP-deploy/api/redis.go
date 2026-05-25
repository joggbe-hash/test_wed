package main

import (
	"log"

	"github.com/redis/go-redis/v9"
)

// 全域 Redis 客戶端，用於 session 儲存、驗證碼暫存、任務佇列
var rdb *redis.Client

// InitRedis 解析 Redis URL 並建立連線
func InitRedis(cfg *Config) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Redis URL 解析失敗: %v", err)
	}
	rdb = redis.NewClient(opt)
	log.Println("Redis 連線已建立")
}
