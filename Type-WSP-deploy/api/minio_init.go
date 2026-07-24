package main

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// minioClient 由 API 上傳原始圖片與讀取處理後圖片共用。
var minioClient *minio.Client

// minioBucket 是專案共用 bucket，raw/ 與 processed/ 以 key prefix 區分。
var minioBucket string

func InitMinio(cfg *Config) {
	var err error
	minioClient, err = minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioSecure,
	})
	if err != nil {
		log.Fatalf("connect MinIO failed: %v", err)
	}

	minioBucket = cfg.MinioBucket

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err != nil {
		log.Fatalf("check MinIO bucket failed: %v", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("create MinIO bucket failed: %v", err)
		}
		log.Printf("created MinIO bucket: %s", minioBucket)
	}

	log.Println("MinIO client initialized")
}
