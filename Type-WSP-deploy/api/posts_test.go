package main

import (
	"bytes"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"typewsp/shared/contracts"
)

func TestDeletePostPersistsCleanupBeforeRemovingQuotaAccounting(t *testing.T) {
	source, err := os.ReadFile("posts.go")
	if err != nil {
		t.Fatalf("read posts.go: %v", err)
	}
	handlerStart := bytes.Index(source, []byte("func handleDeletePost("))
	if handlerStart < 0 {
		t.Fatal("handleDeletePost source was not found")
	}
	handlerEnd := bytes.Index(source[handlerStart:], []byte("\nfunc managedImageKeys("))
	if handlerEnd < 0 {
		t.Fatal("handleDeletePost end was not found")
	}
	handler := source[handlerStart : handlerStart+handlerEnd]
	outboxAt := bytes.Index(handler, []byte("INSERT INTO image_deletion_jobs"))
	deleteAt := bytes.Index(handler, []byte("DELETE FROM posts"))
	if outboxAt < 0 || deleteAt < 0 || outboxAt > deleteAt {
		t.Fatal("post accounting can be deleted before its durable image cleanup job is recorded")
	}
}

func TestImageStorageQuotaIncludesPendingDeletionBytes(t *testing.T) {
	source, err := os.ReadFile("image_upload_reservations.go")
	if err != nil {
		t.Fatalf("read image_upload_reservations.go: %v", err)
	}
	if !bytes.Contains(source, []byte("SUM(reserved_bytes) FROM image_deletion_jobs WHERE user_id = $1")) {
		t.Fatal("image quota releases bytes before durable object deletion completes")
	}
}

func TestPersonalPostsQueryIsOwnerScoped(t *testing.T) {
	query := postListQuery(true, false)
	if !strings.Contains(query, "WHERE user_id = $1") {
		t.Fatalf("personal posts query is not owner scoped: %s", query)
	}
	if strings.Contains(query, "visibility = 'public'") {
		t.Fatalf("personal posts query includes other users' public posts: %s", query)
	}
}

func TestPersonalPostsCursorQueryRemainsOwnerScoped(t *testing.T) {
	query := postListQuery(true, true)
	if !strings.Contains(query, "WHERE user_id = $1 AND (created_at < $2") {
		t.Fatalf("paginated personal posts query is not owner scoped: %s", query)
	}
}

func TestFeedCursorAppliesToPublicAndOwnedPosts(t *testing.T) {
	query := postListQuery(false, true)
	if !strings.Contains(query, "WHERE (visibility = 'public' OR user_id = $1) AND (created_at < $2") {
		t.Fatalf("feed cursor does not apply to the complete visibility filter: %s", query)
	}
}

func TestPrivateImageResponsesCannotBeReusedAcrossSessions(t *testing.T) {
	response := httptest.NewRecorder()
	setAuthorizedImageCachePolicy(response)

	if got := response.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control = %q, want private, no-store", got)
	}
}

func TestIsProcessedImageKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "processed jpg", key: "processed/image.jpg", want: true},
		{name: "processed png", key: "processed/image.png", want: true},
		{name: "raw image", key: "raw/image.jpg", want: false},
		{name: "nested path", key: "processed/user/image.jpg", want: false},
		{name: "traversal", key: "processed/../secret.jpg", want: false},
		{name: "unsupported extension", key: "processed/image.txt", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isProcessedImageKey(tt.key); got != tt.want {
				t.Fatalf("isProcessedImageKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestImageFileInfoUsesMagicBytes(t *testing.T) {
	pngData := make([]byte, 512)
	copy(pngData, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})

	contentType, extension, err := imageFileInfo(bytes.NewReader(pngData), "upload.jpg")
	if err != nil {
		t.Fatalf("imageFileInfo returned error: %v", err)
	}
	if contentType != "image/png" || extension != ".png" {
		t.Fatalf("imageFileInfo = %q, %q", contentType, extension)
	}
}

func TestImageFileInfoRejectsNonImage(t *testing.T) {
	if _, _, err := imageFileInfo(bytes.NewReader([]byte("not an image")), "upload.png"); err == nil {
		t.Fatal("expected non-image upload to be rejected")
	}
}

func TestParsePostVisibilityDefaultsSafelyAndRejectsUnknownValues(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want postVisibility
		ok   bool
	}{
		{raw: "", want: postVisibilityPublic, ok: true},
		{raw: " public ", want: postVisibilityPublic, ok: true},
		{raw: "private", want: postVisibilityPrivate, ok: true},
		{raw: "friends", ok: false},
	} {
		got, ok := parsePostVisibility(tt.raw)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("parsePostVisibility(%q) = %q, %v; want %q, %v", tt.raw, got, ok, tt.want, tt.ok)
		}
	}
}

func TestHasImagePostCapacityBoundsPerUserWork(t *testing.T) {
	if maxPendingImagePostsPerUser != 1 {
		t.Fatalf("maxPendingImagePostsPerUser = %d, want 1 so one user cannot occupy both workers", maxPendingImagePostsPerUser)
	}
	if !hasImagePostCapacity(maxPendingImagePostsPerUser - 1) {
		t.Fatal("legitimate upload below the per-user limit was rejected")
	}
	if hasImagePostCapacity(maxPendingImagePostsPerUser) {
		t.Fatal("upload at the per-user processing limit was accepted")
	}
	if hasImagePostCapacity(-1) {
		t.Fatal("invalid pending count was accepted")
	}
}

func TestImageProcessingBudgetBoundsCumulativePerUserPixels(t *testing.T) {
	fullPostPixels := int64(maxImagesPerPost) * int64(contracts.MaxImagePixels)
	if maxImageProcessingPixelsPerUserWindow < fullPostPixels {
		t.Fatal("processing budget does not preserve one maximum-size image post")
	}
	if !hasImageProcessingCapacity(0, fullPostPixels) {
		t.Fatal("one maximum-size image post was rejected from an empty processing budget")
	}
	if hasImageProcessingCapacity(maxImageProcessingPixelsPerUserWindow-fullPostPixels+1, fullPostPixels) {
		t.Fatal("processing work beyond the rolling per-user pixel budget was accepted")
	}
	if hasImageProcessingCapacity(0, 0) || hasImageProcessingCapacity(-1, 1) {
		t.Fatal("invalid processing pixel counts were accepted")
	}
}

func TestImagePixelCountMatchesWorkerLimit(t *testing.T) {
	pixels, ok := validatedImagePixelCount(4000, 3000)
	if !ok || pixels != 12_000_000 {
		t.Fatalf("validatedImagePixelCount(4000, 3000) = %d, %v", pixels, ok)
	}
	if _, ok := validatedImagePixelCount(int(contracts.MaxImagePixels), 2); ok {
		t.Fatal("image dimensions beyond the shared worker pixel limit were accepted")
	}
	if _, ok := validatedImagePixelCount(0, 100); ok {
		t.Fatal("zero-width image was accepted")
	}
}

func TestImageUploadLimiterBoundsConcurrentMultipartParsing(t *testing.T) {
	limiter := newImageUploadLimiter(maxConcurrentImageUploads)
	for index := 0; index < maxConcurrentImageUploads; index++ {
		if !limiter.tryAcquire() {
			t.Fatalf("upload slot %d was rejected below the concurrency limit", index+1)
		}
	}
	if limiter.tryAcquire() {
		t.Fatal("upload above the concurrent multipart parsing limit was accepted")
	}
	limiter.release()
	if !limiter.tryAcquire() {
		t.Fatal("released upload capacity was not reusable")
	}
}

func TestImageUploadLimitsFitAPITemporaryFilesystem(t *testing.T) {
	const apiTemporaryFilesystemBytes = 64 << 20
	worstCaseSpillPerRequest := int64(maxPostBodyBytes - maxMultipartMemory)
	worstCaseConcurrentSpill := int64(maxConcurrentImageUploads) * worstCaseSpillPerRequest
	if worstCaseConcurrentSpill > apiTemporaryFilesystemBytes {
		t.Fatalf("worst-case concurrent spill = %d, exceeds API tmpfs = %d", worstCaseConcurrentSpill, apiTemporaryFilesystemBytes)
	}
	if maxPostBodyBytes != 13<<20 || maxImageUploadSize != 3<<20 || maxImagesPerPost != 4 {
		t.Fatalf("unexpected upload limits: body=%d image=%d count=%d", maxPostBodyBytes, maxImageUploadSize, maxImagesPerPost)
	}
}

func TestDeploymentUploadLimitsMatchAPIPolicy(t *testing.T) {
	compose, err := os.ReadFile("../docker-compose.yaml")
	if err != nil {
		t.Fatalf("read docker-compose.yaml: %v", err)
	}
	if !bytes.Contains(compose, []byte("/tmp:rw,noexec,nosuid,size=64m")) {
		t.Fatal("API tmpfs is not configured to the tested 64 MiB capacity")
	}

	for _, path := range []string{
		"../nginx/templates/conf.d/default.conf.template",
		"../nginx/templates/conf.d/default.dev.conf.template",
		"../nginx/templates/conf/server.conf.template",
	} {
		template, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !bytes.Contains(template, []byte("client_max_body_size 13m")) {
			t.Fatalf("%s does not enforce the API 13 MiB body limit", path)
		}
	}
}

func TestCreatePostAcquiresUploadSlotBeforeMultipartParsing(t *testing.T) {
	source, err := os.ReadFile("posts.go")
	if err != nil {
		t.Fatalf("read posts.go: %v", err)
	}
	handlerStart := bytes.Index(source, []byte("func handleCreatePost("))
	handlerEnd := bytes.Index(source[handlerStart:], []byte("\nfunc handleDeletePost("))
	if handlerStart < 0 || handlerEnd < 0 {
		t.Fatal("handleCreatePost source was not found")
	}
	handler := source[handlerStart : handlerStart+handlerEnd]
	acquireAt := bytes.Index(handler, []byte("imageUploadConcurrency.tryAcquire()"))
	parseAt := bytes.Index(handler, []byte("parseImageFiles(r)"))
	if acquireAt < 0 || parseAt < 0 || acquireAt > parseAt {
		t.Fatal("multipart parsing is not protected by the upload concurrency limiter")
	}
}

func TestImageStorageReservationUsesWorstCaseProcessedSize(t *testing.T) {
	want := int64(3 * contracts.MaxProcessedImageBytes)
	if got := imageStorageReservation(3); got != want {
		t.Fatalf("imageStorageReservation(3) = %d, want %d", got, want)
	}
}

func TestImageStorageCapacityIncludesReadyAndReservedBytes(t *testing.T) {
	reservation := imageStorageReservation(1)
	if !hasImageStorageCapacity(maxImageStorageBytesPerUser-reservation, reservation) {
		t.Fatal("upload exactly at the storage quota was rejected")
	}
	if hasImageStorageCapacity(maxImageStorageBytesPerUser-reservation+1, reservation) {
		t.Fatal("upload beyond the cumulative storage quota was accepted")
	}
	if hasImageStorageCapacity(0, maxImageStorageBytesPerUser+1) {
		t.Fatal("reservation larger than the total storage quota was accepted")
	}
}

func TestPostCapacityBoundsCumulativePerUserContent(t *testing.T) {
	if !hasPostCapacity(maxPostsPerUser - 1) {
		t.Fatal("post below the cumulative per-user limit was rejected")
	}
	if hasPostCapacity(maxPostsPerUser) {
		t.Fatal("post at the cumulative per-user limit was accepted")
	}
	if hasPostCapacity(-1) {
		t.Fatal("invalid post count was accepted")
	}
}

func TestCreateImagePostReservesCapacityBeforeAllocatingObjectStorage(t *testing.T) {
	source, err := os.ReadFile("posts.go")
	if err != nil {
		t.Fatalf("read posts.go: %v", err)
	}
	functionStart := bytes.Index(source, []byte("func createImagePost("))
	if functionStart < 0 {
		t.Fatal("createImagePost start was not found")
	}
	functionEnd := bytes.Index(source[functionStart:], []byte("\nfunc hasImagePostCapacity("))
	if functionEnd < 0 {
		t.Fatal("createImagePost end was not found")
	}
	functionSource := source[functionStart : functionStart+functionEnd]
	reserveAt := bytes.Index(functionSource, []byte("reserveImageUpload("))
	allocateAt := bytes.Index(functionSource, []byte("minioClient.PutObject"))
	finalizeAt := bytes.Index(functionSource, []byte("finalizeImageUpload("))
	if reserveAt < 0 || allocateAt < 0 || finalizeAt < 0 {
		t.Fatal("createImagePost must reserve quota, allocate object storage, and finalize the reservation")
	}
	if reserveAt > allocateAt {
		t.Fatal("object storage was allocated before the request held a quota reservation")
	}
	if finalizeAt < allocateAt {
		t.Fatal("the upload reservation was finalized before object storage completed")
	}
}
