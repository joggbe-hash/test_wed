package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"golang.org/x/image/draw"
)

const maxDimension = 1920
const jpegQuality = 85

// processImagePost 處理帶有圖片的發文任務
// API 已將貼文寫入 DB（status=processing），此函式負責：
// 1. 逐一下載原始圖片 → 壓縮/縮圖 → 上傳處理後圖片
// 2. 更新 DB 記錄（image_url 改為處理後路徑，status 改為 ready）
// 3. 清理原始圖片
func processImagePost(ctx context.Context, payload ImagePostPayload) {
	ops := NewAtomicRollback()
	var processedKeys []string

	// 逐一處理每張圖片
	for _, rawKey := range payload.RawKeys {
		// 從 MinIO 下載原始圖片
		rawObj, err := minioClient.GetObject(ctx, minioBucket, rawKey, minio.GetObjectOptions{})
		if err != nil {
			log.Printf("[image] 下載原始圖片失敗 key=%s: %v", rawKey, err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		rawData, err := io.ReadAll(rawObj)
		rawObj.Close()
		if err != nil {
			log.Printf("[image] 讀取原始圖片失敗: %v", err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		// 壓縮/縮圖
		processed, err := resizeAndCompress(rawData)
		if err != nil {
			log.Printf("[image] 圖片處理失敗: %v", err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		// 上傳處理後的圖片
		processedKey := "processed/" + rawKey[4:]
		reader := bytes.NewReader(processed)
		_, err = minioClient.PutObject(ctx, minioBucket, processedKey, reader, int64(len(processed)),
			minio.PutObjectOptions{ContentType: "image/jpeg"})
		if err != nil {
			log.Printf("[image] 上傳處理後圖片失敗: %v", err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		processedKeys = append(processedKeys, processedKey)
		capturedKey := processedKey
		ops.Add("刪除處理後圖片 "+processedKey, func() error {
			return minioClient.RemoveObject(ctx, minioBucket, capturedKey, minio.RemoveObjectOptions{})
		})
	}

	// 更新 DB：將 image_url 改為處理後的路徑陣列，status 改為 ready
	processedJSON, _ := json.Marshal(processedKeys)
	_, err := systemPool.Exec(ctx,
		`UPDATE posts SET image_url = $1, image_status = 'ready' WHERE id = $2`,
		string(processedJSON), payload.PostID,
	)
	if err != nil {
		log.Printf("[image] 更新貼文 DB 失敗: %v", err)
		ops.Execute()
		return
	}

	// 清理原始圖片（非關鍵操作）
	for _, rawKey := range payload.RawKeys {
		if err := minioClient.RemoveObject(ctx, minioBucket, rawKey, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("[image] 清理原始圖片失敗（非致命）: %v", err)
		}
	}

	// 清除 feed 快取
	rdb.Del(ctx, "feed:latest")

	// 透過 Redis Pub/Sub 通知前端（API 的 WebSocket 會轉發給該使用者）
	notifyChannel := fmt.Sprintf("notify:user:%d", payload.UserID)
	notifyMsg := fmt.Sprintf(`{"type":"post_ready","post_id":%d}`, payload.PostID)
	rdb.Publish(ctx, notifyChannel, notifyMsg)

	log.Printf("[image] 貼文 #%d 處理完成，共 %d 張圖片", payload.PostID, len(processedKeys))
}

// markPostFailed 將貼文的圖片狀態標記為 failed
func markPostFailed(ctx context.Context, postID int) {
	systemPool.Exec(ctx, `UPDATE posts SET image_status = 'failed' WHERE id = $1`, postID)
}

// resizeAndCompress 解碼圖片、等比例縮放、以 JPEG 壓縮輸出
func resizeAndCompress(data []byte) ([]byte, error) {
	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("圖片解碼失敗（格式: %s）: %w", format, err)
	}

	bounds := src.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	newW, newH := origW, origH
	if origW > maxDimension || origH > maxDimension {
		if origW >= origH {
			newW = maxDimension
			newH = origH * maxDimension / origW
		} else {
			newH = maxDimension
			newW = origW * maxDimension / origH
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("JPEG 編碼失敗: %w", err)
	}

	return buf.Bytes(), nil
}

func init() {
	image.RegisterFormat("png", "\x89PNG", png.Decode, png.DecodeConfig)
}
