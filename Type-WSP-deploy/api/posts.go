package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
)

// Post 代表一則貼文的完整資料結構
type Post struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Username    string    `json:"username"`
	Content     string    `json:"content,omitempty"`
	ImageURLs   []string  `json:"image_urls,omitempty"` // JSON 陣列，多張圖片
	ImageStatus string    `json:"image_status"`
	CreatedAt   time.Time `json:"created_at"`
}

const feedCacheKey = "feed:latest"
const feedCacheTTL = 30 * time.Second

// handleFeed 處理 GET /api/feed
// 回傳最新 20 筆貼文，支援 cursor 分頁
func handleFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cursor := r.URL.Query().Get("cursor")

	// 僅在無 cursor（首頁載入）時使用快取
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
		writeJSON(w, http.StatusInternalServerError, M{"error": "查詢失敗"})
		return
	}
	defer rows.Close()

	posts := make([]Post, 0, 20)
	for rows.Next() {
		var p Post
		var imgURLRaw, imgStatus *string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content,
			&imgURLRaw, &imgStatus, &p.CreatedAt); err != nil {
			log.Printf("掃描貼文列失敗: %v", err)
			continue
		}
		if imgStatus != nil {
			p.ImageStatus = *imgStatus
		}
		// image_url 欄位儲存 JSON 陣列，解析後加上代理路徑前綴
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

// handleCreatePost 處理 POST /api/posts
// 純文字 → API 直接寫 DB
// 有圖片 → 先寫 DB (status=processing) 讓前端立即看到，再丟 Worker 處理圖片
func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	ctx := r.Context()

	// 嘗試解析 multipart form（上限 100MB）
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		// 非 multipart → JSON 純文字貼文
		var body struct {
			Content string `json:"content"`
		}
		if err := readJSON(r, &body); err != nil || body.Content == "" {
			writeJSON(w, http.StatusBadRequest, M{"error": "content 為必填欄位"})
			return
		}
		createTextPost(ctx, w, user, body.Content)
		return
	}

	content := r.FormValue("content")

	// 取得所有上傳的圖片檔案（欄位名稱 "images"）
	var files []*fileEntry
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		for _, fh := range r.MultipartForm.File["images"] {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			files = append(files, &fileEntry{reader: f, size: fh.Size})
		}
	}

	if len(files) == 0 {
		if content == "" {
			writeJSON(w, http.StatusBadRequest, M{"error": "content 為必填欄位"})
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
	reader io.ReadCloser
	size   int64
}

// createTextPost 純文字貼文，直接寫 DB
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
		writeJSON(w, http.StatusInternalServerError, M{"error": "發文失敗"})
		return
	}

	rdb.Del(ctx, feedCacheKey)
	writeJSON(w, http.StatusCreated, M{"message": "發文成功", "post_id": postID})
}

// createImagePost 帶圖貼文
// 流程：上傳原圖到 MinIO → 先寫 DB（status=processing）→ 丟任務給 Worker
// 這樣前端刷新 feed 馬上能看到「圖片處理中」的貼文，不會消失
func createImagePost(ctx context.Context, w http.ResponseWriter, user *User, content string, files []*fileEntry) {
	ops := NewAtomicRollback()

	// 步驟 1：逐一上傳原始圖片到 MinIO
	var rawKeys []string
	for _, f := range files {
		rawKey := fmt.Sprintf("raw/%s.jpg", uuid.New().String())
		_, err := minioClient.PutObject(ctx, minioBucket, rawKey, f.reader, f.size,
			minio.PutObjectOptions{ContentType: "image/jpeg"})
		if err != nil {
			ops.Execute()
			writeJSON(w, http.StatusInternalServerError, M{"error": "圖片上傳失敗"})
			return
		}
		rawKeys = append(rawKeys, rawKey)
		capturedKey := rawKey
		ops.Add("刪除 MinIO 原始圖片 "+rawKey, func() error {
			return minioClient.RemoveObject(ctx, minioBucket, capturedKey, minio.RemoveObjectOptions{})
		})
	}

	// 步驟 2：先寫入 DB，狀態為 processing，前端立即可見
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
		writeJSON(w, http.StatusInternalServerError, M{"error": "發文失敗"})
		return
	}
	ops.Add("刪除 DB 貼文記錄", func() error {
		_, e := systemPool.Exec(ctx, "DELETE FROM posts WHERE id = $1", postID)
		return e
	})

	// 步驟 3：丟任務給 Worker 做圖片壓縮處理
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
		writeJSON(w, http.StatusInternalServerError, M{"error": "任務排程失敗"})
		return
	}

	rdb.Del(ctx, feedCacheKey)
	writeJSON(w, http.StatusCreated, M{"message": "發文成功，圖片處理中", "post_id": postID})
}

// handleGetImage 從 MinIO 讀取圖片並串流回傳給客戶端
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
