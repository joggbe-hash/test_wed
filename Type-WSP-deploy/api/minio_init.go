package main

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// 全域 MinIO 客戶端，用於圖片上傳與讀取
var minioClient *minio.Client

// minioBucket 儲存目前使用的 bucket 名稱
var minioBucket string

// InitMinio 建立 MinIO 連線並確保上傳用的 bucket 存在
func InitMinio(cfg *Config) {
	var err error
	minioClient, err = minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false, // 容器內部通訊不需 TLS
	})
	if err != nil {
		log.Fatalf("MinIO 連線失敗: %v", err)
	}

	minioBucket = cfg.MinioBucket

	// 若 bucket 不存在則自動建立
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err != nil {
		log.Fatalf("檢查 MinIO bucket 失敗: %v", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("建立 MinIO bucket 失敗: %v", err)
		}
		log.Printf("已建立 MinIO bucket: %s", minioBucket)
	}

	log.Println("MinIO 連線已建立")
}
