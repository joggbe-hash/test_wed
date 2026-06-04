package main

import (
	"log"

	"github.com/redis/go-redis/v9"
)

// rdb 是 API 共用 Redis client，用於 session、驗證碼、feed cache、task queue 和 pub/sub。
var rdb *redis.Client

func InitRedis(cfg *Config) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parse Redis URL failed: %v", err)
	}
	rdb = redis.NewClient(opt)
	log.Println("Redis client initialized")
}
