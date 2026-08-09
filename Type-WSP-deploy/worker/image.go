package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
	"typewsp/shared/contracts"
	"typewsp/shared/rollback"
)

const maxDimension = 4096
const jpegQuality = 95
const maxRawImageBytes = 8 << 20
const maxImagePixels = 20_000_000
const maxImageWorkingBytes = 192 << 20
const maxDecodedSourceBytesPerPixel = 8
const imageProcessingLease = 10 * time.Minute

var errProcessedImageTooLarge = errors.New("processed image exceeds size limit")

type processedImage struct {
	data        []byte
	contentType string
	extension   string
}

// processImagePost 是圖文貼文的非同步流程：
// 1. 從 MinIO raw/ 讀原圖。
// 2. 修正 EXIF 方向，必要時縮圖並重新壓縮。
// 3. 把處理後的圖片存到 processed/，更新 DB 為 ready，並通知前端。
func processImagePost(ctx context.Context, payload ImagePostPayload) (resultErr error) {
	if payload.PostID <= 0 || payload.UserID <= 0 || len(payload.RawKeys) == 0 {
		return fmt.Errorf("invalid image task payload")
	}

	processingToken := uuid.NewString()
	claimed, err := claimImagePost(ctx, payload.PostID, processingToken)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	completed := false
	defer func() {
		if completed {
			return
		}
		if releaseErr := releaseImageClaim(ctx, payload.PostID, processingToken); releaseErr != nil {
			resultErr = errors.Join(resultErr, releaseErr)
		}
	}()

	ops := rollback.New()
	var processedKeys []string
	var processedImages []*processedImage

	for _, rawKey := range payload.RawKeys {
		if !isRawImageKey(rawKey) {
			return rollbackImageAttempt(ctx, ops, fmt.Errorf("invalid raw image key %q", rawKey))
		}
		rawObj, err := minioClient.GetObject(ctx, minioBucket, rawKey, minio.GetObjectOptions{})
		if err != nil {
			return rollbackImageAttempt(ctx, ops, fmt.Errorf("get raw object failed key=%s: %w", rawKey, err))
		}

		rawData, err := readRawImage(rawObj)
		if err != nil {
			return rollbackImageAttempt(ctx, ops, fmt.Errorf("read raw object failed: %w", err))
		}

		processed, err := resizeAndCompress(rawData)
		if err != nil {
			return rollbackImageAttempt(ctx, ops, fmt.Errorf("process image failed: %w", err))
		}

		processedKey := processedImageKey(rawKey, processingToken, processed.extension)
		reader := bytes.NewReader(processed.data)
		_, err = minioClient.PutObject(ctx, minioBucket, processedKey, reader, int64(len(processed.data)),
			minio.PutObjectOptions{ContentType: processed.contentType})
		if err != nil {
			return rollbackImageAttempt(ctx, ops, fmt.Errorf("put processed object failed: %w", err))
		}

		processedKeys = append(processedKeys, processedKey)
		processedImages = append(processedImages, processed)
		capturedKey := processedKey
		ops.Add("remove processed object "+processedKey, func(cleanupCtx context.Context) error {
			return minioClient.RemoveObject(cleanupCtx, minioBucket, capturedKey, minio.RemoveObjectOptions{})
		})
	}

	processedJSON, _ := json.Marshal(processedKeys)
	storedBytes := processedImageBytes(processedImages)
	tag, err := systemPool.Exec(ctx,
		`UPDATE posts
		 SET image_url = $1,
		     image_storage_bytes = $2,
		     image_reserved_bytes = 0,
		     image_status = 'ready',
		     processing_token = NULL,
		     processing_started_at = NULL
		 WHERE id = $3
		   AND image_status = 'processing'
		   AND processing_token = $4`,
		string(processedJSON), storedBytes, payload.PostID, processingToken,
	)
	if err != nil {
		return rollbackImageAttempt(ctx, ops, fmt.Errorf("update post image state failed: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return rollbackImageAttempt(ctx, ops, nil)
	}
	completed = true

	for _, rawKey := range payload.RawKeys {
		if err := minioClient.RemoveObject(ctx, minioBucket, rawKey, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("[image] remove raw object failed, ignored: %v", err)
		}
	}

	rdb.Del(ctx, contracts.FeedCacheKey)

	notifyChannel := contracts.NotifyUserChannel(payload.UserID)
	notifyMsg := fmt.Sprintf(`{"type":"post_ready","post_id":%d}`, payload.PostID)
	rdb.Publish(ctx, notifyChannel, notifyMsg)

	log.Printf("[image] post #%d processed, files=%d", payload.PostID, len(processedKeys))
	return nil
}

func claimImagePost(ctx context.Context, postID int, token string) (bool, error) {
	expiredBefore := time.Now().UTC().Add(-imageProcessingLease)
	tag, err := systemPool.Exec(ctx,
		`UPDATE posts
		 SET processing_token = $1,
		     processing_started_at = NOW()
		 WHERE id = $2
		   AND image_status = 'processing'
		   AND (
		     processing_token IS NULL
		     OR processing_started_at IS NULL
		     OR processing_started_at < $3
		   )`,
		token, postID, expiredBefore,
	)
	if err != nil {
		return false, fmt.Errorf("claim image post failed: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func releaseImageClaim(ctx context.Context, postID int, token string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	_, err := systemPool.Exec(cleanupCtx,
		`UPDATE posts
		 SET processing_token = NULL,
		     processing_started_at = NULL
		 WHERE id = $1
		   AND image_status = 'processing'
		   AND processing_token = $2`,
		postID, token,
	)
	if err != nil {
		return fmt.Errorf("release image processing claim failed: %w", err)
	}
	return nil
}

func processedImageKey(rawKey, processingToken, extension string) string {
	name := strings.TrimPrefix(rawKey, contracts.RawImagePrefix)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return contracts.ProcessedImagePrefix + base + "-" + processingToken + extension
}

func rollbackImageAttempt(ctx context.Context, operations *rollback.Manager, cause error) error {
	if rollbackErr := operations.Execute(ctx); rollbackErr != nil {
		return errors.Join(cause, fmt.Errorf("rollback image attempt failed: %w", rollbackErr))
	}
	return cause
}

func markPostFailed(ctx context.Context, postID int) error {
	_, err := systemPool.Exec(ctx,
		`UPDATE posts
		 SET image_status = 'failed',
		     image_reserved_bytes = 0,
		     image_storage_bytes = 0,
		     processing_token = NULL,
		     processing_started_at = NULL
		 WHERE id = $1
		   AND image_status = 'processing'`,
		postID,
	)
	return err
}

func finalizeFailedImagePost(ctx context.Context, payload ImagePostPayload) error {
	if err := deleteImages(ctx, ImageDeletePayload{Keys: payload.RawKeys}); err != nil {
		return fmt.Errorf("remove raw images after terminal failure: %w", err)
	}
	if err := markPostFailed(ctx, payload.PostID); err != nil {
		return fmt.Errorf("mark image post failed: %w", err)
	}
	return nil
}

func isRawImageKey(key string) bool {
	if !strings.HasPrefix(key, contracts.RawImagePrefix) {
		return false
	}
	name := strings.TrimPrefix(key, contracts.RawImagePrefix)
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.ContainsAny(name, "\\\x00") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}

func isProcessedImageKey(key string) bool {
	if !strings.HasPrefix(key, contracts.ProcessedImagePrefix) {
		return false
	}
	name := strings.TrimPrefix(key, contracts.ProcessedImagePrefix)
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.ContainsAny(name, "\\\x00") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}

func deleteImages(ctx context.Context, payload ImageDeletePayload) error {
	if len(payload.Keys) == 0 {
		return nil
	}
	for _, key := range payload.Keys {
		if !isRawImageKey(key) && !isProcessedImageKey(key) {
			return fmt.Errorf("invalid image cleanup key %q", key)
		}
		if err := minioClient.RemoveObject(ctx, minioBucket, key, minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("remove image %s failed: %w", key, err)
		}
	}
	return nil
}

func readRawImage(rawObj io.ReadCloser) ([]byte, error) {
	defer rawObj.Close()

	rawData, err := io.ReadAll(io.LimitReader(rawObj, maxRawImageBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(rawData)) > maxRawImageBytes {
		return nil, fmt.Errorf("raw image exceeds %d bytes", maxRawImageBytes)
	}
	return rawData, nil
}

// resizeAndCompress 保留原圖格式：PNG 繼續輸出 PNG，其餘格式輸出 JPEG。
// 小於 maxDimension 且方向正常時直接回傳原始 bytes，避免不必要的畫質損失。
func resizeAndCompress(data []byte) (*processedImage, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image config failed: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, fmt.Errorf("invalid image dimensions")
	}
	if int64(config.Width) > maxImagePixels/int64(config.Height) {
		return nil, fmt.Errorf("image dimensions exceed %d pixels", maxImagePixels)
	}

	orientation := imageOrientation(data)
	newW, newH := resizedDimensions(config.Width, config.Height)
	workingBytes := estimatedImageWorkingBytes(config.Width, config.Height, newW, newH, orientation) + int64(len(data))
	if workingBytes > maxImageWorkingBytes {
		return nil, fmt.Errorf("image working set %d exceeds %d bytes", workingBytes, maxImageWorkingBytes)
	}

	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image failed, format=%s: %w", format, err)
	}
	contentType, extension := imageOutputInfo(format)

	output := src
	bounds := src.Bounds()
	if bounds.Dx() != newW || bounds.Dy() != newH {
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		output = dst
	}
	output = applyOrientation(output, orientation)

	buf := &limitedImageBuffer{limit: contracts.MaxProcessedImageBytes}
	if format == "png" {
		if err := png.Encode(buf, output); err != nil {
			return nil, fmt.Errorf("encode PNG failed: %w", err)
		}
		return &processedImage{
			data:        buf.Bytes(),
			contentType: contentType,
			extension:   extension,
		}, nil
	}

	if err := jpeg.Encode(buf, output, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("encode JPEG failed: %w", err)
	}

	return &processedImage{
		data:        buf.Bytes(),
		contentType: contentType,
		extension:   extension,
	}, nil
}

func resizedDimensions(width, height int) (int, int) {
	if width <= maxDimension && height <= maxDimension {
		return width, height
	}
	if width >= height {
		return maxDimension, max(1, height*maxDimension/width)
	}
	return max(1, width*maxDimension/height), maxDimension
}

func estimatedImageWorkingBytes(width, height, resizedWidth, resizedHeight, orientation int) int64 {
	// PNG may decode to an 8-byte-per-pixel NRGBA64 image. Use that as the
	// conservative source cost even when JPEG and 8-bit PNG use less memory.
	sourceBytes := int64(width) * int64(height) * maxDecodedSourceBytesPerPixel
	resizedBytes := int64(resizedWidth) * int64(resizedHeight) * 4
	total := sourceBytes + contracts.MaxProcessedImageBytes
	if resizedWidth != width || resizedHeight != height {
		total += resizedBytes
	}
	if orientation != 1 {
		total += resizedBytes
	}
	return total
}

func processedImageBytes(images []*processedImage) int64 {
	var total int64
	for _, image := range images {
		if image != nil {
			total += int64(len(image.data))
		}
	}
	return total
}

type limitedImageBuffer struct {
	bytes.Buffer
	limit int
}

func (buffer *limitedImageBuffer) Write(data []byte) (int, error) {
	if len(data) > buffer.limit-buffer.Len() {
		return 0, errProcessedImageTooLarge
	}
	return buffer.Buffer.Write(data)
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
