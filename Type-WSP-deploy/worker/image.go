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

// processImagePost 是圖文貼文的非同步流程：
// 1. 從 MinIO raw/ 讀原圖。
// 2. 修正 EXIF 方向，必要時縮圖並重新壓縮。
// 3. 把處理後的圖片存到 processed/，更新 DB 為 ready，並通知前端。
func processImagePost(ctx context.Context, payload ImagePostPayload) {
	ops := NewAtomicRollback()
	var processedKeys []string

	for _, rawKey := range payload.RawKeys {
		rawObj, err := minioClient.GetObject(ctx, minioBucket, rawKey, minio.GetObjectOptions{})
		if err != nil {
			log.Printf("[image] get raw object failed key=%s: %v", rawKey, err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		rawData, err := io.ReadAll(rawObj)
		rawObj.Close()
		if err != nil {
			log.Printf("[image] read raw object failed: %v", err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		processed, err := resizeAndCompress(rawData)
		if err != nil {
			log.Printf("[image] process image failed: %v", err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		processedKey := "processed/" + rawKey[4:len(rawKey)-len(filepath.Ext(rawKey))] + processed.extension
		reader := bytes.NewReader(processed.data)
		_, err = minioClient.PutObject(ctx, minioBucket, processedKey, reader, int64(len(processed.data)),
			minio.PutObjectOptions{ContentType: processed.contentType})
		if err != nil {
			log.Printf("[image] put processed object failed: %v", err)
			markPostFailed(ctx, payload.PostID)
			ops.Execute()
			return
		}

		processedKeys = append(processedKeys, processedKey)
		capturedKey := processedKey
		ops.Add("remove processed object "+processedKey, func() error {
			return minioClient.RemoveObject(ctx, minioBucket, capturedKey, minio.RemoveObjectOptions{})
		})
	}

	processedJSON, _ := json.Marshal(processedKeys)
	_, err := systemPool.Exec(ctx,
		`UPDATE posts SET image_url = $1, image_status = 'ready' WHERE id = $2`,
		string(processedJSON), payload.PostID,
	)
	if err != nil {
		log.Printf("[image] update post image state failed: %v", err)
		ops.Execute()
		return
	}

	for _, rawKey := range payload.RawKeys {
		if err := minioClient.RemoveObject(ctx, minioBucket, rawKey, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("[image] remove raw object failed, ignored: %v", err)
		}
	}

	rdb.Del(ctx, "feed:latest")

	notifyChannel := fmt.Sprintf("notify:user:%d", payload.UserID)
	notifyMsg := fmt.Sprintf(`{"type":"post_ready","post_id":%d}`, payload.PostID)
	rdb.Publish(ctx, notifyChannel, notifyMsg)

	log.Printf("[image] post #%d processed, files=%d", payload.PostID, len(processedKeys))
}

func markPostFailed(ctx context.Context, postID int) {
	systemPool.Exec(ctx, `UPDATE posts SET image_status = 'failed' WHERE id = $1`, postID)
}

// resizeAndCompress 保留原圖格式：PNG 繼續輸出 PNG，其餘格式輸出 JPEG。
// 小於 maxDimension 且方向正常時直接回傳原始 bytes，避免不必要的畫質損失。
func resizeAndCompress(data []byte) (*processedImage, error) {
	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image failed, format=%s: %w", format, err)
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
			return nil, fmt.Errorf("encode PNG failed: %w", err)
		}
		return &processedImage{
			data:        buf.Bytes(),
			contentType: "image/png",
			extension:   ".png",
		}, nil
	}

	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("encode JPEG failed: %w", err)
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
