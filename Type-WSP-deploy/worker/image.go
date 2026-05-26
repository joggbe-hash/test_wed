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
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
)

const maxDimension = 4096
const jpegQuality = 95

type processedImage struct {
	data        []byte
	contentType string
	extension   string
}

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
		processedKey := "processed/" + rawKey[4:len(rawKey)-len(filepath.Ext(rawKey))] + processed.extension
		reader := bytes.NewReader(processed.data)
		_, err = minioClient.PutObject(ctx, minioBucket, processedKey, reader, int64(len(processed.data)),
			minio.PutObjectOptions{ContentType: processed.contentType})
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

// resizeAndCompress 解碼圖片，只有超過尺寸上限才等比例縮放，避免小圖被二次壓縮。
func resizeAndCompress(data []byte) (*processedImage, error) {
	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("圖片解碼失敗（格式: %s）: %w", format, err)
	}
	orientation := imageOrientation(data)
	src = applyOrientation(src, orientation)

	bounds := src.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	contentType, extension := imageOutputInfo(format)

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

	if newW == origW && newH == origH && orientation == 1 {
		return &processedImage{
			data:        data,
			contentType: contentType,
			extension:   extension,
		}, nil
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if format == "png" {
		if err := png.Encode(&buf, dst); err != nil {
			return nil, fmt.Errorf("PNG 編碼失敗: %w", err)
		}
		return &processedImage{
			data:        buf.Bytes(),
			contentType: "image/png",
			extension:   ".png",
		}, nil
	}

	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("JPEG 編碼失敗: %w", err)
	}

	return &processedImage{
		data:        buf.Bytes(),
		contentType: "image/jpeg",
		extension:   ".jpg",
	}, nil
}

func imageOutputInfo(format string) (string, string) {
	if format == "png" {
		return "image/png", ".png"
	}
	return "image/jpeg", ".jpg"
}

func imageOrientation(data []byte) int {
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return 1
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	orientation, err := tag.Int(0)
	if err != nil || orientation < 1 || orientation > 8 {
		return 1
	}
	return orientation
}

func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation == 1 {
		return src
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dstW, dstH := w, h
	if orientation >= 5 && orientation <= 8 {
		dstW, dstH = h, w
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			var sx, sy int
			switch orientation {
			case 2:
				sx, sy = w-1-x, y
			case 3:
				sx, sy = w-1-x, h-1-y
			case 4:
				sx, sy = x, h-1-y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, h-1-x
			case 7:
				sx, sy = w-1-y, h-1-x
			case 8:
				sx, sy = w-1-y, x
			default:
				sx, sy = x, y
			}
			dst.Set(x, y, src.At(bounds.Min.X+sx, bounds.Min.Y+sy))
		}
	}
	return dst
}

func init() {
	image.RegisterFormat("png", "\x89PNG", png.Decode, png.DecodeConfig)
}
