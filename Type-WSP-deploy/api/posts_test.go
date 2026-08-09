package main

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"typewsp/shared/contracts"
)

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
