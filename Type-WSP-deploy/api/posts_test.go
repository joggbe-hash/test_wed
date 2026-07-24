package main

import (
	"bytes"
	"testing"
)

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
