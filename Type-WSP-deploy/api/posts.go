package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
)

// Post 是 feed API 回給前端的貼文格式；image_urls 會是可直接載入的 API URL。
type Post struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Username    string    `json:"username"`
	Content     string    `json:"content,omitempty"`
	ImageURLs   []string  `json:"image_urls,omitempty"`
	ImageStatus string    `json:"image_status"`
	CreatedAt   time.Time `json:"created_at"`
}

const feedCacheKey = "feed:latest"
const feedCacheTTL = 30 * time.Second

// handleFeed 讀取最新 20 筆貼文。第一頁使用 Redis 快取，cursor 分頁直接查 DB。
func handleFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cursor := r.URL.Query().Get("cursor")

	if cursor == "" {
		if cached, err := rdb.Get(ctx, feedCacheKey).Bytes(); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(cached)
			return
		}
	}

	var rows pgx.Rows
	var err error
	if cursor != "" {
		rows, err = systemPool.Query(ctx,
			`SELECT id, user_id, username, content, image_url, image_status, created_at
			 FROM posts WHERE created_at < $1
			 ORDER BY created_at DESC LIMIT 20`, cursor)
	} else {
		rows, err = systemPool.Query(ctx,
			`SELECT id, user_id, username, content, image_url, image_status, created_at
			 FROM posts ORDER BY created_at DESC LIMIT 20`)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "load feed failed"})
		return
	}
	defer rows.Close()

	posts := make([]Post, 0, 20)
	for rows.Next() {
		var p Post
		var imgURLRaw, imgStatus *string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content,
			&imgURLRaw, &imgStatus, &p.CreatedAt); err != nil {
			log.Printf("scan post failed: %v", err)
			continue
		}
		if imgStatus != nil {
			p.ImageStatus = *imgStatus
		}
		if imgURLRaw != nil && *imgURLRaw != "" {
			var keys []string
			if err := json.Unmarshal([]byte(*imgURLRaw), &keys); err == nil {
				for _, k := range keys {
					p.ImageURLs = append(p.ImageURLs, "/api/images/"+k)
				}
			}
		}
		posts = append(posts, p)
	}

	var nextCursor string
	if len(posts) == 20 {
		nextCursor = posts[len(posts)-1].CreatedAt.Format(time.RFC3339Nano)
	}

	result := M{"posts": posts, "next_cursor": nextCursor}

	if cursor == "" {
		if data, err := json.Marshal(result); err == nil {
			rdb.Set(ctx, feedCacheKey, data, feedCacheTTL)
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// handleCreatePost 同時支援 JSON 純文字貼文，以及 multipart 圖文貼文。
// 圖片先存 raw/ 到 MinIO，DB 標記 processing，再交給 worker 壓縮與轉存 processed/。
func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	ctx := r.Context()

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		var body struct {
			Content string `json:"content"`
		}
		if err := readJSON(r, &body); err != nil || body.Content == "" {
			writeJSON(w, http.StatusBadRequest, M{"error": "content is required"})
			return
		}
		createTextPost(ctx, w, user, body.Content)
		return
	}

	content := r.FormValue("content")

	var files []*fileEntry
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		for _, fh := range r.MultipartForm.File["images"] {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			contentType, extension := imageFileInfo(fh.Header.Get("Content-Type"), fh.Filename)
			files = append(files, &fileEntry{
				reader:      f,
				size:        fh.Size,
				contentType: contentType,
				extension:   extension,
			})
		}
	}

	if len(files) == 0 {
		if content == "" {
			writeJSON(w, http.StatusBadRequest, M{"error": "content is required"})
			return
		}
		createTextPost(ctx, w, user, content)
		return
	}
	defer func() {
		for _, f := range files {
			f.reader.Close()
		}
	}()

	createImagePost(ctx, w, user, content, files)
}

func handleDeletePost(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	postID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || postID <= 0 {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid post id"})
		return
	}

	ctx := r.Context()
	tag, err := systemPool.Exec(ctx, "DELETE FROM posts WHERE id = $1 AND user_id = $2", postID, user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "delete post failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, M{"error": "post not found"})
		return
	}

	rdb.Del(ctx, feedCacheKey)
	writeJSON(w, http.StatusOK, M{"message": "post deleted"})
}

type fileEntry struct {
	reader      io.ReadCloser
	size        int64
	contentType string
	extension   string
}

func imageFileInfo(fhContentType, filename string) (string, string) {
	contentType := strings.ToLower(strings.TrimSpace(fhContentType))
	extension := strings.ToLower(filepath.Ext(filename))

	if contentType == "" {
		contentType = mime.TypeByExtension(extension)
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}

	switch contentType {
	case "image/jpeg", "image/jpg":
		return "image/jpeg", ".jpg"
	case "image/png":
		return "image/png", ".png"
	default:
		if extension == ".png" {
			return "image/png", ".png"
		}
		return "image/jpeg", ".jpg"
	}
}

func createTextPost(ctx context.Context, w http.ResponseWriter, user *User, content string) {
	var postID int
	err := WithTx(ctx, systemPool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO posts (user_id, username, content, image_status)
			 VALUES ($1, $2, $3, 'none') RETURNING id`,
			user.ID, user.Username, content,
		).Scan(&postID)
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "create post failed"})
		return
	}

	rdb.Del(ctx, feedCacheKey)
	writeJSON(w, http.StatusCreated, M{"message": "post created", "post_id": postID})
}

func createImagePost(ctx context.Context, w http.ResponseWriter, user *User, content string, files []*fileEntry) {
	ops := NewAtomicRollback()

	var rawKeys []string
	for _, f := range files {
		rawKey := fmt.Sprintf("raw/%s%s", uuid.New().String(), f.extension)
		_, err := minioClient.PutObject(ctx, minioBucket, rawKey, f.reader, f.size,
			minio.PutObjectOptions{ContentType: f.contentType})
		if err != nil {
			ops.Execute()
			writeJSON(w, http.StatusInternalServerError, M{"error": "upload image failed"})
			return
		}
		rawKeys = append(rawKeys, rawKey)
		capturedKey := rawKey
		ops.Add("remove raw object "+rawKey, func() error {
			return minioClient.RemoveObject(ctx, minioBucket, capturedKey, minio.RemoveObjectOptions{})
		})
	}

	rawKeysJSON, _ := json.Marshal(rawKeys)
	var postID int
	err := WithTx(ctx, systemPool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO posts (user_id, username, content, image_url, image_status)
			 VALUES ($1, $2, $3, $4, 'processing') RETURNING id`,
			user.ID, user.Username, content, string(rawKeysJSON),
		).Scan(&postID)
	})
	if err != nil {
		ops.Execute()
		writeJSON(w, http.StatusInternalServerError, M{"error": "create post failed"})
		return
	}
	ops.Add("delete post row", func() error {
		_, e := systemPool.Exec(ctx, "DELETE FROM posts WHERE id = $1", postID)
		return e
	})

	job, _ := json.Marshal(M{
		"type": "process_image_post",
		"payload": M{
			"post_id":  postID,
			"user_id":  user.ID,
			"raw_keys": rawKeys,
		},
	})
	if err := rdb.RPush(ctx, "task_queue", job).Err(); err != nil {
		ops.Execute()
		writeJSON(w, http.StatusInternalServerError, M{"error": "enqueue image job failed"})
		return
	}

	rdb.Del(ctx, feedCacheKey)
	writeJSON(w, http.StatusCreated, M{"message": "post created, image processing", "post_id": postID})
}

// handleGetImage 從 MinIO 讀圖並由 API 回傳，前端不用知道物件儲存內部位址。
func handleGetImage(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	obj, err := minioClient.GetObject(ctx, minioBucket, key, minio.GetObjectOptions{})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer obj.Close()

	info, err := obj.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.Copy(w, obj)
}
