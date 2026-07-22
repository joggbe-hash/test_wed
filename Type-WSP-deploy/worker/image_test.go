package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"testing"
)

func TestReadRawImageAllowsLimitedData(t *testing.T) {
	data := []byte("small image payload")
	got, err := readRawImage(io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		t.Fatalf("readRawImage returned error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("readRawImage = %q", got)
	}
}

func TestReadRawImageRejectsOversizedData(t *testing.T) {
	data := bytes.Repeat([]byte("a"), maxRawImageBytes+1)
	if _, err := readRawImage(io.NopCloser(bytes.NewReader(data))); err == nil {
		t.Fatal("expected oversized raw image to be rejected")
	}
}

func TestResizeAndCompressAlwaysReencodesImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{G: 255, A: 255})
	src.Set(0, 1, color.RGBA{B: 255, A: 255})
	src.Set(1, 1, color.RGBA{R: 255, G: 255, A: 255})

	var original bytes.Buffer
	if err := jpeg.Encode(&original, src, &jpeg.Options{Quality: jpegQuality}); err != nil {
		t.Fatalf("encode test JPEG: %v", err)
	}

	metadataMarker := []byte("GPS-EXIF-SECRET")
	exifPayload := append([]byte("Exif\x00\x00"), metadataMarker...)
	exifLength := len(exifPayload) + 2
	input := []byte{0xff, 0xd8, 0xff, 0xe1, byte(exifLength >> 8), byte(exifLength)}
	input = append(input, exifPayload...)
	input = append(input, original.Bytes()[2:]...)
	processed, err := resizeAndCompress(input)
	if err != nil {
		t.Fatalf("resizeAndCompress returned error: %v", err)
	}
	if processed.contentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", processed.contentType)
	}
	if bytes.Contains(processed.data, metadataMarker) {
		t.Fatal("processed image retained metadata appended to the source image")
	}
	if _, _, err := image.Decode(bytes.NewReader(processed.data)); err != nil {
		t.Fatalf("processed image is not decodable: %v", err)
	}
}
