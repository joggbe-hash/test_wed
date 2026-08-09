package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"typewsp/shared/contracts"
	"typewsp/shared/rollback"
)

// Post 是 feed API 回給前端的貼文格式；image_urls 會是可直接載入的 API URL。
type Post struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Username    string    `json:"username"`
	Visibility  string    `json:"visibility"`
	Content     string    `json:"content,omitempty"`
	ImageURLs   []string  `json:"image_urls,omitempty"`
	ImageStatus string    `json:"image_status"`
	CreatedAt   time.Time `json:"created_at"`
}

const (
	maxPostBodyBytes                        = 25 << 20
	maxPostContentRunes                     = 5000
	maxMultipartMemory                      = 8 << 20
	maxImageUploadSize                      = 8 << 20
	maxImagesPerPost                        = 4
	maxPendingImagePostsPerUser             = 4
	maxPostsPerUser                         = 5000
	maxImageStorageBytesPerUser       int64 = 1 << 30
	contentQuotaAdvisoryLockNamespace       = 0x54575350
)

var (
	errImagePostQuotaExceeded    = errors.New("too many pending image posts")
	errImageStorageQuotaExceeded = errors.New("image storage quota exceeded")
	errPostQuotaExceeded         = errors.New("post quota exceeded")
)

type postVisibility string

const (
	postVisibilityPublic  postVisibility = "public"
	postVisibilityPrivate postVisibility = "private"
)

func parsePostVisibility(raw string) (postVisibility, bool) {
	switch postVisibility(strings.TrimSpace(raw)) {
	case "", postVisibilityPublic:
		return postVisibilityPublic, true
	case postVisibilityPrivate:
		return postVisibilityPrivate, true
	default:
		return "", false
	}
}

type feedCursor struct {
	CreatedAt time.Time
	ID        int
}

func encodeFeedCursor(createdAt time.Time, id int) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.Itoa(id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeFeedCursor(raw string) (feedCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return feedCursor{}, fmt.Errorf("decode cursor failed: %w", err)
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return feedCursor{}, fmt.Errorf("invalid cursor format")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return feedCursor{}, fmt.Errorf("invalid cursor time: %w", err)
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return feedCursor{}, fmt.Errorf("invalid cursor id")
	}

	return feedCursor{CreatedAt: createdAt, ID: id}, nil
}

func validPostContent(content string) bool {
	return utf8.ValidString(content) && utf8.RuneCountInString(content) <= maxPostContentRunes
}

func postListQuery(onlyOwner, hasCursor bool) string {
	ownerFilter := "(visibility = 'public' OR user_id = $1)"
	if onlyOwner {
		ownerFilter = "user_id = $1"
	}

	cursorFilter := ""
	if hasCursor {
		cursorFilter = " AND (created_at < $2 OR (created_at = $2 AND id < $3))"
	}

	return `SELECT id, user_id, username, visibility, content, image_url, image_status, created_at
		 FROM posts
		 WHERE ` + ownerFilter + cursorFilter + `
		 ORDER BY created_at DESC, id DESC LIMIT 20`
}

// handleFeed reads public posts plus private posts owned by the authenticated user.
// The response is intentionally not shared through a global cache because visibility is user-specific.
func handleFeed(w http.ResponseWriter, r *http.Request) {
	handlePostList(w, r, false)
}

// handlePersonalPosts only returns posts owned by the authenticated user.
func handlePersonalPosts(w http.ResponseWriter, r *http.Request) {
	handlePostList(w, r, true)
}

func handlePostList(w http.ResponseWriter, r *http.Request, onlyOwner bool) {
	ctx := r.Context()
	user := currentUser(r)
	cursor := r.URL.Query().Get("cursor")

	var rows pgx.Rows
	var err error
	if cursor != "" {
		parsedCursor, parseErr := decodeFeedCursor(cursor)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, M{"error": "invalid feed cursor"})
			return
		}
		rows, err = systemPool.Query(ctx,
			postListQuery(onlyOwner, true), user.ID, parsedCursor.CreatedAt, parsedCursor.ID)
	} else {
		rows, err = systemPool.Query(ctx, postListQuery(onlyOwner, false), user.ID)
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
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Visibility, &p.Content,
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
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "load feed failed"})
		return
	}

	var nextCursor string
	if len(posts) == 20 {
		lastPost := posts[len(posts)-1]
		nextCursor = encodeFeedCursor(lastPost.CreatedAt, lastPost.ID)
	}

	result := M{"posts": posts, "next_cursor": nextCursor}

	writeJSON(w, http.StatusOK, result)
}

// handleCreatePost 同時支援 JSON 純文字貼文，以及 multipart 圖文貼文。
// 圖片先存 raw/ 到 MinIO，DB 標記 processing，再交給 worker 壓縮與轉存 processed/。
func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, maxPostBodyBytes)

	if !isMultipartRequest(r) {
		var body struct {
			Content    string `json:"content"`
			Visibility string `json:"visibility"`
		}
		if err := readJSON(r, &body); err != nil {
			if isMaxBytesError(err) {
				writeJSON(w, http.StatusRequestEntityTooLarge, M{"error": "post body is too large"})
				return
			}
			writeJSON(w, http.StatusBadRequest, M{"error": "invalid JSON body"})
			return
		}
		body.Content = strings.TrimSpace(body.Content)
		if body.Content == "" {
			writeJSON(w, http.StatusBadRequest, M{"error": "content is required"})
			return
		}
		if !validPostContent(body.Content) {
			writeJSON(w, http.StatusBadRequest, M{"error": "post content must be at most 5000 characters"})
			return
		}
		visibility, valid := parsePostVisibility(body.Visibility)
		if !valid {
			writeJSON(w, http.StatusBadRequest, M{"error": "invalid post visibility"})
			return
		}
		createTextPost(ctx, w, user, body.Content, visibility)
		return
	}

	files, uploadErr := parseImageFiles(r)
	if uploadErr != nil {
		writeJSON(w, uploadErr.status, M{"error": uploadErr.message})
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	defer closeFileEntries(files)

	content := strings.TrimSpace(r.FormValue("content"))
	if !validPostContent(content) {
		writeJSON(w, http.StatusBadRequest, M{"error": "post content must be at most 5000 characters"})
		return
	}
	visibility, valid := parsePostVisibility(r.FormValue("visibility"))
	if !valid {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid post visibility"})
		return
	}

	if len(files) == 0 {
		if content == "" {
			writeJSON(w, http.StatusBadRequest, M{"error": "content is required"})
			return
		}
		createTextPost(ctx, w, user, content, visibility)
		return
	}

	createImagePost(ctx, w, user, content, visibility, files)
}

func handleDeletePost(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	postID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || postID <= 0 {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid post id"})
		return
	}

	ctx := r.Context()
	var imageURLRaw *string
	err = WithTx(ctx, systemPool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			"SELECT image_url FROM posts WHERE id = $1 AND user_id = $2 FOR UPDATE",
			postID, user.ID,
		).Scan(&imageURLRaw); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, "DELETE FROM posts WHERE id = $1 AND user_id = $2", postID, user.ID)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, M{"error": "post not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "delete post failed"})
		return
	}

	imageKeys := managedImageKeys(imageURLRaw)
	if len(imageKeys) > 0 {
		if err := enqueueTask(ctx, contracts.TaskDeleteImages, M{"keys": imageKeys}); err != nil {
			log.Printf("enqueue image cleanup failed for post %d: %v", postID, err)
			removeImagesBestEffort(ctx, imageKeys)
		}
	}

	writeJSON(w, http.StatusOK, M{"message": "post deleted"})
}

func managedImageKeys(raw *string) []string {
	if raw == nil || *raw == "" {
		return nil
	}
	var keys []string
	if err := json.Unmarshal([]byte(*raw), &keys); err != nil {
		return nil
	}
	validKeys := keys[:0]
	for _, key := range keys {
		if isManagedImageKey(key) {
			validKeys = append(validKeys, key)
		}
	}
	return validKeys
}

func isManagedImageKey(key string) bool {
	if isProcessedImageKey(key) {
		return true
	}
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

func removeImagesBestEffort(ctx context.Context, keys []string) {
	for _, key := range keys {
		if err := minioClient.RemoveObject(ctx, minioBucket, key, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("remove image object failed key=%s: %v", key, err)
		}
	}
}

type fileEntry struct {
	reader      io.ReadCloser
	size        int64
	contentType string
	extension   string
}

type uploadValidationError struct {
	status  int
	message string
}

func isMultipartRequest(r *http.Request) bool {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	return strings.HasPrefix(contentType, "multipart/form-data")
}

func parseImageFiles(r *http.Request) ([]*fileEntry, *uploadValidationError) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		if isMaxBytesError(err) {
			return nil, &uploadValidationError{status: http.StatusRequestEntityTooLarge, message: "post body is too large"}
		}
		return nil, &uploadValidationError{status: http.StatusBadRequest, message: "invalid multipart form"}
	}

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, nil
	}

	headers := r.MultipartForm.File["images"]
	if len(headers) > maxImagesPerPost {
		return nil, &uploadValidationError{status: http.StatusBadRequest, message: "too many images"}
	}

	files := make([]*fileEntry, 0, len(headers))
	for _, fh := range headers {
		if fh.Size <= 0 {
			closeFileEntries(files)
			return nil, &uploadValidationError{status: http.StatusBadRequest, message: "empty image file"}
		}
		if fh.Size > maxImageUploadSize {
			closeFileEntries(files)
			return nil, &uploadValidationError{status: http.StatusRequestEntityTooLarge, message: "image file is too large"}
		}

		f, err := fh.Open()
		if err != nil {
			closeFileEntries(files)
			return nil, &uploadValidationError{status: http.StatusBadRequest, message: "invalid image file"}
		}

		contentType, extension, err := imageFileInfo(f, fh.Filename)
		if err != nil {
			f.Close()
			closeFileEntries(files)
			return nil, &uploadValidationError{status: http.StatusBadRequest, message: "unsupported image file"}
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			f.Close()
			closeFileEntries(files)
			return nil, &uploadValidationError{status: http.StatusBadRequest, message: "invalid image file"}
		}

		files = append(files, &fileEntry{
			reader:      f,
			size:        fh.Size,
			contentType: contentType,
			extension:   extension,
		})
	}

	return files, nil
}

func closeFileEntries(files []*fileEntry) {
	for _, f := range files {
		f.reader.Close()
	}
}

func imageFileInfo(file io.ReadSeeker, filename string) (string, string, error) {
	var header [512]byte
	n, err := io.ReadFull(file, header[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", "", fmt.Errorf("read image header failed: %w", err)
	}
	if n == 0 {
		return "", "", fmt.Errorf("empty image file")
	}

	extension := strings.ToLower(filepath.Ext(filename))
	contentType := http.DetectContentType(header[:n])
	switch contentType {
	case "image/jpeg":
		return "image/jpeg", ".jpg", nil
	case "image/png":
		return "image/png", ".png", nil
	default:
		if extension == ".jpg" || extension == ".jpeg" || extension == ".png" {
			return "", "", fmt.Errorf("image extension does not match content")
		}
		return "", "", fmt.Errorf("unsupported image type %q", contentType)
	}
}

func isMaxBytesError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func isProcessedImageKey(key string) bool {
	if !strings.HasPrefix(key, contracts.ProcessedImagePrefix) {
		return false
	}
	if strings.Contains(key, "..") || strings.ContainsAny(key, "\\\x00") {
		return false
	}

	name := strings.TrimPrefix(key, contracts.ProcessedImagePrefix)
	if name == "" || strings.Contains(name, "/") {
		return false
	}

	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}

func createTextPost(ctx context.Context, w http.ResponseWriter, user *User, content string, visibility postVisibility) {
	var postID int
	err := WithTx(ctx, systemPool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1::integer, $2::integer)", contentQuotaAdvisoryLockNamespace, user.ID); err != nil {
			return err
		}
		var total int
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM posts WHERE user_id = $1", user.ID).Scan(&total); err != nil {
			return err
		}
		if !hasPostCapacity(total) {
			return errPostQuotaExceeded
		}
		return tx.QueryRow(ctx,
			`INSERT INTO posts (user_id, username, visibility, content, image_status)
			 VALUES ($1, $2, $3, $4, 'none') RETURNING id`,
			user.ID, user.Username, visibility, content,
		).Scan(&postID)
	})
	if err != nil {
		if errors.Is(err, errPostQuotaExceeded) {
			writeJSON(w, http.StatusTooManyRequests, M{"error": "post quota exceeded"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "create post failed"})
		return
	}

	writeJSON(w, http.StatusCreated, M{"message": "post created", "post_id": postID})
}

func createImagePost(ctx context.Context, w http.ResponseWriter, user *User, content string, visibility postVisibility, files []*fileEntry) {
	var total int
	var pending int
	var storedAndReserved int64
	reservation := imageStorageReservation(len(files))
	if err := systemPool.QueryRow(ctx,
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE image_status = 'processing'),
		        COALESCE(SUM(image_reserved_bytes + image_storage_bytes), 0)
		 FROM posts WHERE user_id = $1`,
		user.ID,
	).Scan(&total, &pending, &storedAndReserved); err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "check image queue capacity failed"})
		return
	}
	if !hasPostCapacity(total) {
		writeJSON(w, http.StatusTooManyRequests, M{"error": "post quota exceeded"})
		return
	}
	if !hasImagePostCapacity(pending) {
		writeJSON(w, http.StatusTooManyRequests, M{"error": "too many images are still processing"})
		return
	}
	if !hasImageStorageCapacity(storedAndReserved, reservation) {
		writeJSON(w, http.StatusTooManyRequests, M{"error": "image storage quota exceeded"})
		return
	}

	ops := rollback.New()

	var rawKeys []string
	for _, f := range files {
		rawKey := fmt.Sprintf("%s%s%s", contracts.RawImagePrefix, uuid.New().String(), f.extension)
		_, err := minioClient.PutObject(ctx, minioBucket, rawKey, f.reader, f.size,
			minio.PutObjectOptions{ContentType: f.contentType})
		if err != nil {
			if rollbackErr := ops.Execute(ctx); rollbackErr != nil {
				log.Printf("rollback image upload failed: %v", rollbackErr)
			}
			writeJSON(w, http.StatusInternalServerError, M{"error": "upload image failed"})
			return
		}
		rawKeys = append(rawKeys, rawKey)
		capturedKey := rawKey
		ops.Add("remove raw object "+rawKey, func(cleanupCtx context.Context) error {
			return minioClient.RemoveObject(cleanupCtx, minioBucket, capturedKey, minio.RemoveObjectOptions{})
		})
	}

	rawKeysJSON, _ := json.Marshal(rawKeys)
	var postID int
	err := WithTx(ctx, systemPool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1::integer, $2::integer)", contentQuotaAdvisoryLockNamespace, user.ID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*),
			        COUNT(*) FILTER (WHERE image_status = 'processing'),
			        COALESCE(SUM(image_reserved_bytes + image_storage_bytes), 0)
			 FROM posts WHERE user_id = $1`,
			user.ID,
		).Scan(&total, &pending, &storedAndReserved); err != nil {
			return err
		}
		if !hasPostCapacity(total) {
			return errPostQuotaExceeded
		}
		if !hasImagePostCapacity(pending) {
			return errImagePostQuotaExceeded
		}
		if !hasImageStorageCapacity(storedAndReserved, reservation) {
			return errImageStorageQuotaExceeded
		}
		return tx.QueryRow(ctx,
			`INSERT INTO posts (user_id, username, visibility, content, image_url, image_status, image_reserved_bytes)
			 VALUES ($1, $2, $3, $4, $5, 'processing', $6) RETURNING id`,
			user.ID, user.Username, visibility, content, string(rawKeysJSON), reservation,
		).Scan(&postID)
	})
	if err != nil {
		if rollbackErr := ops.Execute(ctx); rollbackErr != nil {
			log.Printf("rollback image post creation failed: %v", rollbackErr)
		}
		if errors.Is(err, errImagePostQuotaExceeded) {
			writeJSON(w, http.StatusTooManyRequests, M{"error": "too many images are still processing"})
			return
		}
		if errors.Is(err, errPostQuotaExceeded) {
			writeJSON(w, http.StatusTooManyRequests, M{"error": "post quota exceeded"})
			return
		}
		if errors.Is(err, errImageStorageQuotaExceeded) {
			writeJSON(w, http.StatusTooManyRequests, M{"error": "image storage quota exceeded"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "create post failed"})
		return
	}
	ops.Add("delete post row", func(cleanupCtx context.Context) error {
		_, e := systemPool.Exec(cleanupCtx, "DELETE FROM posts WHERE id = $1", postID)
		return e
	})

	if err := enqueueTask(ctx, contracts.TaskProcessImagePost, M{
		"post_id":  postID,
		"user_id":  user.ID,
		"raw_keys": rawKeys,
	}); err != nil {
		if rollbackErr := ops.Execute(ctx); rollbackErr != nil {
			log.Printf("rollback queued image post failed: %v", rollbackErr)
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "enqueue image job failed"})
		return
	}

	writeJSON(w, http.StatusCreated, M{"message": "post created, image processing", "post_id": postID})
}

func hasImagePostCapacity(pending int) bool {
	return pending >= 0 && pending < maxPendingImagePostsPerUser
}

func hasPostCapacity(total int) bool {
	return total >= 0 && total < maxPostsPerUser
}

func imageStorageReservation(imageCount int) int64 {
	if imageCount <= 0 {
		return 0
	}
	return int64(imageCount) * int64(contracts.MaxProcessedImageBytes)
}

func hasImageStorageCapacity(currentBytes, reservationBytes int64) bool {
	return currentBytes >= 0 && reservationBytes > 0 && reservationBytes <= maxImageStorageBytesPerUser &&
		currentBytes <= maxImageStorageBytesPerUser-reservationBytes
}

// handleGetImage 從 MinIO 讀圖並由 API 回傳，前端不用知道物件儲存內部位址。
func handleGetImage(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	key := r.PathValue("key")
	if !isProcessedImageKey(key) {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	var authorized bool
	err := systemPool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM posts
			WHERE image_url::jsonb @> jsonb_build_array($1::text)
			  AND (visibility = 'public' OR user_id = $2)
		)`,
		key, user.ID,
	).Scan(&authorized)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "authorize image access failed"})
		return
	}
	if !authorized {
		http.NotFound(w, r)
		return
	}

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
	setAuthorizedImageCachePolicy(w)
	io.Copy(w, obj)
}

func setAuthorizedImageCachePolicy(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
}
